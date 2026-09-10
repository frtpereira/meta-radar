import type { ArchetypeStat, Meta } from "@/lib/types";
import { getArchetypeStats, getMetas } from "@/lib/api";
import Hero from "@/components/hero";
import Card from "@/components/card";
import FilterForm from "@/components/filter-form";
import ArchetypeSearch from "./ArchetypeSearch";

type SearchParams = {
    meta_id?: string;
    min_matches?: string;
};

function EmptyState({ title, copy }: { title: string; copy: string }) {
    return (
        <div className="empty-state">
            <h3>{title}</h3>
            <p>{copy}</p>
        </div>
    );
}

function ArchetypeFilters({
    metas,
    activeMeta,
    minMatches,
}: {
    metas: Meta[];
    activeMeta: Meta | null;
    minMatches: number;
}) {
    return (
        <FilterForm className="selector selector--stack">
            <div className="selector__field">
                <p className="eyebrow">Meta</p>
                <label className="sr-only" htmlFor="meta_id">
                    Select meta
                </label>
                <select
                    id="meta_id"
                    name="meta_id"
                    defaultValue={activeMeta?.id ?? ""}
                >
                    {metas.map((meta) => (
                        <option key={meta.id} value={meta.id}>
                            {meta.name}
                        </option>
                    ))}
                </select>
            </div>

            <div className="selector__field">
                <p className="eyebrow">Minimum matches</p>
                <label className="sr-only" htmlFor="min_matches">
                    Minimum matches
                </label>
                <input
                    id="min_matches"
                    name="min_matches"
                    type="number"
                    min={1}
                    defaultValue={minMatches}
                />
            </div>

            <div className="selector__field">
                <p
                    className="eyebrow selector__field-spacer"
                    aria-hidden="true"
                >
                    Apply
                </p>
                <button type="submit">Apply</button>
            </div>
        </FilterForm>
    );
}

function MetaSelector({
    metas,
    activeMeta,
}: {
    metas: Meta[];
    activeMeta: Meta | null;
}) {
    return (
        <form className="selector" method="get">
            <div>
                <p className="eyebrow">Meta</p>
                <label className="sr-only" htmlFor="meta_id">
                    Select meta
                </label>
                <select
                    id="meta_id"
                    name="meta_id"
                    defaultValue={activeMeta?.id ?? ""}
                >
                    {metas.map((meta) => (
                        <option key={meta.id} value={meta.id}>
                            {meta.name}
                        </option>
                    ))}
                </select>
            </div>
            <button type="submit">Apply</button>
        </form>
    );
}

export default async function DecklistsPage({
    searchParams,
}: {
    searchParams: Promise<SearchParams>;
}) {
    const params = await searchParams;
    const metas = await getMetas().catch(() => [] as Meta[]);
    const activeMeta =
        metas.find((m) => m.id === params.meta_id) ?? metas[0] ?? null;

    const parsedMinMatches = Number.parseInt(params.min_matches ?? "40", 10);
    const minMatches =
        Number.isFinite(parsedMinMatches) && parsedMinMatches > 0
            ? parsedMinMatches
            : 40;

    const archetypes = activeMeta
        ? await getArchetypeStats(activeMeta.id).catch(
              () => [] as ArchetypeStat[],
          )
        : [];

    return (
        <main className="page">
            <div className="ambient ambient--one" />
            <div className="ambient ambient--two" />

            <div className="shell">
                <Hero
                    eyebrow="Meta Radar — Archetypes"
                    title="Archetype Explorer"
                    lede="Browse every archetype in the meta. Open one to see its
                        deck skeleton, optional tech choices, and head-to-head
                        matchup summary."
                    meta={
                        <>
                            {activeMeta ? (
                                <span className="pill">{activeMeta.name}</span>
                            ) : null}
                            <span className="pill pill--soft">
                                {archetypes.length} archetypes
                            </span>
                        </>
                    }
                />

                <Card
                    heading={
                        <>
                            <p className="eyebrow">Filters</p>
                            <h2>Search Archetypes</h2>
                        </>
                    }
                    headingMeta={
                        <span className="muted">
                            {archetypes.length.toLocaleString()} archetypes
                        </span>
                    }
                >
                    {metas.length > 0 ? (
                        <ArchetypeFilters
                            metas={metas}
                            activeMeta={activeMeta}
                            minMatches={minMatches}
                        />
                    ) : (
                        <EmptyState
                            title="No metas yet"
                            copy="Seed a meta before decklists can be loaded."
                        />
                    )}
                </Card>

                <Card
                    className="section--spaced"
                    heading={
                        <>
                            <p className="eyebrow">Archetypes</p>
                            <h2>
                                {activeMeta
                                    ? activeMeta.name
                                    : "No meta loaded"}
                            </h2>
                        </>
                    }
                >
                    {archetypes.length > 0 ? (
                        <ArchetypeSearch
                            archetypes={archetypes}
                            metaId={activeMeta!.id}
                            minMatches={minMatches}
                        />
                    ) : (
                        <EmptyState
                            title="No archetypes found"
                            copy="Run the clustering pipeline for this meta to populate archetype data."
                        />
                    )}
                </Card>
            </div>
        </main>
    );
}
