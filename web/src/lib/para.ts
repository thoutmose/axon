import type { Schemas } from './api';

export type ParaItem = Schemas['ParaItem'];

export type ParaKind = {
	/** API collection and route, e.g. areas. */
	path: 'areas' | 'resources';
	type: 'area' | 'resource';
	title: string;
	one: string;
	intro: string;
	placeholder: string;
	empty: string;
	/** Notes folder where this kind lives in the vault. */
	folder: string;
	/** Ready-made starting points; each fills the form. */
	templates: { title: string; description: string }[];
};

export const areas: ParaKind = {
	path: 'areas',
	type: 'area',
	title: 'Areas',
	one: 'area',
	intro:
		'Areas are responsibilities you keep up over time, each with a standard to maintain. File projects, goals, habits, notes, and resources under them.',
	placeholder: 'New area, e.g. Health…',
	empty: 'No areas yet. Start from a template or name a responsibility above.',
	folder: '2-Areas',
	templates: [
		{ title: 'Health', description: 'Standard: sleep 7–8 h, move every day, eat mostly plants.' },
		{
			title: 'Finances',
			description: 'Standard: spend less than I earn, save 15 %, review monthly.'
		},
		{ title: 'Career', description: 'Standard: learn every week, ship work I am proud of.' },
		{ title: 'Home', description: 'Standard: tidy weekly, repairs within a month.' },
		{
			title: 'Relationships',
			description: 'Standard: one real conversation with family and friends each week.'
		},
		{ title: 'Learning', description: 'Standard: one roadmap active, review notes weekly.' }
	]
};

export const resources: ParaKind = {
	path: 'resources',
	type: 'resource',
	title: 'Resources',
	one: 'resource',
	intro:
		'Resources are topics and references of ongoing interest, useful later. Link notes, skills, and projects to them; archive them when they stop being useful.',
	placeholder: 'New resource, e.g. Rust, cooking, photography…',
	empty: 'No resources yet. Start from a template or name a topic above.',
	folder: '3-Resources',
	templates: [
		{
			title: 'Reading list',
			description: 'Books and articles to read, with why each one matters.'
		},
		{ title: 'Recipes', description: 'Recipes worth cooking again.' },
		{
			title: 'Writing references',
			description: 'Style guides, examples, and phrases worth keeping.'
		},
		{ title: 'Tools and setup', description: 'Software, settings, and dotfiles.' },
		{ title: 'Travel ideas', description: 'Places, itineraries, and tips.' }
	]
};
