/** Themed spaces (EP-38): each area of Axon has its own scenery. */
export type Space =
	| 'notebook'
	| 'library'
	| 'blueprint'
	| 'garden'
	| 'summit'
	| 'almanac'
	| 'ledger'
	| 'atlas'
	| 'desk'
	| 'diary'
	| 'observatory'
	| 'lighthouse'
	| 'automata';

const byPath: [string, Space][] = [
	['/notes', 'notebook'],
	['/roadmaps', 'library'],
	['/resources', 'library'],
	['/learn', 'library'],
	['/cards', 'library'],
	['/media', 'library'],
	['/projects', 'blueprint'],
	['/habits', 'garden'],
	['/health', 'garden'],
	['/goals', 'summit'],
	['/progress', 'summit'],
	['/calendar', 'almanac'],
	['/household', 'ledger'],
	['/budget', 'ledger'],
	['/areas', 'atlas'],
	['/archive', 'atlas'],
	['/tasks', 'desk'],
	['/focus', 'desk'],
	['/journal', 'diary'],
	['/insights', 'observatory'],
	['/review', 'lighthouse'],
	['/agents', 'automata']
];

/** The space for a page, or null for pages with their own scenery or none. */
export function spaceFor(path: string): Space | null {
	for (const [prefix, space] of byPath) {
		if (path === prefix || path.startsWith(prefix + '/')) return space;
	}
	return null;
}

/**
 * What each scenery shows and what it nods to (EP-98): an original drawing
 * made from geometry, after a painting, a film, or a game the space recalls.
 */
export const spaceInfo: Record<Space, { title: string; about: string }> = {
	notebook: {
		title: 'A Da Vinci notebook',
		about:
			'Sepia studies in the margins, mirror writing, a flying machine: after Leonardo’s notebooks (Codex Atlanticus, 1478–1519), where every idea links to the next.'
	},
	library: {
		title: 'An old reading room',
		about:
			'Shelves and a warm lamp, after the Bodleian’s Duke Humfrey’s Library in Oxford, which played Hogwarts’ library on film: a place to learn slowly.'
	},
	blueprint: {
		title: 'A blueprint',
		about:
			'Plans on cyanotype paper, the way engineers drew from the 1840s: projects as something you build, one measured line at a time.'
	},
	garden: {
		title: 'A kitchen garden',
		about:
			'Leaves and stems that grow a little every day, like the gardens of Monet at Giverny or Stardew Valley’s seasons: habits and health compound.'
	},
	summit: {
		title: 'A contour map',
		about:
			'Contour lines climbing to a summit flag, after the survey maps of the Alps and the mountain in Celeste: goals are climbed, step by step.'
	},
	almanac: {
		title: 'A star almanac',
		about:
			'Constellations on a night chart, after the old almanacs farmers and sailors planned their year by: the calendar as a map of time.'
	},
	ledger: {
		title: 'A ledger',
		about:
			'Green-ruled accounting paper with a red margin, as in the merchants’ books of Luca Pacioli’s double entry (1494): money, written down calmly.'
	},
	atlas: {
		title: 'An old atlas',
		about:
			'Meridians, a coastline, and a compass rose, after the portolan charts of the Age of Sail: areas of life to look after, like lands on a map.'
	},
	desk: {
		title: 'A dotted notebook',
		about:
			'A bullet journal’s dot grid on a clean desk: tasks to capture and cross off, without noise.'
	},
	diary: {
		title: 'Evening light on a desk',
		about:
			'Light through a leaded window falls on an open diary, after Vermeer’s rooms (The Milkmaid, c. 1658): a quiet moment to write how the day felt.'
	},
	observatory: {
		title: 'An observatory',
		about:
			'Star trails turning round the pole and Galileo’s sketches of the moon (Sidereus Nuncius, 1610): patterns found by watching patiently.'
	},
	lighthouse: {
		title: 'A lighthouse',
		about:
			'A beam sweeping the sea from a headland, after Edward Hopper’s lighthouses on the Maine coast: once a week, look back and set the course.'
	},
	automata: {
		title: 'Clockwork automata',
		about:
			'Gears and the cams that remember each letter, after Jaquet-Droz’s automaton The Writer (1770s) and the clockwork of Hugo: agents that work for you, and wait for your approval.'
	}
};
