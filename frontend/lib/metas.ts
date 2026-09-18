import type { Meta } from "@/lib/types";

// pickDefaultMeta chooses which meta a page should show: the one named
// in the URL if it still exists, otherwise the currently open
// "standard" meta -- the permanent, format-level view the site
// defaults to (see db/migrations/0009_meta_hierarchy.sql on the
// backend) -- otherwise, if this format hasn't been bootstrapped with
// a standard meta yet, the most recently started meta of any kind so
// the page still has something to show.
export function pickDefaultMeta(
    metas: Meta[],
    requestedId?: string | null,
): Meta | null {
    if (requestedId) {
        const requested = metas.find((m) => m.id === requestedId);
        if (requested) return requested;
    }

    const openStandard = metas.find(
        (m) => m.type === "standard" && !m.ends_at,
    );
    if (openStandard) return openStandard;

    return metas[0] ?? null;
}

// groupMetasForSelect splits metas into the permanent standard metas
// (shown first/plain in a <select>) and the set metas nested under
// each one (shown grouped under their parent via <optgroup>), plus any
// set meta whose parent isn't in the list (e.g. a very old era).
export function groupMetasForSelect(metas: Meta[]): {
    standards: Meta[];
    setsByParent: Map<string, Meta[]>;
    orphanSets: Meta[];
} {
    const standards = metas.filter((m) => m.type === "standard");
    const standardIds = new Set(standards.map((m) => m.id));

    const setsByParent = new Map<string, Meta[]>();
    const orphanSets: Meta[] = [];

    for (const m of metas) {
        if (m.type !== "set") continue;
        if (m.parent_meta_id && standardIds.has(m.parent_meta_id)) {
            const list = setsByParent.get(m.parent_meta_id) ?? [];
            list.push(m);
            setsByParent.set(m.parent_meta_id, list);
        } else {
            orphanSets.push(m);
        }
    }

    return { standards, setsByParent, orphanSets };
}
