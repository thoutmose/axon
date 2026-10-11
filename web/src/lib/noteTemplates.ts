/** Note templates for Zettelkasten, PARA, and a daily note. They only write
 * plain Markdown with front matter, so vault sync stays byte for byte. */

export type Template = {
	id: string;
	label: string;
	hint: string;
	path: (title: string, now: Date) => string;
	content: (title: string, now: Date) => string;
};

/** "Raft & Paxos!" → "raft-paxos" (as the API slugs titles). */
export function slug(title: string): string {
	const s = title
		.toLowerCase()
		.replace(/[^\p{L}\p{N}]+/gu, '-')
		.replace(/^-+|-+$/g, '')
		.slice(0, 80)
		.replace(/-+$/, '');
	return s || 'note';
}

const pad = (n: number) => String(n).padStart(2, '0');

/** Local date as YYYY-MM-DD. */
export function isoDate(d: Date): string {
	return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** Zettelkasten id: local YYYYMMDDHHmm. */
export function zettelId(d: Date): string {
	return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}${pad(d.getHours())}${pad(d.getMinutes())}`;
}

export function dailyPath(d: Date): string {
	return `Daily/${isoDate(d)}.md`;
}

export function dailyContent(d: Date): string {
	const yesterday = new Date(d);
	yesterday.setDate(d.getDate() - 1);
	return `---\ntype: daily\ndate: ${isoDate(d)}\n---\n# ${isoDate(d)}\n\nPrevious: [[${isoDate(yesterday)}]]\n\n## Top three\n- [ ] \n- [ ] \n- [ ] \n\n## Notes\n\n\n## Reflection\n- What went well?\n- What did I learn?\n`;
}

export const templates: Template[] = [
	{
		id: 'fleeting',
		label: 'Fleeting note',
		hint: 'A quick thought for your inbox, to process later.',
		path: (t, now) => `0-Inbox/${zettelId(now)}-${slug(t)}.md`,
		content: (t, now) => `---\ntype: fleeting\ncreated: ${isoDate(now)}\n---\n# ${t}\n\n`
	},
	{
		id: 'literature',
		label: 'Literature note',
		hint: 'What a book, article, or course says, in your own words.',
		path: (t) => `Zettel/literature/${slug(t)}.md`,
		content: (t, now) =>
			`---\ntype: literature\ntitle: ${t}\nsource: \nauthor: \ncreated: ${isoDate(now)}\ntags: []\n---\n## Key ideas\n- \n\n## Quotes\n> \n\n## My thoughts\n\n`
	},
	{
		id: 'permanent',
		label: 'Permanent note',
		hint: 'One idea, stated so it stands alone, linked to related ideas.',
		path: (t, now) => `Zettel/permanent/${zettelId(now)}-${slug(t)}.md`,
		content: (t, now) =>
			`---\ntype: permanent\nid: ${zettelId(now)}\ntitle: ${t}\ncreated: ${isoDate(now)}\ntags: []\n---\n## Idea\n\n\n## Why it matters\n\n\n## Related\n- [[ ]]\n`
	},
	{
		id: 'project',
		label: 'Project',
		hint: 'An outcome with a deadline (PARA: Projects).',
		path: (t) => `1-Projects/${slug(t)}/${slug(t)}.md`,
		content: (t, now) =>
			`---\ntype: project\ntitle: ${t}\nstatus: active\ncreated: ${isoDate(now)}\n---\n## Outcome\n\n\n## Next actions\n- [ ] \n\n## Notes\n\n`
	},
	{
		id: 'area',
		label: 'Area',
		hint: 'A responsibility you keep up over time (PARA: Areas).',
		path: (t) => `2-Areas/${slug(t)}.md`,
		content: (t) =>
			`---\ntype: area\ntitle: ${t}\n---\n## Standard to keep\n\n\n## Notes & resources\n- \n`
	},
	{
		id: 'resource',
		label: 'Resource',
		hint: 'A topic you are interested in (PARA: Resources).',
		path: (t) => `3-Resources/${slug(t)}.md`,
		content: (t) =>
			`---\ntype: resource\ntitle: ${t}\ntags: []\n---\n## Summary\n\n\n## Links\n- \n`
	},
	{
		id: 'meeting',
		label: 'Meeting',
		hint: 'Who, what was decided, and what happens next.',
		path: (t, now) => `0-Inbox/${isoDate(now)}-${slug(t)}.md`,
		content: (t, now) =>
			`---\ntype: meeting\ndate: ${isoDate(now)}\nattendees: \n---\n# ${t}\n\n## Decisions\n- \n\n## Actions\n- [ ] \n`
	},
	{
		id: 'blank',
		label: 'Blank note',
		hint: 'Just a title, in your inbox.',
		path: (t) => `0-Inbox/${slug(t)}.md`,
		content: (t) => `# ${t}\n\n`
	}
];

/** The person's own default template from Preferences ({title} is replaced). */
export function customTemplate(body: string): Template {
	return {
		id: 'mine',
		label: 'My template',
		hint: 'Your default from Preferences, in your inbox.',
		path: (t) => `0-Inbox/${slug(t)}.md`,
		content: (t) => body.replaceAll('{title}', t)
	};
}
