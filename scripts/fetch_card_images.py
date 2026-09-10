#!/usr/bin/env python3
"""Download card art from limitlesstcg.com for a batch of sets.

Reads a sets file (default: sets.txt) describing which sizes to download
and which sets/card-counts to fetch:

    xl,md
    pbl,84
    cri,86
    por,88

The first line is the comma-separated list of sizes to download (one or
more of XL, LG, MD, SM, XS -- XL is the standard, no-suffix image, e.g.
PBL_081_R_EN.png; anything else gets that suffix appended, e.g.
PBL_081_R_EN_MD.png). Every line after that is "<set>,<card count>" --
card numbers 1..<card count> are fetched for that set.

For each card, fetches https://limitlesstcg.com/cards/<SET>/<N>, finds
the regular (non alt-art) card's <img>, and downloads its `data-src` CDN
URL (https://limitlesstcg.nyc3.cdn.digitaloceanspaces.com/tpci/PBL/
PBL_001_R_EN.png) -- once per requested size -- to
cards/<SET>/<LANGUAGE>/.

With --write-db, each card also gets one row upserted into Postgres'
card_images table (see db/migrations/0007_card_images.sql and
0008_card_images_language.sql). image_url is stored as a path relative
to our own R2 bucket (e.g. "PBL/PBL_080_R_EN.png"), not the upstream
limitlesstcg CDN URL -- the frontend prefixes it with the bucket's base
URL (lib/card-images.ts), same pattern as Pokémon icons (lib/icons.ts).
That table has no size column, so if multiple sizes are requested, only
one canonical size's path is stored: XL if it was requested, otherwise
the first size listed in the sets file. The other sizes still get
downloaded to disk, just not recorded in the DB.

This script does not upload anything to the R2 bucket itself -- it only
downloads locally and records the path the file will live at once
uploaded (matching fetch_pokemon_icons.py's separate --upload-to-r2
step, not implemented here).

Usage:
    python scripts/fetch_card_images.py
    python scripts/fetch_card_images.py --sets-file my-sets.txt
    python scripts/fetch_card_images.py --language FR
    python scripts/fetch_card_images.py --write-db

Requires: requests, beautifulsoup4, and (only if using --write-db) psycopg2-binary
    pip install -r scripts/requirements.txt
"""

from __future__ import annotations

import argparse
import os
import re
import sys
import time
from dataclasses import dataclass, field
from pathlib import Path

import requests
from bs4 import BeautifulSoup

BASE_URL = "https://limitlesstcg.com/cards"

# Matches the DATABASE_URL shape docker-compose.yml gives the Go backend,
# but pointed at localhost:5432 -- this script runs on the host, outside
# the Docker network, against Postgres' host-mapped port.
DEFAULT_DATABASE_URL = "postgres://app:devpassword@localhost:5432/pokemontcg?sslmode=disable"

# Sites frequently block the default python-requests UA outright.
HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 "
        "(KHTML, like Gecko) Chrome/124.0 Safari/537.36"
    )
}

# Regular (non alt-art/special) card images ship with exactly these three
# classes on the page -- see the sample <img> in the migration/PR
# description. Alt-art/special prints live at a different card number
# entirely, so this exact match is enough to only ever pick up the base
# card -- no separate "include variants" mode needed.
REGULAR_CARD_CLASSES = {"card", "shadow", "resp-w"}

# XL is the standard, no-suffix filename (PBL_081_R_EN.png); every other
# size appends its own suffix (PBL_081_R_EN_MD.png).
VALID_SIZES = ("XL", "LG", "MD", "SM", "XS")

# Parses e.g. "PBL_001_R_EN.png" or "PBL_001_R_EN_MD.png".
FILENAME_RE = re.compile(
    r"^(?P<set>[A-Z0-9]+)_(?P<number>\d+)_(?P<rarity>[A-Z]+)_(?P<lang>[A-Z]+)"
    r"(?:_(?P<size>LG|MD|SM|XS))?\.(?P<ext>png|jpg|jpeg|webp)$"
)


@dataclass
class RunStats:
    downloaded: int = 0
    skipped_existing: int = 0
    no_image_found: list[tuple[str, int]] = field(default_factory=list)
    failed: list[tuple[str, int]] = field(default_factory=list)
    db_upserted: int = 0
    db_failed: int = 0


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Download card art for a batch of sets from limitlesstcg.com",
    )
    parser.add_argument(
        "--sets-file",
        default="scripts/sets.txt",
        help="Path to the sets file (default: sets.txt). First line is the "
        "comma-separated sizes to download; each following line is "
        "'<set>,<card count>'.",
    )
    parser.add_argument(
        "--language",
        default="EN",
        help="Language to download (default: EN). The page itself is always "
        "served in English -- other languages live at the same set/number "
        "under a differently-suffixed filename (e.g. PBL_001_R_FR.png for "
        "French), so this rewrites the filename rather than requesting a "
        "different URL. If a card wasn't printed in the requested language, "
        "the download will simply 404 and get logged as a failure.",
    )
    parser.add_argument(
        "--output-dir",
        default="cards",
        help="Root output directory (default: cards). Files land in "
        "<output-dir>/<SET>/<LANGUAGE>/.",
    )
    parser.add_argument(
        "--delay",
        type=float,
        default=0.75,
        help="Seconds to wait between page requests (default: 0.75). "
        "Keep this polite -- we're hitting someone else's site once per card.",
    )
    parser.add_argument(
        "--write-db",
        action="store_true",
        help="Also upsert one row per card into Postgres' card_images "
        "table (requires psycopg2-binary and migrations 0007+0008 applied).",
    )
    parser.add_argument(
        "--database-url",
        default=os.environ.get("DATABASE_URL", DEFAULT_DATABASE_URL),
        help="Postgres connection string, only used with --write-db "
        "(default: $DATABASE_URL, falling back to the docker-compose "
        "dev default on localhost:5432).",
    )
    return parser.parse_args(argv)


def parse_sets_file(path: Path) -> tuple[list[str], list[tuple[str, int]]]:
    lines = [line.strip() for line in path.read_text().splitlines() if line.strip()]
    if not lines:
        raise ValueError(f"{path} is empty")

    sizes = [chunk.strip().upper() for chunk in lines[0].split(",") if chunk.strip()]
    unknown = [s for s in sizes if s not in VALID_SIZES]
    if unknown:
        raise ValueError(
            f"unknown size(s) on the first line of {path}: {unknown} -- valid: {VALID_SIZES}"
        )
    if not sizes:
        raise ValueError(f"first line of {path} must list at least one size")

    sets: list[tuple[str, int]] = []
    for lineno, line in enumerate(lines[1:], start=2):
        set_code, _, cards = line.partition(",")
        set_code = set_code.strip().upper()
        cards = cards.strip()
        if not set_code or not cards.isdigit():
            raise ValueError(f"{path}:{lineno}: expected '<set>,<card count>', got {line!r}")
        sets.append((set_code, int(cards)))

    if not sets:
        raise ValueError(f"{path} lists no sets after the sizes line")

    return sizes, sets


def build_page_url(set_code: str, number: int) -> str:
    return f"{BASE_URL}/{set_code}/{number}"


def is_regular_card_img(img) -> bool:
    return set(img.get("class", [])) == REGULAR_CARD_CLASSES


def find_card_image_urls(html: str) -> list[str]:
    soup = BeautifulSoup(html, "html.parser")
    urls = []
    for img in soup.find_all("img"):
        if not is_regular_card_img(img):
            continue
        data_src = img.get("data-src")
        if data_src:
            urls.append(data_src)
    return urls


def build_download_url(data_src_url: str, language: str, size: str) -> str:
    """data-src is always the standard, always-English filename (e.g.
    PBL_001_R_EN.png) -- other languages live at the same path with just
    the language segment swapped (e.g. PBL_001_R_FR.png for French), and
    other sizes get their suffix appended (PBL_001_R_EN_MD.png). Rebuild
    the filename from its parsed parts rather than a plain string-replace,
    so we can't accidentally clobber an unrelated "EN" substring elsewhere
    in the path."""
    directory, _, filename = data_src_url.rpartition("/")
    match = FILENAME_RE.match(filename)
    if not match:
        raise ValueError(f"unrecognized filename format: {filename}")

    new_filename = f"{match.group('set')}_{match.group('number')}_{match.group('rarity')}_{language}"
    if size != "XL":
        new_filename += f"_{size}"
    new_filename += f".{match.group('ext')}"

    return f"{directory}/{new_filename}"


def normalized_number(raw_number: str) -> str:
    """CDN filenames zero-pad the card number (PBL_001_...), but the app's
    own data doesn't -- decklist/archetype cards come from Limitless's
    tournament API with unpadded numbers ("1", "129", "194": see
    limitless/decklist_test.go). Strip leading zeros here so a row written
    by this script actually matches what the hover-preview's (set, number)
    lookup queries with -- storing "001" verbatim would silently never
    join against anything."""
    return str(int(raw_number))


def connect_db(database_url: str):
    try:
        import psycopg2
    except ImportError as exc:
        raise RuntimeError(
            "psycopg2-binary is required for --write-db: pip install -r scripts/requirements.txt"
        ) from exc
    return psycopg2.connect(database_url)


def upsert_card_image(
    conn,
    set_code: str,
    number: str,
    language: str,
    image_url: str,
) -> None:
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO card_images (set_code, number, language, image_url, source)
            VALUES (%s, %s, %s, %s, 'limitless')
            ON CONFLICT (set_code, number, language)
            DO UPDATE SET
                image_url = EXCLUDED.image_url,
                source = EXCLUDED.source,
                resolved_at = now()
            """,
            (set_code, number, language, image_url),
        )
    conn.commit()


def bucket_relative_path(set_code: str, filename: str) -> str:
    """What actually gets stored in card_images.image_url: a path relative
    to our own R2 bucket (e.g. "PBL/PBL_080_R_EN.png"), not the upstream
    limitlesstcg CDN URL -- the frontend prefixes this with its own bucket
    base URL (see lib/card-images.ts), the same way lib/icons.ts does for
    Pokémon icons. This script does not upload the file to that bucket --
    it only downloads locally and records where it *will* live once
    uploaded (matching fetch_pokemon_icons.py's separate --upload-to-r2
    step)."""
    return f"{set_code}/{filename}"


def download(
    session: requests.Session,
    url: str,
    dest: Path,
    stats: RunStats,
) -> None:
    if dest.exists():
        stats.skipped_existing += 1
        print(f"    already have {dest.name}, skipping")
        return

    resp = session.get(url, headers=HEADERS, timeout=30)
    resp.raise_for_status()

    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_bytes(resp.content)
    stats.downloaded += 1
    print(f"    saved {dest}")


def fetch_card(
    session: requests.Session,
    set_code: str,
    number: int,
    sizes: list[str],
    args: argparse.Namespace,
    stats: RunStats,
    db_conn,
    db_size: str,
) -> None:
    padded = f"{number:03d}"
    url = build_page_url(set_code, number)
    print(f"[{set_code} {padded}] {url}")

    try:
        resp = session.get(url, headers=HEADERS, timeout=30)
        resp.raise_for_status()
    except requests.RequestException as exc:
        print(f"    request failed: {exc}", file=sys.stderr)
        stats.failed.append((set_code, number))
        return

    image_urls = find_card_image_urls(resp.text)
    if not image_urls:
        print("    no matching <img> found")
        stats.no_image_found.append((set_code, number))
        return

    target_lang = args.language.upper()
    for data_src in image_urls:
        db_download_url = None

        for size in sizes:
            try:
                download_url = build_download_url(data_src, target_lang, size)
            except ValueError as exc:
                print(f"    {exc}, skipping")
                continue

            if size == db_size:
                db_download_url = download_url

            download_filename = download_url.rsplit("/", 1)[-1]
            dest = (
                Path(args.output_dir)
                / set_code
                / target_lang
                / download_filename
            )

            try:
                download(session, download_url, dest, stats)
            except requests.RequestException as exc:
                print(f"    download failed for {download_url}: {exc}", file=sys.stderr)
                stats.failed.append((set_code, number))
                if size == db_size:
                    db_download_url = None

        if db_conn is not None and db_download_url is not None:
            db_filename = db_download_url.rsplit("/", 1)[-1]
            match = FILENAME_RE.match(db_filename)
            db_number = normalized_number(match.group("number"))
            relative_path = bucket_relative_path(set_code, db_filename)
            try:
                upsert_card_image(
                    db_conn, set_code, db_number, target_lang, relative_path
                )
                stats.db_upserted += 1
            except Exception as exc:  # noqa: BLE001 -- log and keep going
                print(f"    db upsert failed: {exc}", file=sys.stderr)
                db_conn.rollback()
                stats.db_failed += 1


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)

    sets_file = Path(args.sets_file)
    if not sets_file.exists():
        print(f"sets file not found: {sets_file}", file=sys.stderr)
        return 1

    try:
        sizes, sets = parse_sets_file(sets_file)
    except ValueError as exc:
        print(str(exc), file=sys.stderr)
        return 1

    # card_images has no size column -- if multiple sizes are requested,
    # only one is recorded in the DB. Prefer XL (the standard/full-size
    # image) when it's among the requested sizes, else whichever was
    # listed first.
    db_size = "XL" if "XL" in sizes else sizes[0]

    db_conn = None
    if args.write_db:
        try:
            db_conn = connect_db(args.database_url)
        except Exception as exc:
            print(f"could not connect to Postgres: {exc}", file=sys.stderr)
            return 1

    stats = RunStats()
    session = requests.Session()

    try:
        for set_code, card_count in sets:
            print(f"=== {set_code} ({card_count} cards, sizes: {', '.join(sizes)}) ===")
            for number in range(1, card_count + 1):
                fetch_card(session, set_code, number, sizes, args, stats, db_conn, db_size)
                time.sleep(args.delay)
    finally:
        if db_conn is not None:
            db_conn.close()

    print()
    print("=== summary ===")
    print(f"downloaded:            {stats.downloaded}")
    print(f"skipped (already had): {stats.skipped_existing}")
    print(f"no image found:        {len(stats.no_image_found)} {stats.no_image_found}")
    print(f"failed:                {len(stats.failed)} {stats.failed}")
    if args.write_db:
        print(f"db rows upserted:      {stats.db_upserted}")
        print(f"db upserts failed:     {stats.db_failed}")

    return 1 if (stats.failed or stats.db_failed) else 0


if __name__ == "__main__":
    raise SystemExit(main())
