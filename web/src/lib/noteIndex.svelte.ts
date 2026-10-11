import { api } from './api';
import type { Note } from './notes';

/** Every note's metadata (no content), for the sidebar, the quick switcher,
 * and [[link]] suggestions. Refreshed after notes change. */
export const noteIndex = $state({ notes: [] as Note[], loaded: false });

let pending: Promise<void> | null = null;

export function refreshNotes(): Promise<void> {
	pending ??= (async () => {
		const all: Note[] = [];
		let after: string | null = null;
		try {
			do {
				const q = new URLSearchParams({ limit: '100' });
				if (after) q.set('after', after);
				const res: { items: Note[]; next_cursor: string | null } = await api(`/v1/notes?${q}`);
				all.push(...res.items);
				after = res.next_cursor;
			} while (after && all.length < 5000);
			noteIndex.notes = all;
			noteIndex.loaded = true;
		} finally {
			pending = null;
		}
	})();
	return pending;
}
