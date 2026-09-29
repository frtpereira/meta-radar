// Remembers whether the user last looked at a decklist as an image grid
// or as a list, so the choice sticks across decklists and visits. Same
// mechanism as the theme (see components/theme-toggle.tsx): a plain
// localStorage key, read on the client only.

export type DecklistViewMode = "image" | "list";

export const DECKLIST_VIEW_STORAGE_KEY = "decklist-view";
export const DEFAULT_DECKLIST_VIEW: DecklistViewMode = "image";

function isViewMode(value: unknown): value is DecklistViewMode {
    return value === "image" || value === "list";
}

// Returns the stored mode, or the default when nothing valid is stored.
// localStorage can throw (blocked cookies, private mode), so never let
// that break the page.
export function readDecklistViewMode(): DecklistViewMode {
    try {
        const stored = window.localStorage.getItem(DECKLIST_VIEW_STORAGE_KEY);
        return isViewMode(stored) ? stored : DEFAULT_DECKLIST_VIEW;
    } catch {
        return DEFAULT_DECKLIST_VIEW;
    }
}

export function writeDecklistViewMode(mode: DecklistViewMode): void {
    try {
        window.localStorage.setItem(DECKLIST_VIEW_STORAGE_KEY, mode);
    } catch {
        // Preference just won't persist; the toggle still works this visit.
    }
}
