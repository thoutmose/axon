import { describe, expect, it } from 'vitest';
import { searchPages } from './search.svelte';
import { PAGES } from './uses';

const routes = Object.keys(import.meta.glob('/src/routes/**/+page.svelte'));

// The search in the menu bar (EP-115) finds every page, not only content.
describe('search', () => {
	it('offers every page of the menu, and each one exists', () => {
		const hrefs = searchPages.map((p) => p.href);
		for (const p of PAGES) expect(hrefs).toContain(`/${p}`);
		for (const href of hrefs)
			expect(routes, href).toContain(`/src/routes/(app)${href}/+page.svelte`);
	});
});
