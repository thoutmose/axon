/** Note templates for Zettelkasten, PARA, and a daily note. They only write
 * plain Markdown with front matter, so vault sync stays byte for byte. */
import { localDate as isoDate } from './dates';
import { stem } from './notes';
// The owner's vault templates (EP-100), shipped as they are in ~/SecondBrain/Templates.
import areaMd from './vaultTemplates/area.md?raw';
import bookMd from './vaultTemplates/book.md?raw';
import filmMd from './vaultTemplates/film.md?raw';
import gameMd from './vaultTemplates/game.md?raw';
import seriesMd from './vaultTemplates/series.md?raw';
import dailyMd from './vaultTemplates/daily.md?raw';
import goalMd from './vaultTemplates/goal.md?raw';
import habitMd from './vaultTemplates/habit.md?raw';
import literatureMd from './vaultTemplates/literature.md?raw';
import mediaMd from './vaultTemplates/media.md?raw';
import milestoneMd from './vaultTemplates/milestone.md?raw';
import monthlyMd from './vaultTemplates/monthly.md?raw';
import permanentMd from './vaultTemplates/permanent.md?raw';
import projectMd from './vaultTemplates/project.md?raw';
import resourceMd from './vaultTemplates/resource.md?raw';
import roadmapMd from './vaultTemplates/roadmap.md?raw';
import weeklyMd from './vaultTemplates/weekly.md?raw';
import test_5wMd from './vaultTemplates/test-5w.md?raw';
import test_blankMd from './vaultTemplates/test-blank.md?raw';
import test_cardsMd from './vaultTemplates/test-cards.md?raw';
import test_clozeMd from './vaultTemplates/test-cloze.md?raw';
import test_feynmanMd from './vaultTemplates/test-feynman.md?raw';
import test_monthlyMd from './vaultTemplates/test-monthly.md?raw';

export type Group =
	'zettelkasten' | 'para' | 'periodic' | 'plans' | 'reviews' | 'items' | 'other' | 'vault';

export type Template = {
	id: string;
	label: string;
	hint: string;
	group?: Group;
	path: (title: string, now: Date) => string;
	content: (title: string, now: Date) => string;
	/** The template file as it is in the vault, when it comes from one. */
	raw?: string;
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

/** Zettelkasten id: local YYYYMMDDHHmm. */
export function zettelId(d: Date): string {
	return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}${pad(d.getHours())}${pad(d.getMinutes())}`;
}

export function dailyPath(d: Date): string {
	return `Daily/${isoDate(d)}.md`;
}

export function dailyContent(d: Date): string {
	return fill(dailyMd, isoDate(d), d);
}

/** ISO week: 2026-W41. */
export function isoWeek(d: Date): string {
	const t = new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()));
	const day = t.getUTCDay() || 7;
	t.setUTCDate(t.getUTCDate() + 4 - day);
	const year = t.getUTCFullYear();
	const week = Math.ceil(((t.getTime() - Date.UTC(year, 0, 1)) / 86_400_000 + 1) / 7);
	return `${year}-W${pad(week)}`;
}

/** A template from the vault's Templates folder, filled when used. */
const fromVault = (
	id: string,
	group: Group,
	label: string,
	hint: string,
	raw: string,
	path: (title: string, now: Date) => string
): Template => ({ id, group, label, hint, raw, path, content: (t, now) => fill(raw, t, now) });

export const templates: Template[] = [
	{
		id: 'fleeting',
		group: 'zettelkasten',
		label: 'Fleeting note',
		hint: 'A quick thought for your inbox, to process later.',
		path: (t, now) => `0-Inbox/${zettelId(now)}-${slug(t)}.md`,
		content: (t, now) => `---\ntype: fleeting\ncreated: ${isoDate(now)}\n---\n# ${t}\n\n`
	},
	fromVault(
		'literature',
		'zettelkasten',
		'Literature note',
		'What a book, article, or course says, in your own words, with its retrieval tests.',
		literatureMd,
		(t, now) => `Zettel/literature/${zettelId(now)}-${slug(t)}.md`
	),
	// Films, series, books (EP-136), and video games (EP-149): literature notes made for each.
	fromVault(
		'film',
		'zettelkasten',
		'Film note',
		'A film: the scenes that stay, its ideas, its craft, and your verdict.',
		filmMd,
		(t, now) => `Zettel/literature/${zettelId(now)}-${slug(t)}.md`
	),
	fromVault(
		'series',
		'zettelkasten',
		'Series note',
		'A series, season by season: its characters, ideas, and your verdict.',
		seriesMd,
		(t, now) => `Zettel/literature/${zettelId(now)}-${slug(t)}.md`
	),
	fromVault(
		'book',
		'zettelkasten',
		'Book note',
		'A book, chapter by chapter, with a reading log, quotes, and what you will do differently.',
		bookMd,
		(t, now) => `Zettel/literature/${zettelId(now)}-${slug(t)}.md`
	),
	fromVault(
		'game',
		'zettelkasten',
		'Game note',
		'A video game: how it plays, the moments that stay, its design ideas, a play log, and your verdict.',
		gameMd,
		(t, now) => `Zettel/literature/${zettelId(now)}-${slug(t)}.md`
	),
	fromVault(
		'permanent',
		'zettelkasten',
		'Permanent note',
		'One idea, stated so it stands alone, linked to related ideas and why.',
		permanentMd,
		(t, now) => `Zettel/permanent/${zettelId(now)}-${slug(t)}.md`
	),
	fromVault(
		'project',
		'para',
		'Project note',
		'An outcome with a deadline, in 1-Projects (PARA).',
		projectMd,
		(t) => `1-Projects/${slug(t)}/${slug(t)}.md`
	),
	fromVault(
		'area',
		'para',
		'Area note',
		'A responsibility with a standard to keep, in 2-Areas (PARA).',
		areaMd,
		(t) => `2-Areas/${slug(t)}/${slug(t)}.md`
	),
	fromVault(
		'resource',
		'para',
		'Resource note',
		'A topic of ongoing interest, in 3-Resources (PARA).',
		resourceMd,
		(t) => `3-Resources/${slug(t)}.md`
	),
	{
		id: 'archive',
		group: 'para',
		label: 'Archive note',
		hint: 'Something finished or inactive, kept for reference, in 4-Archive (PARA).',
		path: (t) => `4-Archive/${slug(t)}.md`,
		content: (t, now) =>
			`---\ntype: archive\ntitle: ${t}\narchived: ${isoDate(now)}\n---\n## What it was\n\n\n## Outcome\n\n\n## Lessons\n- \n`
	},
	fromVault(
		'daily',
		'periodic',
		'Daily note',
		'Top 3, the retrieval tests due, habits, what you learned, and the journal.',
		dailyMd,
		(_t, now) => dailyPath(now)
	),
	fromVault(
		'weekly',
		'periodic',
		'Weekly note',
		'The GTD weekly review, the week’s tasks, wins and lessons.',
		weeklyMd,
		(_t, now) => `Weekly/${isoWeek(now)}.md`
	),
	fromVault(
		'monthly',
		'periodic',
		'Monthly note',
		'Goals and milestones reviewed, the roadmap checked, what you learned.',
		monthlyMd,
		(_t, now) => `Monthly/${isoDate(now).slice(0, 7)}.md`
	),
	fromVault(
		'roadmap',
		'plans',
		'Roadmap note',
		'Why, constraints, phases with milestones and exit checks, habits, and the learning method.',
		roadmapMd,
		(t) => `Roadmaps/${slug(t)}.md`
	),
	fromVault(
		'goal',
		'plans',
		'Goal note',
		'A specific, measurable outcome, why it matters, its milestones and habits.',
		goalMd,
		(t) => `Tracker/goals/${slug(t)}.md`
	),
	fromVault(
		'milestone',
		'plans',
		'Milestone note',
		'A definition of done, the evidence, and the resources.',
		milestoneMd,
		(t) => `Tracker/milestones/${slug(t)}.md`
	),
	fromVault(
		'habit',
		'plans',
		'Habit note',
		'Cue, routine, reward, why, and a log of check-ins.',
		habitMd,
		(t) => `Tracker/habits/${slug(t)}.md`
	),
	fromVault(
		'media',
		'plans',
		'Media note',
		'A book, film, series, course, podcast, or game: want, doing, done.',
		mediaMd,
		(t) => `Tracker/media/${slug(t)}.md`
	),
	fromVault(
		'test-5w',
		'reviews',
		'Five Ws test',
		'Half an hour to an hour after writing: what, why, who, when, where, how, without the note.',
		test_5wMd,
		(t) => `Reviews/${slug(t)}/5w.md`
	),
	fromVault(
		'test-blank',
		'reviews',
		'Blank page test',
		'A day later: write everything you remember.',
		test_blankMd,
		(t) => `Reviews/${slug(t)}/blank.md`
	),
	fromVault(
		'test-cards',
		'reviews',
		'Flashcards test',
		'Three days later: fronts and backs, then to Anki.',
		test_cardsMd,
		(t) => `Reviews/${slug(t)}/cards.md`
	),
	fromVault(
		'test-cloze',
		'reviews',
		'Cloze test',
		'A week later: fill in the blanks.',
		test_clozeMd,
		(t) => `Reviews/${slug(t)}/cloze.md`
	),
	fromVault(
		'test-monthly',
		'reviews',
		'Monthly test',
		'A month later: questions mixing this note with related ones.',
		test_monthlyMd,
		(t) => `Reviews/${slug(t)}/monthly.md`
	),
	fromVault(
		'test-feynman',
		'reviews',
		'Feynman test',
		'Three months later: explain it to a twelve-year-old, then study the gaps.',
		test_feynmanMd,
		(t) => `Reviews/${slug(t)}/feynman.md`
	),
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
		...(['film', 'series', 'book', 'game'] as const).map((k) =>
			item(
				k,
				{ film: 'Film', series: 'Series', book: 'Book', game: 'Video game' }[k],
				`Becomes a ${k === 'game' ? 'video game' : k} on your Media list.`,
				`Media/${{ film: 'Films', series: 'Series', book: 'Books', game: 'Games' }[k]}`,
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
	literature: (t, now) => `Zettel/literature/${zettelId(now)}-${slug(t)}.md`,
	permanent: (t, now) => `Zettel/permanent/${zettelId(now)}-${slug(t)}.md`,
	project: (t) => `1-Projects/${slug(t)}/${slug(t)}.md`,
	area: (t) => `2-Areas/${slug(t)}/${slug(t)}.md`,
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
	return (
		body
			.replace(/\{\{\s*title\s*\}\}|<%\s*tp\.file\.title\s*%>|\{title\}/g, title)
			.replace(/\{\{\s*date(:[^}]*)?\s*\}\}|<%\s*tp\.date\.now\([^)]*\)\s*%>/g, isoDate(now))
			.replace(/\{\{\s*time(:[^}]*)?\s*\}\}/g, time)
			.replace(/\{\{\s*zettel\s*\}\}/g, zettelId(now))
			// The owner's vault templates (EP-100): the creation time, the title's slug,
			// a test's note, and sensible starts; anything else is left blank.
			.replace(/\{\{\s*created\s*\}\}/g, `${isoDate(now)} ${time}`)
			.replace(/\{\{\s*slug\s*\}\}/g, slug(title))
			.replace(/\{\{\s*note\s*\}\}/g, title)
			.replace(/\{\{\s*kind\s*\}\}/g, 'book')
			.replace(/\{\{\s*status\s*\}\}/g, 'want')
			.replace(/\{\{\s*[\w-]+\s*\}\}/g, '')
	);
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
	return templates.map((t) =>
		t.raw
			? { path: `${templatesFolder}/${t.id}.md`, content: t.raw }
			: {
					path: `${templatesFolder}/${t.label}.md`,
					content: t
						.content('{{title}}', now)
						.replaceAll(zettelId(now), '{{zettel}}')
						.replaceAll(isoDate(now), '{{date}}')
				}
	);
}

/** A film's, series', book's, or game's details for its note template (EP-136, EP-149). */
export type MediaDetails = {
	kind: 'film' | 'series' | 'book' | 'game';
	title: string;
	creators: string;
	year: number | null;
	genres: string[];
	total: number | null;
	cover_id: string | null;
};

/** The note for a film, series, or book: its template, filled from the item. */
export function mediaNote(it: MediaDetails, now: Date): { path: string; content: string } {
	const t = templates.find((x) => x.id === it.kind)!;
	const raw = t
		.raw!.replace(/\{\{\s*creator\s*\}\}/g, it.creators)
		.replace(/\{\{\s*year\s*\}\}/g, it.year ? String(it.year) : '')
		.replace(/\{\{\s*genres\s*\}\}/g, it.genres.length ? `[${it.genres.join(', ')}]` : '')
		.replace(/\{\{\s*total\s*\}\}/g, it.total ? String(it.total) : '')
		.replace(/\{\{\s*cover\s*\}\}/g, it.cover_id ? `/api/v1/media/covers/${it.cover_id}` : '');
	return { path: t.path(it.title, now), content: fill(raw, it.title, now) };
}
