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
	| 'desk';

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
	['/focus', 'desk']
];

/** The space for a page, or null for pages with their own scenery or none. */
export function spaceFor(path: string): Space | null {
	for (const [prefix, space] of byPath) {
		if (path === prefix || path.startsWith(prefix + '/')) return space;
	}
	return null;
}
