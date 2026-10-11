import type { Note } from './notes';

export type Folder = {
	name: string;
	path: string;
	folders: Folder[];
	notes: Note[];
	count: number;
};
export type Group = { para: string; label: string; root: Folder };

const groups: [string, string][] = [
	['inbox', 'Inbox'],
	['projects', 'Projects'],
	['areas', 'Areas'],
	['resources', 'Resources'],
	['archive', 'Archive'],
	['other', 'Zettelkasten & more']
];

/** Builds PARA groups, each a folder tree from the note paths. */
export function tree(notes: Note[]): Group[] {
	const out: Group[] = groups.map(([para, label]) => ({
		para,
		label,
		root: { name: label, path: '', folders: [], notes: [], count: 0 }
	}));
	const byPara = new Map(out.map((g) => [g.para, g]));
	for (const n of notes) {
		const g = byPara.get(n.para) ?? byPara.get('other')!;
		const parts = n.path.split('/').slice(0, -1);
		let folder = g.root;
		folder.count++;
		let prefix = '';
		for (const part of parts) {
			prefix = prefix ? `${prefix}/${part}` : part;
			let next = folder.folders.find((f) => f.name === part);
			if (!next) {
				next = { name: part, path: prefix, folders: [], notes: [], count: 0 };
				folder.folders.push(next);
			}
			next.count++;
			folder = next;
		}
		folder.notes.push(n);
	}
	const sort = (f: Folder) => {
		f.folders.sort((a, b) => a.name.localeCompare(b.name));
		f.notes.sort((a, b) => a.title.localeCompare(b.title));
		f.folders.forEach(sort);
	};
	out.forEach((g) => sort(g.root));
	// A PARA group's top folder (e.g. "1-Projects") is the group itself.
	for (const g of out) {
		if (
			g.para !== 'other' &&
			g.para !== 'inbox' &&
			g.root.folders.length === 1 &&
			g.root.notes.length === 0
		) {
			const only = g.root.folders[0];
			g.root = { ...only, name: g.label };
		}
		if (g.para === 'inbox') {
			const inbox = g.root.folders.find((f) => /inbox/i.test(f.name));
			if (inbox) {
				g.root.folders = g.root.folders.filter((f) => f !== inbox);
				g.root.folders.push(...inbox.folders);
				g.root.notes.push(...inbox.notes);
			}
		}
	}
	return out;
}
