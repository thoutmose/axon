/** Note templates for Zettelkasten, PARA, and a daily note. They only write
 * plain Markdown with front matter, so vault sync stays byte for byte. */
import { stem } from './notes';

export type Group = 'zettelkasten' | 'para' | 'items' | 'other' | 'vault';

export type Template = {
	id: string;
	label: string;
	hint: string;
	group?: Group;
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
function isoDate(d: Date): string {
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
		group: 'zettelkasten',
		label: 'Fleeting note',
		hint: 'A quick thought for your inbox, to process later.',
		path: (t, now) => `0-Inbox/${zettelId(now)}-${slug(t)}.md`,
		content: (t, now) => `---\ntype: fleeting\ncreated: ${isoDate(now)}\n---\n# ${t}\n\n`
	},
	{
		id: 'literature',
		group: 'zettelkasten',
		label: 'Literature note',
		hint: 'What a book, article, or course says, in your own words.',
		path: (t) => `Zettel/literature/${slug(t)}.md`,
		content: (t, now) =>
			`---\ntype: literature\ntitle: ${t}\nsource: \nauthor: \ncreated: ${isoDate(now)}\ntags: []\n---\n## Key ideas\n- \n\n## Quotes\n> \n\n## My thoughts\n\n`
	},
	{
		id: 'permanent',
		group: 'zettelkasten',
		label: 'Permanent note',
		hint: 'One idea, stated so it stands alone, linked to related ideas.',
		path: (t, now) => `Zettel/permanent/${zettelId(now)}-${slug(t)}.md`,
		content: (t, now) =>
			`---\ntype: permanent\nid: ${zettelId(now)}\ntitle: ${t}\ncreated: ${isoDate(now)}\ntags: []\n---\n## Idea\n\n\n## Why it matters\n\n\n## Related\n- [[ ]]\n`
	},
	{
		id: 'project',
		group: 'para',
		label: 'Project note',
		hint: 'Notes for an outcome with a deadline, in 1-Projects (PARA).',
		path: (t) => `1-Projects/${slug(t)}/${slug(t)}.md`,
		content: (t, now) =>
			`---\ntype: project\ntitle: ${t}\nstatus: active\ncreated: ${isoDate(now)}\n---\n## Outcome\n\n\n## Next actions\n- [ ] \n\n## Notes\n\n`
	},
	{
		id: 'area',
		group: 'para',
		label: 'Area note',
		hint: 'Notes for a responsibility you keep up, in 2-Areas (PARA).',
		path: (t) => `2-Areas/${slug(t)}.md`,
		content: (t) =>
			`---\ntype: area\ntitle: ${t}\n---\n## Standard to keep\n\n\n## Notes & resources\n- \n`
	},
	{
		id: 'resource',
		group: 'para',
		label: 'Resource note',
		hint: 'Notes on a topic you collect, in 3-Resources (PARA).',
		path: (t) => `3-Resources/${slug(t)}.md`,
		content: (t) =>
			`---\ntype: resource\ntitle: ${t}\ntags: []\n---\n## Summary\n\n\n## Links\n- \n`
	},
	{
		id: 'archive',
		group: 'para',
		label: 'Archive note',
		hint: 'Something finished or inactive, kept for reference, in 4-Archive (PARA).',
		path: (t) => `4-Archive/${slug(t)}.md`,
		content: (t, now) =>
			`---\ntype: archive\ntitle: ${t}\narchived: ${isoDate(now)}\n---\n## What it was\n\n\n## Outcome\n\n\n## Lessons\n- \n`
	},
	{
		id: 'meeting',
		group: 'other',
		label: 'Meeting',
		hint: 'Who, what was decided, and what happens next.',
		path: (t, now) => `0-Inbox/${isoDate(now)}-${slug(t)}.md`,
		content: (t, now) =>
			`---\ntype: meeting\ndate: ${isoDate(now)}\nattendees: \n---\n# ${t}\n\n## Decisions\n- \n\n## Actions\n- [ ] \n`
	},
	{
		id: 'blank',
		group: 'other',
		label: 'Blank note',
		hint: 'Just a title, in your inbox.',
		path: (t) => `0-Inbox/${slug(t)}.md`,
		content: (t) => `# ${t}\n\n`
	}
];

/** Item notes: under the vault folder they become real tasks, goals,
 * milestones, projects, areas, resources, habits, and films, series, or
 * books, and stay in step with them (EP-48). The sections match what Axon
 * writes for each item. */
export function itemTemplates(folder = 'Axon'): Template[] {
	const fm = (type: string, extra: string) => `---\ntype: ${type}\n${extra}---\n`;
	const item = (
		id: string,
		label: string,
		hint: string,
		dir: string,
		body: (t: string, now: Date) => string
	): Template => ({
		id: `item-${id}`,
		group: 'items',
		label,
		hint,
		path: (t) => `${folder}/${dir}/${fileName(t)}.md`,
		content: body
	});
	return [
		item(
			'task',
			'Task',
			'Becomes a task (inbox, next, waiting, someday).',
			'Tasks',
			(t) => fm('task', 'status: inbox\ncontext: \nscheduled: \ndue: \n') + `# ${t}\n\n## Notes\n\n`
		),
		item(
			'goal',
			'Goal',
			'Becomes a goal with an outcome and a target date.',
			'Goals',
			(t) => fm('goal', 'status: active\ntarget: \n') + `# ${t}\n\n## Outcome\n\n\n## Notes\n\n`
		),
		item(
			'milestone',
			'Milestone',
			'A step of a goal: name the goal as goal: [[Goal title]].',
			'Milestones',
			(t) => fm('milestone', 'goal: "[[ ]]"\ndue: \ndone: false\n') + `# ${t}\n\n## Notes\n\n`
		),
		item(
			'project',
			'Project',
			'Becomes a project with a Kanban board.',
			'Projects',
			(t) =>
				fm('project', 'status: active\nwip_limit: \n') +
				`# ${t}\n\n## Description\n\n\n## Notes\n\n`
		),
		item(
			'area',
			'Area',
			'Becomes an area: a standard you keep up.',
			'Areas',
			(t) => fm('area', 'status: active\nurl: \n') + `# ${t}\n\n## Description\n\n\n## Notes\n\n`
		),
		item(
			'resource',
			'Resource',
			'Becomes a resource: a topic you collect.',
			'Resources',
			(t) =>
				fm('resource', 'status: active\nurl: \n') + `# ${t}\n\n## Description\n\n\n## Notes\n\n`
		),
		item(
			'habit',
			'Habit',
			'Becomes a habit, designed with Atomic Habits.',
			'Habits',
			(t) =>
				fm(
					'habit',
					'kind: build\nstatus: active\ndays: [1, 2, 3, 4, 5, 6, 7]\nintention_time: \nintention_place: \nstack_after: \n'
				) +
				`# ${t}\n\n## Identity\nI am a person who \n\n## Make it obvious (cue)\n\n\n## Make it attractive (craving)\n\n\n## Make it easy (two-minute version)\n\n\n## Make it satisfying (reward)\n\n\n## Habit contract\n\n\n## Notes\n\n`
		),
		...(['film', 'series', 'book'] as const).map((k) =>
			item(
				k,
				k === 'film' ? 'Film' : k === 'series' ? 'Series' : 'Book',
				`Becomes a ${k} on your Media list.`,
				`Media/${k === 'film' ? 'Films' : k === 'series' ? 'Series' : 'Books'}`,
				(t) =>
					fm(k, 'status: want\nrating: \nprogress: 0\ntotal: \nyear: \ncreators: \ngenres: []\n') +
					`# ${t}\n\n## Review\n\n\n## Notes\n\n`
			)
		)
	];
}

/** File names as the vault mirror writes them: the title, without characters files cannot hold. */
function fileName(title: string): string {
	const s = title
		.replace(/[/\\:*?"<>|#^[\]]+/g, ' ')
		.split(/\s+/)
		.filter(Boolean)
		.join(' ')
		.slice(0, 120)
		.trim();
	return !s || s.startsWith('.') ? `Untitled ${s}`.trim() : s;
}

/** The "## " sections a template or note has, in order. */
export function sections(content: string): string[] {
	return [...content.matchAll(/^## (.+)$/gm)].map((m) => m[1].trim());
}

/** The front-matter keys a template sets. */
export function frontKeys(content: string): string[] {
	const m = /^---\n([\s\S]*?)\n---/.exec(content);
	if (!m) return [];
	return m[1]
		.split('\n')
		.map((l) => l.split(':')[0].trim())
		.filter((k) => k && !k.startsWith('#'));
}

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

/** Where templates live in the vault, as in Obsidian: Templates/, _templates/, 99-Templates/… */
export const templatesFolder = 'Templates';

export function isTemplatePath(path: string): boolean {
	const top = path.split('/')[0];
	return path.includes('/') && /^(\d+[-_ ]*)?_?templates?$/i.test(top);
}

/** Where a note made from a vault template goes, by the template's front-matter type. */
const destinations: Record<string, (t: string, now: Date, folder: string) => string> = {
	fleeting: (t, now) => `0-Inbox/${zettelId(now)}-${slug(t)}.md`,
	literature: (t) => `Zettel/literature/${slug(t)}.md`,
	permanent: (t, now) => `Zettel/permanent/${zettelId(now)}-${slug(t)}.md`,
	project: (t) => `1-Projects/${slug(t)}/${slug(t)}.md`,
	area: (t) => `2-Areas/${slug(t)}.md`,
	resource: (t) => `3-Resources/${slug(t)}.md`,
	archive: (t) => `4-Archive/${slug(t)}.md`,
	daily: (_t, now) => dailyPath(now),
	// Item types land in the vault mirror's folder (Preferences), where they become items.
	task: (t, _now, folder) => `${folder}/Tasks/${fileName(t)}.md`,
	goal: (t, _now, folder) => `${folder}/Goals/${fileName(t)}.md`,
	milestone: (t, _now, folder) => `${folder}/Milestones/${fileName(t)}.md`,
	habit: (t, _now, folder) => `${folder}/Habits/${fileName(t)}.md`,
	film: (t, _now, folder) => `${folder}/Media/Films/${fileName(t)}.md`,
	series: (t, _now, folder) => `${folder}/Media/Series/${fileName(t)}.md`,
	book: (t, _now, folder) => `${folder}/Media/Books/${fileName(t)}.md`
};

/**
 * Fills a template body. Understands Axon's {title}, Obsidian's core
 * Templates plugin ({{title}}, {{date}}, {{time}}, {{date:FORMAT}} as a
 * plain date), the common Templater calls for title and date, and {{zettel}}.
 */
export function fill(body: string, title: string, now: Date): string {
	const time = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`;
	return body
		.replace(/\{\{\s*title\s*\}\}|<%\s*tp\.file\.title\s*%>|\{title\}/g, title)
		.replace(/\{\{\s*date(:[^}]*)?\s*\}\}|<%\s*tp\.date\.now\([^)]*\)\s*%>/g, isoDate(now))
		.replace(/\{\{\s*time(:[^}]*)?\s*\}\}/g, time)
		.replace(/\{\{\s*zettel\s*\}\}/g, zettelId(now));
}

/** A template that is a note in the vault's templates folder. */
export function vaultTemplate(
	note: { id: string; path: string; title: string; kind: string },
	body: string,
	folder = 'Axon'
): Template {
	const dest = destinations[note.kind] ?? ((t: string) => `0-Inbox/${slug(t)}.md`);
	return {
		id: `vault:${note.id}`,
		label: templateName(note.path),
		hint: `Your template ${note.path}`,
		path: (t, now) => dest(t, now, folder),
		content: (t, now) => fill(body, t, now)
	};
}

/** A vault template is named after its file: Templates/Book note.md is "Book note". */
export const templateName = stem;

/** The built-in templates as editable files for the vault's templates folder. */
export function builtInFiles(now: Date): { path: string; content: string }[] {
	return templates.map((t) => ({
		path: `${templatesFolder}/${t.label}.md`,
		content: t
			.content('{{title}}', now)
			.replaceAll(zettelId(now), '{{zettel}}')
			.replaceAll(isoDate(now), '{{date}}')
	}));
}
