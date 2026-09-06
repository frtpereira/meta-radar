-- db/migrations/0006_pokemon_icons.sql
-- Self-hosted Pokémon icon registry, backing the archetype-icon feature.
-- Icons themselves are pulled from limitlesstcg.com's R2 bucket and
-- re-uploaded to our own R2 bucket by fetch_pokemon_icons.py; this
-- migration only adds the Postgres side of that. Apply with `make migrate`.

-- Inventory of icons we actually have in our own R2 bucket. Slug matches
-- the R2 object's base filename (e.g. 'ogerpon-cornerstone' for
-- pokemon-icons/ogerpon-cornerstone.png). This table is the source of
-- truth for "do we have an icon for this Pokémon" -- populated by
-- fetch_pokemon_icons.py's --upload-to-r2 step, independent of any
-- archetype logic, so it can be reused by other icon consumers later
-- (card detail pages, matchup views, etc.) without touching archetypes.
CREATE TABLE IF NOT EXISTS pokemon_icons (
    slug        TEXT PRIMARY KEY,
    r2_key      TEXT NOT NULL,          -- e.g. 'pokemon-icons/ogerpon-cornerstone.png'
    gen         SMALLINT,               -- which limitlesstcg gen folder it resolved from (debugging aid)
    resolved_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A card's exact printed name (as stored in decklists.cards and
-- archetypes.core_cards, e.g. "Ogerpon Cornerstone Mask ex") doesn't
-- reliably transform into its icon slug ("ogerpon-cornerstone") --
-- suffixes like "ex"/"V"/"VMAX", reordered regional-form names ("Hisuian
-- Zoroark" -> "zoroark-hisui"), and Mega naming all vary case by case.
-- So this mapping is curated/populated deliberately rather than derived
-- with a string transform.
CREATE TABLE IF NOT EXISTS card_pokemon_icons (
    card_name    TEXT PRIMARY KEY,
    pokemon_slug TEXT NOT NULL REFERENCES pokemon_icons(slug),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Which icon(s) represent an archetype, and in what order to display
-- them. A join table rather than a column on archetypes because an
-- archetype's core can include more than one icon-worthy attacker (e.g.
-- dual-attacker builds like "Roaring Moon ex / Ogerpon ex") -- this is a
-- one-to-many relationship, not one-to-one.
CREATE TABLE IF NOT EXISTS archetype_icons (
    archetype_id  BIGINT NOT NULL REFERENCES archetypes(id) ON DELETE CASCADE,
    pokemon_slug  TEXT NOT NULL REFERENCES pokemon_icons(slug),
    display_order SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (archetype_id, pokemon_slug)
);

CREATE INDEX IF NOT EXISTS idx_archetype_icons_archetype
    ON archetype_icons (archetype_id, display_order);
