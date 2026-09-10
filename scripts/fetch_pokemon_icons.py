#!/usr/bin/env python3
"""
fetch_pokemon_icons.py

Download Pokémon icons from limitlesstcg.com's public R2 asset bucket so
they can be self-hosted (e.g. synced into your own S3/R2 bucket or CDN)
instead of hotlinked at runtime.

WHY "gen" PROBING:
Limitless serves icons under paths like:
    https://r2.limitlesstcg.net/pokemon/gen9/dragapult.png
    https://r2.limitlesstcg.net/pokemon/gen9/absol-mega.png
The "genN" segment does not reliably correspond to the Pokémon's actual
Pokédex generation (e.g. an old Pokémon can still live under gen9 if that's
the art style used). Rather than hardcoding a name->gen map (which will
break/need maintenance as new Pokémon and forms are added), this script
probes gen9 down to gen1 for each name and keeps the first hit. Results
are cached in a manifest so repeat runs (e.g. a nightly cron picking up
newly-seen archetypes from your ingestion pipeline) don't re-probe names
already resolved.

USAGE:
    # From a newline-separated file of names (one Pokémon per line)
    python fetch_pokemon_icons.py --names names.txt --out ./icons

    # From a comma-separated inline list
    python fetch_pokemon_icons.py --names "Dragapult,Absol-Mega,Ogerpon,Ogerpon-Cornerstone,Zoroark-Hisui" --out ./icons

    # Re-run any time; already-downloaded icons and already-resolved
    # names (per the manifest) are skipped automatically.

NAME FORMAT:
Accepts names either plain ("Dragapult") or with a form suffix using a
space, underscore, or hyphen ("Ogerpon Cornerstone", "Zoroark_Hisui",
"Ogerpon-Cornerstone") — all are normalized to Limitless's hyphenated,
lowercase slug format ("ogerpon-cornerstone").

OUTPUT:
    ./icons/<slug>.png               (one file per resolved Pokémon — kept
                                       locally even after a successful R2
                                       upload; nothing is deleted)
    ./icons/manifest.json            (name -> {slug, gen, url, path, status,
                                       uploaded})
    ./icons/unresolved.txt           (names that failed on every gen, for
                                       manual follow-up — e.g. typos or
                                       genuinely new art not yet uploaded)

UPLOADING TO YOUR OWN R2 BUCKET (optional):
    Pass --upload-to-r2 to push every resolved icon straight to your own
    R2 bucket after downloading, setting a long-lived, immutable
    Cache-Control header on each object (since these files never change
    once fetched). Local files under --out are always kept regardless —
    this is download-then-upload, not download-then-move.

    Requires `pip install boto3` and R2 credentials, supplied via
    environment variables (preferred, so secrets don't end up in shell
    history or process listings) or the matching --r2-* flags:

        R2_ACCOUNT_ID           Cloudflare account ID
        R2_ACCESS_KEY_ID        R2 API token access key ID
        R2_SECRET_ACCESS_KEY    R2 API token secret access key
        R2_BUCKET               destination bucket name

    Example:
        export R2_ACCOUNT_ID=xxxx
        export R2_ACCESS_KEY_ID=xxxx
        export R2_SECRET_ACCESS_KEY=xxxx
        export R2_BUCKET=meta-radar
        python fetch_pokemon_icons.py --names names.txt --out ./icons \\
            --upload-to-r2 --r2-prefix pokemon-icons
"""

import argparse
import json
import mimetypes
import os
import re
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

BASE_URL = "https://r2.limitlesstcg.net/pokemon/gen{gen}/{slug}.png"
GENS_TO_TRY = list(range(9, 0, -1))  # gen9 first (most likely), down to gen1
REQUEST_TIMEOUT = 10
RATE_LIMIT_SECONDS = 0.3  # polite delay between requests
MAX_RETRIES = 3
RETRY_BACKOFF_BASE = 1.5

USER_AGENT = "meta-radar-icon-sync/1.0 (self-host asset sync; contact: frtpereira on GitHub)"

# These icons are immutable once fetched (a re-fetch of the same slug
# returns the same art), so tell every downstream cache — R2, Cloudflare's
# edge, and the visitor's browser — to hold onto them for a year.
R2_CACHE_CONTROL = "public, max-age=31536000, immutable"


def slugify(name: str) -> str:
    """Normalize a Pokémon name into Limitless's URL slug format."""
    slug = name.strip().lower()
    slug = re.sub(r"[ _]+", "-", slug)
    slug = re.sub(r"[^a-z0-9\-]", "", slug)
    slug = re.sub(r"-+", "-", slug).strip("-")
    return slug


def load_names(names_arg: str) -> list[str]:
    """Load names either from a file path or a comma-separated string."""
    path = Path(names_arg)
    text = path.read_text(encoding="utf-8") if path.exists() else names_arg
    # Split on both newlines and commas so either a one-per-line file or a
    # comma-separated file/inline string works.
    raw = re.split(r"[\n,]", text)
    names = [n.strip() for n in raw if n.strip()]
    # de-dupe while preserving order
    seen = set()
    deduped = []
    for n in names:
        if n not in seen:
            seen.add(n)
            deduped.append(n)
    return deduped


def fetch_url(url: str) -> bytes | None:
    """Fetch a URL, returning bytes on 200, None on 404, raising on other errors."""
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    last_err = None
    for attempt in range(1, MAX_RETRIES + 1):
        try:
            with urllib.request.urlopen(req, timeout=REQUEST_TIMEOUT) as resp:
                return resp.read()
        except urllib.error.HTTPError as e:
            if e.code == 404:
                return None
            last_err = e
        except (urllib.error.URLError, TimeoutError) as e:
            last_err = e
        # transient failure: backoff and retry
        sleep_for = RETRY_BACKOFF_BASE ** attempt
        time.sleep(sleep_for)
    print(f"  ! giving up on {url} after {MAX_RETRIES} attempts: {last_err}", file=sys.stderr)
    return None


def resolve_and_download(name: str, out_dir: Path, manifest: dict) -> dict:
    """Resolve a name to a working gen/URL and download it, or reuse manifest."""
    slug = slugify(name)
    dest = out_dir / f"{slug}.png"

    cached = manifest.get(name)
    if cached and cached.get("status") == "ok" and dest.exists():
        return cached  # already resolved and file present, nothing to do

    if cached and cached.get("status") == "ok" and not dest.exists():
        # manifest says we solved it before but file is missing (e.g. wiped
        # icons dir) - redownload from the known-good gen directly.
        gens_to_try = [cached["gen"]] + [g for g in GENS_TO_TRY if g != cached["gen"]]
    else:
        gens_to_try = GENS_TO_TRY

    for gen in gens_to_try:
        url = BASE_URL.format(gen=gen, slug=slug)
        time.sleep(RATE_LIMIT_SECONDS)
        data = fetch_url(url)
        if data is not None:
            dest.write_bytes(data)
            print(f"  ✓ {name} -> gen{gen}/{slug}.png ({len(data)} bytes)")
            return {
                "slug": slug,
                "gen": gen,
                "url": url,
                "path": str(dest),
                "status": "ok",
            }

    print(f"  ✗ {name} (slug: {slug}) not found in any gen1-9 folder")
    return {"slug": slug, "gen": None, "url": None, "path": None, "status": "not_found"}


def build_r2_client(args):
    """Create a boto3 S3-compatible client pointed at the R2 account."""
    try:
        import boto3
    except ImportError:
        print(
            "! --upload-to-r2 requires boto3. Install it with: pip install boto3",
            file=sys.stderr,
        )
        sys.exit(1)

    account_id = args.r2_account_id or os.environ.get("R2_ACCOUNT_ID")
    access_key = args.r2_access_key_id or os.environ.get("R2_ACCESS_KEY_ID")
    secret_key = args.r2_secret_access_key or os.environ.get("R2_SECRET_ACCESS_KEY")

    missing = [
        name
        for name, val in [
            ("account id (--r2-account-id / R2_ACCOUNT_ID)", account_id),
            ("access key (--r2-access-key-id / R2_ACCESS_KEY_ID)", access_key),
            ("secret key (--r2-secret-access-key / R2_SECRET_ACCESS_KEY)", secret_key),
        ]
        if not val
    ]
    if missing:
        print(f"! --upload-to-r2 is missing: {', '.join(missing)}", file=sys.stderr)
        sys.exit(1)

    endpoint_url = f"https://{account_id}.r2.cloudflarestorage.com"
    return boto3.client(
        "s3",
        endpoint_url=endpoint_url,
        aws_access_key_id=access_key,
        aws_secret_access_key=secret_key,
        region_name="auto",
    )


def upload_to_r2(client, bucket: str, local_path: Path, key: str) -> bool:
    """Upload one file to R2 with a long-lived, immutable Cache-Control header."""
    content_type = mimetypes.guess_type(str(local_path))[0] or "application/octet-stream"
    try:
        client.upload_file(
            str(local_path),
            bucket,
            key,
            ExtraArgs={
                "CacheControl": R2_CACHE_CONTROL,
                "ContentType": content_type,
            },
        )
        return True
    except Exception as e:  # noqa: BLE001 - surface any boto3/network error plainly
        print(f"  ! failed to upload {local_path} -> r2://{bucket}/{key}: {e}", file=sys.stderr)
        return False


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--names", required=True, help="Path to a newline-separated names file, or a comma-separated list of names")
    parser.add_argument("--out", default="./icons", help="Output directory for downloaded icons (default: ./icons)")
    parser.add_argument("--upload-to-r2", action="store_true", help="After downloading, upload each resolved icon to your R2 bucket (local files are kept either way)")
    parser.add_argument("--r2-prefix", default="pokemon-icons", help="Key prefix inside the bucket (default: pokemon-icons)")
    parser.add_argument("--r2-account-id", default=None, help="Cloudflare account ID (or set R2_ACCOUNT_ID)")
    parser.add_argument("--r2-access-key-id", default=None, help="R2 API token access key ID (or set R2_ACCESS_KEY_ID)")
    parser.add_argument("--r2-secret-access-key", default=None, help="R2 API token secret access key (or set R2_SECRET_ACCESS_KEY)")
    parser.add_argument("--r2-bucket", default=None, help="Destination bucket name (or set R2_BUCKET)")
    args = parser.parse_args()

    out_dir = Path(args.out)
    out_dir.mkdir(parents=True, exist_ok=True)

    r2_client = None
    r2_bucket = None
    if args.upload_to_r2:
        r2_bucket = args.r2_bucket or os.environ.get("R2_BUCKET")
        if not r2_bucket:
            print("! --upload-to-r2 needs a bucket: pass --r2-bucket or set R2_BUCKET", file=sys.stderr)
            sys.exit(1)
        r2_client = build_r2_client(args)

    manifest_path = out_dir / "manifest.json"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8")) if manifest_path.exists() else {}

    names = load_names(args.names)
    print(f"Resolving {len(names)} name(s) into {out_dir}/ ...\n")

    unresolved = []
    for name in names:
        result = resolve_and_download(name, out_dir, manifest)

        if result["status"] == "ok" and r2_client is not None and not result.get("uploaded"):
            key = f"{args.r2_prefix.rstrip('/')}/{result['slug']}.png"
            ok = upload_to_r2(r2_client, r2_bucket, Path(result["path"]), key)
            result["uploaded"] = ok
            result["r2_key"] = key if ok else None
            if ok:
                print(f"  ↑ uploaded {result['slug']}.png -> r2://{r2_bucket}/{key}")

        manifest[name] = result
        if result["status"] != "ok":
            unresolved.append(name)
        # persist incrementally so a Ctrl+C mid-run doesn't lose progress
        # (local icon files under --out are never deleted, uploaded or not)
        manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True), encoding="utf-8")

    if unresolved:
        (out_dir / "unresolved.txt").write_text("\n".join(unresolved) + "\n", encoding="utf-8")
        print(f"\n{len(unresolved)} name(s) unresolved — see {out_dir}/unresolved.txt")
    else:
        unresolved_path = out_dir / "unresolved.txt"
        if unresolved_path.exists():
            unresolved_path.unlink()
        print("\nAll names resolved.")

    ok_count = sum(1 for v in manifest.values() if v.get("status") == "ok")
    print(f"Manifest: {ok_count}/{len(manifest)} resolved -> {manifest_path}")


if __name__ == "__main__":
    main()