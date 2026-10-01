export default function SourceBadges({
    ingestSource,
    isOnline,
}: {
    ingestSource: "play_api" | "labs";
    isOnline: boolean;
}) {
    if (ingestSource === "labs") {
        return <span className="badge">Pokémon</span>;
    }
    return (
        <>
            <span className="badge">Limitless</span>{" "}
            <span className={`badge ${isOnline ? "badge--online" : ""}`}>
                {isOnline ? "Online" : "In Person"}
            </span>
        </>
    );
}
