import { PAGES, pageLabels } from './uses';

/** The search (command center): open from the menu bar, Ctrl+K, or "/" (EP-115). */
export const searching = $state({ open: false });

/** The pages the search can open: every page of the menu, and a few more. */
export const searchPages: { label: string; href: string }[] = [
	{ label: 'Today', href: '/today' },
	...PAGES.map((p) => ({ label: pageLabels[p], href: `/${p}` })),
	{ label: 'Roadmaps', href: '/roadmaps' },
	{ label: 'Progress', href: '/progress' },
	{ label: 'Collection', href: '/collection' },
	{ label: 'Life web', href: '/health#life-web-title' },
	{ label: 'Coffee and tea', href: '/coffee' },
	{ label: 'YouTube videos', href: '/watch?kind=video' },
	{ label: 'Podcasts', href: '/watch?kind=podcast' },
	{ label: 'Huberman protocols', href: '/habits#protocols-title' },
	{ label: 'Account', href: '/account' },
	{ label: 'Preferences', href: '/account/preferences' },
	{ label: 'Trash', href: '/account/trash' }
];
