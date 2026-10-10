import { describe, expect, it } from 'vitest';
import { render, stem, wikilinks } from './notes';

const ids: Record<string, string> = { proofs: 'p-1', 'mit 6.042j': 'm-1' };
const resolve = (t: string) => ids[t];

describe('render', () => {
	it('never lets a note inject markup', () => {
		const html = render(
			'<script>alert(1)</script>\n[x](javascript:alert(1))\n<img src=x onerror=alert(1)>\n[[<b>x</b>]]',
			resolve
		);
		expect(html).not.toMatch(/<script|<img|<b>|href="javascript/);
		expect(html).toContain('&lt;script&gt;');
	});

	it('turns wikilinks into note links and marks unresolved ones', () => {
		const html = render(
			'See [[proofs#induction|induction]] and [[MIT 6.042J]] and [[nowhere]].',
			resolve
		);
		expect(html).toContain('<a class="wikilink" href="/notes/p-1">induction</a>');
		expect(html).toContain('<a class="wikilink" href="/notes/m-1">MIT 6.042J</a>');
		expect(html).toContain(
			'<span class="wikilink unresolved" title="No note named nowhere yet">nowhere</span>'
		);
	});

	it('renders front matter, headings, lists, quotes, and code', () => {
		const html = render(
			'---\ntype: literature\narea: "[[proofs]]"\n---\n# Title\n- one\n- [x] two\n\n1. first\n> quote\n```\n**not bold**\n```\nA **bold** `*code*` *word*.',
			resolve
		);
		expect(html).toContain('<dt>type</dt><dd>literature</dd>');
		expect(html).toContain('<dd><a class="wikilink" href="/notes/p-1">proofs</a></dd>');
		expect(html).toContain('<h2>Title</h2>');
		expect(html).toContain('<ul>\n<li>one</li>\n<li>☑ two</li>\n</ul>');
		expect(html).toContain('<ol>\n<li>first</li>\n</ol>');
		expect(html).toContain('<blockquote>quote</blockquote>');
		expect(html).toContain('<pre><code>**not bold**</code></pre>');
		expect(html).toContain('A <strong>bold</strong> <code>*code*</code> <em>word</em>.');
	});

	it('keeps only safe link targets', () => {
		expect(render('[a](https://example.org)', resolve)).toContain(
			'<a href="https://example.org" rel="noopener noreferrer" target="_blank">a</a>'
		);
		expect(render('[a](/tasks)', resolve)).toContain('<a href="/tasks">a</a>');
		expect(render('[a](data:text/html,x)', resolve)).not.toContain('<a');
	});
});

describe('wikilinks', () => {
	it('lists distinct targets, case-insensitively', () => {
		expect(wikilinks('[[A]] [[a#h]] [[b|B]]')).toEqual(['A', 'b']);
	});
});

describe('stem', () => {
	it('drops folders and the extension', () => {
		expect(stem('Zettel/permanent/proofs.md')).toBe('proofs');
	});
});
