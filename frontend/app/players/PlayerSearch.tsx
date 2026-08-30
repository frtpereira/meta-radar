"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

// A simple search box: the player list isn't paginated/browsable server
// side (there's no /api/players list endpoint, only a per-nickname
// lookup), so the only entry point into a player's history is knowing
// their exact nickname and navigating straight to /players/[nickname].
export default function PlayerSearch() {
    const router = useRouter();
    const [nickname, setNickname] = useState("");

    function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
        e.preventDefault();
        const trimmed = nickname.trim();
        if (!trimmed) return;
        router.push(`/players/${encodeURIComponent(trimmed)}`);
    }

    return (
        <form className="selector" onSubmit={handleSubmit}>
            <div className="selector__field" style={{ flex: 1 }}>
                <label className="sr-only" htmlFor="player_nickname">
                    Player nickname
                </label>
                <input
                    id="player_nickname"
                    type="search"
                    placeholder="Search by nickname…"
                    value={nickname}
                    onChange={(e) => setNickname(e.target.value)}
                    style={{ width: "100%", minWidth: "auto" }}
                />
            </div>
            <button type="submit">Search</button>
        </form>
    );
}
