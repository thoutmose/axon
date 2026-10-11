/** Keyboard shortcuts (EP-32). Single keys and "g" chords never fire while
 * typing, and they can all be turned off in Preferences (WCAG 2.1.4). */

export type Shortcut = { keys: string; label: string; href?: string };

export const goTo: Shortcut[] = [
	{ keys: 'g h', label: 'Today', href: '/today' },
	{ keys: 'g t', label: 'Tasks', href: '/tasks' },
	{ keys: 'g n', label: 'Notes', href: '/notes' },
	{ keys: 'g o', label: 'Goals', href: '/goals' },
	{ keys: 'g p', label: 'Projects', href: '/projects' },
	{ keys: 'g l', label: 'Learn', href: '/roadmaps' },
	{ keys: 'g r', label: 'Review flashcards', href: '/cards/review' },
	{ keys: 'g b', label: 'Habits', href: '/habits' },
	{ keys: 'g f', label: 'Focus', href: '/focus' },
	{ keys: 'g c', label: 'Calendar', href: '/calendar' },
	{ keys: 'g m', label: 'Media', href: '/media' },
	{ keys: 'g w', label: 'Health', href: '/health' },
	{ keys: 'g a', label: 'Agents', href: '/agents' },
	{ keys: 'g s', label: 'Preferences', href: '/account/preferences' }
];

export const actions: Shortcut[] = [
	{ keys: 'n', label: 'Capture a task' },
	{ keys: '/', label: 'Search everything' },
	{ keys: 'Ctrl K', label: 'Command center (works while typing)' },
	{ keys: '?', label: 'Show these shortcuts' },
	{ keys: 'Shift ← → ↑ ↓', label: 'Move a focused card on a board' },
	{ keys: 'Ctrl B / I / E / L', label: 'Bold, italic, code, link in the editor' }
];

export const vimKeys: Shortcut[] = [
	{ keys: 'Esc · i a o', label: 'Normal mode · insert before, after, new line' },
	{ keys: 'h j k l · w b e · 0 $ · gg G', label: 'Move' },
	{ keys: 'd c y + motion · dd cc yy', label: 'Delete, change, yank' },
	{ keys: 'x p P u Ctrl R', label: 'Delete a character, paste, undo, redo' },
	{ keys: 'v V', label: 'Visual and visual line' },
	{ keys: ':w', label: 'Save the note' }
];

/** Whether a key event comes from a place where the person is typing. */
export function typing(target: EventTarget | null): boolean {
	const el = target as HTMLElement | null;
	return !!el && (el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName));
}

/** Resolves a "g" chord's second key to a page. */
export function chord(second: string): string | undefined {
	return goTo.find((s) => s.keys === `g ${second}`)?.href;
}
