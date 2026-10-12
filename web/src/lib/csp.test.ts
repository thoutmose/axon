import { describe, expect, it } from 'vitest';
import appHtml from '../app.html?raw';

// The Content Security Policy (style-src 'self') blocks style attributes
// written in the HTML, in Firefox at least; style: directives are set from
// JavaScript and are allowed. A blocked style once showed the page title as
// text on every page.
const templates = import.meta.glob('/src/**/*.svelte', {
	query: '?raw',
	import: 'default',
	eager: true
});

describe('the CSP', () => {
	it('meets no style attribute in the markup', () => {
		const offenders = Object.entries({ ...templates, '/src/app.html': appHtml })
			.filter(([, src]) => / style="/.test(src as string))
			.map(([path]) => path);
		expect(offenders).toEqual([]);
	});
});
