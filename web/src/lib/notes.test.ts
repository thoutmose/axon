import { describe, expect, it } from 'vitest';
import { highlight, render, stem, wikilinks } from './notes';

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
		expect(html).toContain('<h2 id="h-title">Title</h2>');
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

describe('render extensions', () => {
	it('renders tables with alignment, escaping cells', () => {
		const html = render('| Name | Score |\n| :-- | --: |\n| <b>Ada</b> | 10 |', resolve);
		expect(html).toContain('<th>Name</th><th style="text-align:right">Score</th>');
		expect(html).toContain('<td>&lt;b&gt;Ada&lt;/b&gt;</td><td style="text-align:right">10</td>');
	});
	it('renders callouts, merged quotes, highlights, strikes, tags, and heading anchors', () => {
		const html = render(
			'> [!WARNING] Careful\n> Hot surface\n\n> line one\n> line two\n\n==key== ~~old~~ #maths not a#b\n## Big Idea!',
			resolve
		);
		expect(html).toContain(
			'<aside class="callout warning"><p class="callout-title">Careful</p><p>Hot surface</p></aside>'
		);
		expect(html).toContain('<blockquote>line one<br>line two</blockquote>');
		expect(html).toContain(
			'<mark>key</mark> <del>old</del> <a class="tag" href="/notes?tag=maths">#maths</a> not a#b'
		);
		expect(html).toContain('<h3 id="h-big-idea">Big Idea!</h3>');
	});
	it('ignores unknown callout kinds and escaped entities', () => {
		expect(render('> [!SCRIPT] x', resolve)).toContain('<blockquote>');
		expect(render("it's", resolve)).not.toContain('class="tag"');
	});
});

describe('titles for wikilinks', () => {
	it('shows the linked note title unless an alias is given', () => {
		const titleOf = (id: string) => ({ 'p-1': 'Proofs & <b>' })[id];
		const html = render('[[proofs]] and [[proofs|see here]]', resolve, titleOf);
		expect(html).toContain('href="/notes/p-1">Proofs &amp; &lt;b&gt;</a>');
		expect(html).toContain('href="/notes/p-1">see here</a>');
	});
});

describe('highlight', () => {
	it('escapes before marking matches', () => {
		expect(highlight('a \u0002<b>x</b>\u0003 c')).toBe('a <mark>&lt;b&gt;x&lt;/b&gt;</mark> c');
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
