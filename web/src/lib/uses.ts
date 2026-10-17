// What a person wants Axon for (EP-94), and the pages and Today sections
// each use brings: the menu shows those, so a new account is not met by
// every page at once. Hidden pages still open from a link or a search.

/** The pages the menu can hide, as in the settings' menu.hidden. */
export const PAGES = [
	'tasks',
	'projects',
	'calendar',
	'focus',
	'notes',
	'watch',
	'areas',
	'resources',
	'archive',
	'learn',
	'goals',
	'habits',
	'health',
	'media',
	'budget',
	'household',
	'journal',
	'digest',
	'review',
	'decisions',
	'insights',
	'agents'
] as const;

/** How each page is named in the menu. */
export const pageLabels: Record<(typeof PAGES)[number], string> = {
	tasks: 'Tasks',
	projects: 'Projects',
	calendar: 'Calendar',
	focus: 'Focus',
	notes: 'Notes',
	watch: 'Tech watch',
	areas: 'Areas',
	resources: 'Resources',
	archive: 'Archive',
	learn: 'Learn',
	goals: 'Goals',
	habits: 'Habits',
	health: 'Health',
	media: 'Media',
	budget: 'Budget',
	household: 'Household',
	journal: 'Journal',
	digest: 'Digest',
	review: 'Weekly review',
	decisions: 'Decisions',
	insights: 'Insights',
	agents: 'Agents'
};

/** Today's sections in their usual order. */
const SECTIONS = [
	'kpis',
	'tasks',
	'milestones',
	'goals',
	'notes',
	'learning',
	'habits',
	'calendar',
	'household',
	'journal',
	'water',
	'caffeine',
	'overview'
];

/** Reflecting is for everyone: the digest, the weekly review, decisions, insights, and agents stay. */
const ALWAYS = ['digest', 'review', 'decisions', 'insights', 'agents'];

export const USES: { id: string; label: string; pages: string[]; sections: string[] }[] = [
	{
		id: 'tasks',
		label: 'Tasks and projects',
		pages: ['tasks', 'projects', 'calendar', 'focus'],
		sections: ['tasks', 'milestones', 'calendar']
	},
	{
		id: 'notes',
		label: 'Notes and knowledge',
		pages: ['notes', 'areas', 'resources', 'archive'],
		sections: ['notes']
	},
	{ id: 'learning', label: 'Learning', pages: ['learn'], sections: ['roadmap', 'learning'] },
	{ id: 'watch', label: 'A tech watch', pages: ['watch'], sections: [] },
	{
		id: 'goals',
		label: 'Goals and habits',
		pages: ['goals', 'habits'],
		sections: ['goals', 'habits']
	},
	{
		id: 'health',
		label: 'Health and well-being',
		pages: ['health', 'journal'],
		sections: ['journal', 'water', 'caffeine']
	},
	{ id: 'money', label: 'Money', pages: ['budget'], sections: [] },
	{ id: 'media', label: 'Films, series, books, and games', pages: ['media'], sections: [] },
	{ id: 'home', label: 'A shared home', pages: ['household'], sections: ['household'] }
];

/** The pages to hide and Today's sections for the chosen uses. */
export function planFor(uses: string[]): { hidden: string[]; sections: string[] } {
	const chosen = USES.filter((u) => uses.includes(u.id));
	const shown = new Set([...ALWAYS, ...chosen.flatMap((u) => u.pages)]);
	const sections = new Set(['kpis', 'overview', ...chosen.flatMap((u) => u.sections)]);
	return {
		hidden: PAGES.filter((p) => !shown.has(p)),
		sections: SECTIONS.filter((s) => sections.has(s))
	};
}
