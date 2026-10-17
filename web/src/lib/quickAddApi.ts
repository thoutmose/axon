import { api } from './api';
import { templates } from './noteTemplates';
import type { Parsed } from './quickAdd';
import { capture } from './tasks';

/** A fleeting note in the inbox, with the tags typed as #tags. */
export function fleeting(text: string, tags: string[]) {
	const t = templates.find((x) => x.id === 'fleeting')!;
	const now = new Date();
	let content = t.content(text, now);
	if (tags.length)
		content = content.replace('type: fleeting\n', `type: fleeting\ntags: [${tags.join(', ')}]\n`);
	return { path: t.path(text, now), content };
}

/** Adds what a quick-add line describes, and says what was done. */
export async function addParsed(p: Parsed): Promise<string> {
	switch (p.kind) {
		case 'task':
			await capture(p.title, {
				...(p.context ? { context: p.context } : {}),
				...(p.due ? { due_on: p.due } : {}),
				...(p.status ? { status: p.status } : {}),
				...(p.remind ? { remind_at: p.remind.toISOString() } : {})
			});
			return p.status ? 'Saved for someday.' : 'Captured to your inbox.';
		case 'note':
			await api('/v1/notes', { method: 'POST', body: fleeting(p.title, p.tags) });
			return 'Saved as a note in your inbox.';
		case 'book':
		case 'film':
		case 'series':
		case 'game':
			await api('/v1/media', {
				method: 'POST',
				body: { kind: p.kind, title: p.title, status: 'want' }
			});
			return `Added to ${p.kind === 'book' ? 'books to read' : p.kind === 'game' ? 'games to play' : 'films and series to watch'}.`;
		case 'habit':
			await api('/v1/habits', { method: 'POST', body: { title: p.title } });
			return 'Habit added. Give it a two-minute version on the Habits page.';
		case 'goal':
			await api('/v1/goals', {
				method: 'POST',
				body: { title: p.title, ...(p.due ? { target_date: p.due } : {}) }
			});
			return 'Goal added.';
		case 'project':
			await api('/v1/projects', { method: 'POST', body: { title: p.title } });
			return 'Project added.';
		case 'card':
			await api('/v1/flashcards', {
				method: 'POST',
				body: { front: p.title, back: p.back ?? '', deck: p.tags[0] ?? 'Default' }
			});
			return `Card added to ${p.tags[0] ?? 'Default'}.`;
	}
}
