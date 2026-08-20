import Link from "next/link";
import type { ArchetypeStat, Meta } from "@/lib/types";
import { getArchetypeStats, getMetas } from "@/lib/api";
import Hero from "@/components/hero";
import Table from "@/components/table";
import Card from "@/components/card";
import InfoTooltip from "@/components/info-tooltip";

type SearchParams = {
    meta_id?: string;
};

function formatPercent(value: number | null) {
    if (value === null) return "—";
    return `${Math.round(value * 1000) / 10}%`;
}

function formatStanding(value: number | null) {
    if (value === null) return "—";
    return `#${Math.round(value)}`;
}

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

function WinRateBadge({ value }: { value: number | null }) {
    if (value === null) return <span className="muted">—</span>;
    const pct = Math.round(value * 1000) / 10;
    const color =
        pct >= 55
            ? "var(--success)"
            : pct >= 48
              ? "var(--accent)"
              : "var(--accent-2)";
    return <span style={{ color, fontWeight: 600 }}>{pct}%</span>;
}

function ArchetypesTable({
    archetypes,
    metaId,
}: {
    archetypes: ArchetypeStat[];
    metaId: string;
}) {
    const columns = [
        {
            key: "archetype",
            label: "Archetype",
            render: (s: ArchetypeStat) => (
                <Link
                    className="table-link"
                    href={`/decklists/${s.id}?meta_id=${metaId}`}
                >
                    <div className="table-title">{s.name}</div>
                    <div className="muted tiny">{s.slug}</div>
                </Link>
            ),
        },
        {
            key: "decks",
            label: "Decklists",
            render: (s: ArchetypeStat) => s.deck_count.toLocaleString(),
        },
        {
            key: "win_rate",
            label: "Win rate",
            render: (s: ArchetypeStat) => <WinRateBadge value={s.win_rate} />,
        },
        {
            key: "score_rate",
            label: (
                <>
                    Score rate
                    <InfoTooltip text="Share of possible match points earned: (wins + 0.5 × ties) ÷ matches played. Unlike win rate, ties count as half a win instead of being excluded." />
                </>
            ),
            render: (s: ArchetypeStat) => formatPercent(s.score_rate),
        },
        {
            key: "avg_standing",
            label: "Avg standing",
            render: (s: ArchetypeStat) => formatStanding(s.avg_standing),
        },
        {
            key: "record",
            label: "Record",
            render: (s: ArchetypeStat) => `${s.wins}–${s.losses}–${s.ties}`,
        },
        {
            key: "action",
            label: "",
            render: (s: ArchetypeStat) => (
                <Link
                    href={`/decklists/${s.id}?meta_id=${metaId}`}
                    className="button"
                    style={{ whiteSpace: "nowrap" }}
                >
                    View deck →
                </Link>
            ),
        },
    ];

    return <Table columns={columns} rows={archetypes} />;
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
                    eyebrow="Meta Radar — Decklists"
                    title="Archetype Decklists"
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
                            <h2>Meta selection</h2>
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
                    headingMeta={
                        activeMeta ? (
                            <span className="pill">
                                {activeMeta.format_code}
                            </span>
                        ) : null
                    }
                >
                    {archetypes.length > 0 ? (
                        <ArchetypesTable
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
