import { describe, expect, it } from 'vitest';
import type { Note } from './notes';
import { tree } from './noteTree';

const note = (path: string, para: string, title = path) =>
	({
		id: path,
		path,
		para,
		title,
		kind: 'note',
		tags: [],
		hash: '',
		created_at: '',
		updated_at: ''
	}) as Note;

describe('tree', () => {
	it('groups by PARA and nests folders', () => {
		const groups = tree([
			note('0-Inbox/idea.md', 'inbox'),
			note('loose.md', 'inbox'),
			note('1-Projects/axon/plan.md', 'projects'),
			note('1-Projects/axon/notes/a.md', 'projects'),
			note('Zettel/permanent/proofs.md', 'other'),
			note('Daily/2026-10-10.md', 'other')
		]);
		const inbox = groups.find((g) => g.para === 'inbox')!;
		expect(inbox.root.notes.map((n) => n.path).sort()).toEqual(['0-Inbox/idea.md', 'loose.md']);
		const projects = groups.find((g) => g.para === 'projects')!;
		expect(projects.root.name).toBe('Projects');
		expect(projects.root.folders[0].name).toBe('axon');
		expect(projects.root.folders[0].count).toBe(2);
		expect(projects.root.folders[0].folders[0].notes[0].path).toBe('1-Projects/axon/notes/a.md');
		const other = groups.find((g) => g.para === 'other')!;
		expect(other.root.folders.map((f) => f.name)).toEqual(['Daily', 'Zettel']);
		expect(other.root.count).toBe(2);
	});
});
