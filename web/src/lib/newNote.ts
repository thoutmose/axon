import { goto } from '$app/navigation';
import { ApiError, api } from './api';
import { refreshNotes } from './noteIndex.svelte';
import type { Note } from './notes';
import { dailyContent, dailyPath, type Template } from './noteTemplates';

/** Creates a note from a template and opens it for editing. */
export async function createFromTemplate(t: Template, title: string): Promise<void> {
	const now = new Date();
	const n = await api<Note>('/v1/notes', {
		method: 'POST',
		body: { path: t.path(title, now), content: t.content(title, now) }
	});
	await refreshNotes();
	await goto(`/notes/${n.id}?edit=1`);
}

/** Opens today's daily note, creating it the first time. */
export async function openDaily(): Promise<void> {
	const now = new Date();
	const path = dailyPath(now);
	const found = await api<{ items: Note[] }>(`/v1/notes?path=${encodeURIComponent(path)}`);
	if (found.items.length) {
		await goto(`/notes/${found.items[0].id}`);
		return;
	}
	try {
		const n = await api<Note>('/v1/notes', {
			method: 'POST',
			body: { path, content: dailyContent(now) }
		});
		await refreshNotes();
		await goto(`/notes/${n.id}?edit=1`);
	} catch (err) {
		if (err instanceof ApiError && err.status === 409) return openDaily(); // made in another tab
		throw err;
	}
}
