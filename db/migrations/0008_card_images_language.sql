-- 0008_card_images_language.sql
-- card_images (0007) was keyed by (set_code, number) alone, which only
-- has room for one image per card -- fine while everything was English,
-- but scripts/fetch_card_images.py can now pull other languages too
-- (e.g. PBL_001_R_FR.png), and those need their own row rather than
-- overwriting the English one at the same set+number.
ALTER TABLE card_images
    ADD COLUMN IF NOT EXISTS language TEXT NOT NULL DEFAULT 'EN';

ALTER TABLE card_images
    DROP CONSTRAINT IF EXISTS card_images_pkey;

ALTER TABLE card_images
    ADD PRIMARY KEY (set_code, number, language);

-- The hover-preview feature (api/card-images) only ever wants the
-- English row for a given set+number today -- this index keeps that
-- lookup (WHERE (set_code, number) IN (...) AND language = 'EN')
-- as cheap as the old two-column primary key was.
CREATE INDEX IF NOT EXISTS idx_card_images_set_number
    ON card_images (set_code, number);
