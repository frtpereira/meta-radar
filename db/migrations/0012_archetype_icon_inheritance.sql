-- db/migrations/0012_archetype_icon_inheritance.sql
--
-- Carries curated archetype icons across set metas.
--
-- Archetypes are scoped per set meta (UNIQUE (meta_id, slug), see
-- 0001_init.sql), so the same deck gets a brand-new archetypes row --
-- with a brand-new id -- the first time ingestion sees it in a newly
-- opened set meta (db/seed/002_open_set_meta.sql). archetype_icons is
-- keyed by that id (0006_pokemon_icons.sql), so the new row starts with
-- no icons even though the old set's row for the same slug has them.
-- The Standard (parent) meta makes it worse: it merges same-slug
-- archetypes and shows the *newest* row's icons (see
-- archetypeStatsStandardQuery), so every archetype that has been seen in
-- the new set loses its icons there too.
--
-- Fix: whenever an archetype has no icons of its own, copy them from the
-- same-slug archetype in the most recent earlier meta of the same format
-- that does have icons.
--
--   * inherit_archetype_icons(id) does the copy for one archetype. It is
--     a no-op when the archetype already has icons (curated icons are
--     never overwritten) or when no same-slug archetype has any.
--   * a trigger calls it for every newly inserted archetype, so future
--     set metas inherit automatically as ingestion creates their rows.
--     (An upsert that hits ON CONFLICT ... DO UPDATE is not an insert
--     and does not fire it -- correct, that row already exists.)
--   * the backfill at the bottom repairs archetypes that already exist.
--     It only touches archetypes without icons, so re-running this file
--     (`make migrate` runs every migration each time) is safe, and picks
--     up icons you curated on an older set's archetype after the newer
--     one was created.
--
-- Not auto-run by docker-entrypoint-initdb.d -- apply with `make migrate`.
-- Written to be safe to run more than once.

CREATE OR REPLACE FUNCTION inherit_archetype_icons(p_archetype_id BIGINT)
RETURNS INT AS $$
DECLARE
    v_source_id BIGINT;
    v_inserted  INT := 0;
BEGIN
    -- Never touch an archetype that already has icons.
    IF EXISTS (SELECT 1 FROM archetype_icons WHERE archetype_id = p_archetype_id) THEN
        RETURN 0;
    END IF;

    -- Same slug, same format, newest meta first, and it must actually
    -- have icons (otherwise a newer icon-less sibling would shadow an
    -- older row that has them).
    SELECT src.id INTO v_source_id
    FROM archetypes tgt
    JOIN metas tm ON tm.id = tgt.meta_id
    JOIN archetypes src ON src.slug = tgt.slug AND src.id <> tgt.id
    JOIN metas sm ON sm.id = src.meta_id AND sm.format_code = tm.format_code
    WHERE tgt.id = p_archetype_id
      AND EXISTS (SELECT 1 FROM archetype_icons ai WHERE ai.archetype_id = src.id)
    ORDER BY sm.starts_at DESC, src.id DESC
    LIMIT 1;

    IF v_source_id IS NULL THEN
        RETURN 0;
    END IF;

    INSERT INTO archetype_icons (archetype_id, pokemon_slug, display_order)
    SELECT p_archetype_id, ai.pokemon_slug, ai.display_order
    FROM archetype_icons ai
    WHERE ai.archetype_id = v_source_id
    ON CONFLICT (archetype_id, pokemon_slug) DO NOTHING;

    GET DIAGNOSTICS v_inserted = ROW_COUNT;
    RETURN v_inserted;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION archetypes_inherit_icons_trigger()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM inherit_archetype_icons(NEW.id);
    RETURN NULL; -- AFTER trigger: return value is ignored
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_archetypes_inherit_icons ON archetypes;
CREATE TRIGGER trg_archetypes_inherit_icons
    AFTER INSERT ON archetypes
    FOR EACH ROW
    EXECUTE FUNCTION archetypes_inherit_icons_trigger();

-- Backfill archetypes that already exist without icons (i.e. everything
-- created in the current set meta before this migration ran). Loops
-- oldest meta first so a chain of metas (A -> B -> C) resolves in order.
DO $$
DECLARE
    v_id     BIGINT;
    v_copied INT := 0;
BEGIN
    FOR v_id IN
        SELECT a.id
        FROM archetypes a
        JOIN metas m ON m.id = a.meta_id
        ORDER BY m.starts_at, a.id
    LOOP
        v_copied := v_copied + inherit_archetype_icons(v_id);
    END LOOP;

    RAISE NOTICE 'archetype icons: copied % icon row(s) onto archetypes that had none', v_copied;
END $$;
