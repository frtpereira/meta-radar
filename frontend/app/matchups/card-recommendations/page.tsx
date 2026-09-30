import type { ArchetypeStat, Meta, MatchupCards } from "@/lib/types";

import { getArchetypeStats, getMatchupCards, getMetas } from "@/lib/api";
import { pickDefaultMeta } from "@/lib/metas";
import Hero from "@/components/hero";
import Card from "@/components/card";
import FilterForm from "@/components/filter-form";

type SearchParams = {
  meta_id?: string;
  archetype_id?: string;
  opponent_id?: string;
  min_matches?: string;
};

function percent(value: number) {
  return `${(value * 100).toFixed(1)}%`;
}

function signedPercent(value: number) {
  return `${value >= 0 ? "+" : ""}${(value * 100).toFixed(1)} pts`;
}

function RecommendationList({
  title,
  items,
}: {
  title: string;
  items: MatchupCards["recommendations"];
}) {
  return (
    <Card heading={<h2>{title}</h2>} className="section--spaced">
      {items.length > 0 ? (
        <table>
          <thead>
            <tr>
              <th>Card</th>
              <th>With</th>
              <th>Without</th>
              <th>Delta</th>
            </tr>
          </thead>
          <tbody>
            {items.map((card) => (
              <tr key={card.name}>
                <td>{card.name}</td>
                <td>
                  {percent(card.score_rate_with)} ({card.matches_with})
                </td>
                <td>
                  {percent(card.score_rate_without)} ({card.matches_without})
                </td>
                <td>{signedPercent(card.delta)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : (
        <p className="muted">Not enough data.</p>
      )}
    </Card>
  );
}

export default async function MatchupCardsPage({
  searchParams,
}: {
  searchParams: Promise<SearchParams>;
}) {
  const params = await searchParams;
  const metas = await getMetas().catch(() => [] as Meta[]);
  const activeMeta = pickDefaultMeta(metas, params.meta_id);
  const archetypes = activeMeta
    ? await getArchetypeStats(activeMeta.id).catch(() => [] as ArchetypeStat[])
    : [];

  const parsedMin = Number.parseInt(params.min_matches ?? "3", 10);
  const minMatches =
    Number.isFinite(parsedMin) && parsedMin > 0 ? parsedMin : 3;
  const archetypeId = params.archetype_id ?? "";
  const opponentId = params.opponent_id ?? "";

  let data: MatchupCards | null = null;
  let error: string | null = null;
  if (archetypeId && opponentId) {
    if (archetypeId === opponentId) {
      error = "Pick two different archetypes.";
    } else {
      data = await getMatchupCards(archetypeId, opponentId, minMatches).catch(
        (e: unknown) => {
          error = e instanceof Error ? e.message : "Request failed";
          return null;
        },
      );
    }
  }

  const include = (data?.recommendations ?? [])
    .filter((c) => c.delta > 0)
    .slice(0, 10);
  const cut = (data?.recommendations ?? [])
    .filter((c) => c.delta < 0)
    .slice(-10)
    .reverse();

  return (
    <main className="page">
      <div className="shell">
        <Hero
          eyebrow="Meta Radar - Matchups"
          title="Matchup Card Recommendations"
          lede="See which cards perform best, and worst, for an archetype against a specific opponent, based on recorded pairings."
        />

        <Card heading={<h2>Choose a matchup</h2>}>
          <FilterForm className="selector selector--stack">
            <input type="hidden" name="meta_id" value={activeMeta?.id ?? ""} />
            <label htmlFor="archetype_id">Archetype</label>
            <select
              id="archetype_id"
              name="archetype_id"
              defaultValue={archetypeId}
            >
              <option value="">Select archetype</option>
              {archetypes.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
            <label htmlFor="opponent_id">Opponent</label>
            <select
              id="opponent_id"
              name="opponent_id"
              defaultValue={opponentId}
            >
              <option value="">Select opponent</option>
              {archetypes.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
            <label htmlFor="min_matches">
              Minimum matches with/without a card
            </label>
            <input
              id="min_matches"
              name="min_matches"
              type="number"
              min={1}
              defaultValue={minMatches}
            />
            <button type="submit" className="button--gradient">
              Apply
            </button>
          </FilterForm>
        </Card>

        {error ? <p className="muted">{error}</p> : null}
        {data ? (
          <>
            <p className="muted">
              {data.matches} recorded matches
              {data.score_rate !== null
                ? `, ${percent(data.score_rate)} score rate`
                : ""}
            </p>
            <RecommendationList title="Consider including" items={include} />
            <RecommendationList title="Consider cutting" items={cut} />
          </>
        ) : null}
      </div>
    </main>
  );
}
