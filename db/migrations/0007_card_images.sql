-- 0007_card_images.sql
-- Card art lookup, backing the decklist/archetype hover-preview feature.
--
-- Keyed by (set_code, number) -- a card's print identity, matching the
-- `set`/`number` fields already stored per-card in decklists.cards and
-- archetypes.core_cards (see models.Card). We don't derive image_url from
-- a plain string template at read time: Limitless's own CDN path also
-- encodes rarity and language (e.g. tpci/SCR/SCR_115_R_EN_MD.png), which
-- isn't on the card record and can't be reliably guessed from set+number
-- alone (reprints, alt arts, and promos vary the rarity segment). So each
-- row's image_url is resolved once and cached here, the same pattern
-- pokemon_icons (0006) uses for icon art -- a future resolver job
-- populates this table, mirroring fetch_pokemon_icons.py.
CREATE TABLE IF NOT EXISTS card_images (
    set_code    TEXT NOT NULL,
    number      TEXT NOT NULL,
    image_url   TEXT NOT NULL,
    source      TEXT NOT NULL DEFAULT 'limitless', -- where image_url was resolved from, in case a fallback source is added later
    resolved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (set_code, number)
);
