import type { ArchetypeStat, Meta } from "@/lib/types";
import { getArchetypeStats, getMetas } from "@/lib/api";
import Hero from "@/components/hero";
import Card from "@/components/card";
import ArchetypeSearch from "./ArchetypeSearch";

type SearchParams = {
    meta_id?: string;
};

function EmptyState({ title, copy }: { title: string; copy: string }) {
    return (
        <div className="empty-state">
            <h3>{title}</h3>
            <p>{copy}</p>
        </div>
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
            <button type="submit">Load meta</button>
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
                    title="Deck Archetype Explorer"
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
                        <MetaSelector metas={metas} activeMeta={activeMeta} />
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
