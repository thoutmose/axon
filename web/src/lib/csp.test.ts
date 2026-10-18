import { describe, expect, it } from 'vitest';
import appHtml from '../app.html?raw';
import config from '../../svelte.config.js';
import writeRoot from '../../node_modules/@sveltejs/kit/src/core/sync/write_root.js?raw';

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

	it("allows exactly the style of SvelteKit's announcer, by its hash", async () => {
		const style = writeRoot.match(/id="svelte-announcer"[^>]* style="([^"]+)"/)?.[1] ?? '';
		const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(style));
		const hash = `sha256-${btoa(String.fromCharCode(...new Uint8Array(digest)))}`;
		expect(config.kit?.csp?.directives?.['style-src']).toContain(hash);
	});
});
