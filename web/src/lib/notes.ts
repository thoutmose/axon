import katex from 'katex';
import { api, type Schemas } from './api';

export type Note = Schemas['Note'];
export type Para = Schemas['NotePara'];

export const paraViews: { id: Para | 'all'; label: string }[] = [
	{ id: 'all', label: 'All' },
	{ id: 'inbox', label: 'Inbox' },
	{ id: 'projects', label: 'Projects' },
	{ id: 'areas', label: 'Areas' },
	{ id: 'resources', label: 'Resources' },
	{ id: 'archive', label: 'Archive' },
	{ id: 'other', label: 'Other' }
];

export function createNote(content: string, path?: string): Promise<Note> {
	return api<Note>('/v1/notes', { method: 'POST', body: path ? { path, content } : { content } });
}

/** The file name without folder or .md: the name [[wikilinks]] use. */
export function stem(path: string): string {
	return path.replace(/^.*\//, '').replace(/\.md$/, '');
}

const escapes: Record<string, string> = {
	'&': '&amp;',
	'<': '&lt;',
	'>': '&gt;',
	'"': '&quot;',
	"'": '&#39;'
};
function escape(s: string): string {
	return s.replace(/[&<>"']/g, (c) => escapes[c]);
}

const wikilink = /\[\[([^[\]|#\n]+)(#[^[\]|\n]*)?(\|[^[\]\n]*)?\]\]/g;

/** Distinct [[wikilink]] targets, as the API resolves them. */
export function wikilinks(md: string): string[] {
	const seen = new Map<string, string>();
	for (const m of md.matchAll(wikilink)) {
		const t = m[1].trim();
		if (t && !seen.has(t.toLowerCase())) seen.set(t.toLowerCase(), t);
	}
	return [...seen.values()];
}

/** Escapes a search snippet, then marks the matched words (U+0002 … U+0003). */
export function highlight(snippet: string): string {
	return escape(snippet)
		.replace(/\u0002/g, '<mark>')
		.replace(/\u0003/g, '</mark>');
}

/** #tag after a space or the start of a line, with at least one letter (as the API parses it). */
const tagPattern = /(^|[\s(])#([\p{L}\p{N}_/-]*\p{L}[\p{L}\p{N}_/-]*)/gu;

const calloutKinds = new Set([
	'note',
	'tip',
	'info',
	'important',
	'warning',
	'caution',
	'danger',
	'quote',
	'example',
	'question',
	'todo',
	'success'
]);

/** Heading anchor: letters, digits, and dashes only. */
function anchor(text: string): string {
	return (
		'h-' +
		text
			.toLowerCase()
			.replace(/[^\p{L}\p{N}]+/gu, '-')
			.replace(/^-|-$/g, '')
			.slice(0, 60)
	);
}

function cells(row: string): string[] {
	return row
		.trim()
		.replace(/^\||\|$/g, '')
		.split('|')
		.map((c) => c.trim());
}

/** Maps a lower-cased link target to a note id; unresolved targets are absent. */
export type Resolver = (target: string) => string | undefined;

/** Maps an embed target (a vault path or a file name) to the stored file. */
export type FileResolver = (target: string) => { path: string; content_type: string } | undefined;

const fileEmbed = /!\[\[([^[\]|\n]+\.(?:png|jpe?g|gif|webp|svg|pdf))(?:\|([^[\]\n]*))?\]\]/gi;
const imageLink = /!\[([^\]\n]*)\]\(&lt;([^\n]+?)&gt;\)|!\[([^\]\n]*)\]\(([^\s)]+)\)/g;

// Pictures embedded with ![[x.png|300]] (Obsidian) or ![alt](path). Text is
// already escaped, so alt text and paths are safe in attributes; external
// pictures become links, since the page only loads its own.
// Each embed becomes a placeholder until the other inline rules have run, so
// they never touch its markup.
function embeds(s: string, held: string[], file?: FileResolver): string {
	const unescape = (t: string) =>
		t.replace(
			/&(amp|lt|gt|quot|#39);/g,
			(m) => Object.keys(escapes).find((k) => escapes[k] === m) ?? m
		);
	const show = (target: string, alt: string, width = '') => {
		const f = file?.(unescape(target));
		if (!f) {
			return `<span class="embed missing" title="No file at ${target} yet">🖼 ${alt || target}</span>`;
		}
		const url = `/api/v1/notes/file?path=${encodeURIComponent(f.path)}`;
		if (f.content_type === 'application/pdf') {
			return `<a class="embed pdf" href="${url}" download>📄 ${alt || escape(f.path.replace(/^.*\//, ''))}</a>`;
		}
		const w = /^\d{1,4}(x\d{1,4})?$/.test(width) ? ` width="${width.split('x')[0]}"` : '';
		return `<img class="embed" src="${url}" alt="${alt}"${w} loading="lazy">`;
	};
	const hold = (html: string) => `\u0001${held.push(html) - 1}\u0001`;
	s = s.replace(fileEmbed, (_, target: string, opt = '') =>
		hold(show(target, /^\d/.test(opt) ? '' : opt, opt))
	);
	return s.replace(imageLink, (_, alt1, target1, alt2, target2) => {
		const alt = alt1 ?? alt2 ?? '';
		const target = target1 ?? target2;
		if (/^https?:/.test(target)) {
			return hold(
				`<a href="${target}" rel="noopener noreferrer" target="_blank">🖼 ${alt || 'picture'}</a>`
			);
		}
		const [label, width] = alt.split('|');
		return hold(show(target, label, width?.trim()));
	});
}

// Maths (EP-150), written as in Obsidian: $…$ inline, $$…$$ as a block.
// KaTeX writes MathML, which browsers draw themselves, so no inline style is
// needed; it escapes what it is given, and trust is off, so a formula can
// neither link nor load anything. An opening $ is followed and a closing $
// preceded by a non-space, and a closing $ is not followed by a digit, so
// prices such as "$5 and $10" stay text; maths never runs into code.
const inlineMath = /(\$(?=[^\s$`])[^$\n`]*?[^\s$\\`]\$(?!\d)|\$[^\s$`]\$(?!\d))/;

export function math(tex: string, display: boolean): string {
	return katex.renderToString(tex, {
		output: 'mathml',
		displayMode: display,
		throwOnError: false,
		trust: false,
		strict: 'ignore',
		maxSize: 50,
		maxExpand: 500
	});
}

// Inline Markdown on already-escaped text. Code spans and maths are cut out
// first so their content is never interpreted.
function inline(
	text: string,
	resolve: Resolver,
	titleOf?: (id: string) => string | undefined,
	file?: FileResolver
): string {
	return text
		.split(new RegExp(`(\`[^\`]+\`|${inlineMath.source.slice(1, -1)})`))
		.map((part) => {
			if (part.startsWith('`') && part.endsWith('`') && part.length > 1) {
				return `<code>${escape(part.slice(1, -1))}</code>`;
			}
			if (inlineMath.test(part) && part.startsWith('$') && part.endsWith('$')) {
				return math(part.slice(1, -1), false);
			}
			const held: string[] = [];
			let s = escape(part.replace(/\u0001/g, ''));
			s = embeds(s, held, file);
			s = s.replace(wikilink, (_, target: string, heading = '', alias = '') => {
				const id = resolve(target.trim().toLowerCase());
				// A plain [[file-name]] reads as the note's title; an alias wins.
				const named = id && titleOf ? titleOf(id) : undefined;
				const label = alias ? alias.slice(1) : named ? escape(named) + heading : target + heading;
				return id
					? `<a class="wikilink" href="/notes/${encodeURIComponent(id)}">${label}</a>`
					: `<span class="wikilink unresolved" title="No note named ${target} yet">${label}</span>`;
			});
			s = s.replace(/\[([^\]]+)\]\(((?:https?:\/\/|\/|#)[^\s)]*)\)/g, (_, label, url) => {
				const external = url.startsWith('http');
				return `<a href="${url}"${external ? ' rel="noopener noreferrer" target="_blank"' : ''}>${label}</a>`;
			});
			s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
			s = s.replace(/(^|[^*\w])\*([^*\s][^*]*)\*/g, '$1<em>$2</em>');
			s = s.replace(/~~([^~]+)~~/g, '<del>$1</del>');
			s = s.replace(/==([^=]+)==/g, '<mark>$1</mark>');
			s = s.replace(tagPattern, (_, pre: string, tag: string) => {
				return `${pre}<a class="tag" href="/notes?tag=${encodeURIComponent(tag.toLowerCase())}">#${tag}</a>`;
			});
			return s.replace(/\u0001(\d+)\u0001/g, (_, n) => held[Number(n)]);
		})
		.join('');
}

/**
 * Renders a note for reading. It covers what notes mostly use (front matter,
 * headings, lists, quotes, code, links, emphasis) and escapes everything
 * else, so a note can never inject markup.
 */
export function render(
	md: string,
	resolve: Resolver,
	titleOf?: (id: string) => string | undefined,
	file?: FileResolver
): string {
	const out: string[] = [];
	const lines = md.replace(/\r\n?/g, '\n').split('\n');
	let i = 0;
	if (lines[0] === '---') {
		const end = lines.indexOf('---', 1);
		if (end > 0) {
			const props = lines.slice(1, end).map((l) => {
				const [k, ...v] = l.split(':');
				return v.length
					? `<dt>${escape(k.trim())}</dt><dd>${inline(
							v
								.join(':')
								.trim()
								.replace(/^"(.*)"$/, '$1'),
							resolve,
							titleOf,
							file
						)}</dd>`
					: `<dd>${inline(l, resolve, titleOf, file)}</dd>`;
			});
			out.push(`<dl class="props">${props.join('')}</dl>`);
			i = end + 1;
		}
	}
	let list: 'ul' | 'ol' | null = null;
	let para: string[] = [];
	const flush = () => {
		if (para.length)
			out.push(`<p>${para.map((l) => inline(l, resolve, titleOf, file)).join('<br>')}</p>`);
		para = [];
		if (list) out.push(`</${list}>`);
		list = null;
	};
	for (; i < lines.length; i++) {
		const line = lines[i];
		// A block of maths: $$ on its own lines around it, or on one line.
		const oneLine = line.match(/^\s*\$\$(.+)\$\$\s*$/);
		if (oneLine) {
			flush();
			out.push(`<div class="math">${math(oneLine[1], true)}</div>`);
			continue;
		}
		if (/^\s*\$\$\s*$/.test(line)) {
			flush();
			const body: string[] = [];
			for (i++; i < lines.length && !/^\s*\$\$\s*$/.test(lines[i]); i++) body.push(lines[i]);
			out.push(`<div class="math">${math(body.join('\n'), true)}</div>`);
			continue;
		}
		const fence = line.match(/^```/);
		if (fence) {
			flush();
			const body: string[] = [];
			for (i++; i < lines.length && !lines[i].startsWith('```'); i++) body.push(lines[i]);
			out.push(`<pre><code>${escape(body.join('\n'))}</code></pre>`);
			continue;
		}
		const heading = line.match(/^(#{1,6})\s+(.*)$/);
		const item = line.match(/^\s*(?:([-*+])|(\d+)[.)])\s+(.*)$/);
		const quote = line.match(/^>\s?(.*)$/);
		if (heading) {
			flush();
			// The page title is the h1, so note headings start at h2.
			const level = Math.min(heading[1].length + 1, 6);
			out.push(
				`<h${level} id="${anchor(heading[2])}">${inline(heading[2], resolve, titleOf, file)}</h${level}>`
			);
		} else if (item) {
			const kind = item[1] ? 'ul' : 'ol';
			if (para.length || list !== kind) flush();
			if (!list) out.push(`<${kind}>`);
			list = kind;
			const task = item[3].match(/^\[([ xX])\]\s+(.*)$/);
			out.push(
				task
					? `<li>${task[1] === ' ' ? '☐' : '☑'} ${inline(task[2], resolve, titleOf, file)}</li>`
					: `<li>${inline(item[3], resolve, titleOf, file)}</li>`
			);
		} else if (quote) {
			flush();
			const body: string[] = [];
			for (; i < lines.length && /^>/.test(lines[i]); i++) body.push(lines[i].replace(/^>\s?/, ''));
			i--;
			const callout = body[0].match(/^\[!(\w+)\][+-]?\s*(.*)$/);
			if (callout && calloutKinds.has(callout[1].toLowerCase())) {
				const kind = callout[1].toLowerCase();
				const title = callout[2] || kind[0].toUpperCase() + kind.slice(1);
				const rest = body.slice(1).filter((l) => l.trim() !== '');
				out.push(
					`<aside class="callout ${kind}"><p class="callout-title">${inline(title, resolve, titleOf, file)}</p>${rest
						.map((l) => `<p>${inline(l, resolve, titleOf, file)}</p>`)
						.join('')}</aside>`
				);
			} else {
				out.push(
					`<blockquote>${body.map((l) => inline(l, resolve, titleOf, file)).join('<br>')}</blockquote>`
				);
			}
		} else if (/^\s*\|/.test(line) && /^\s*\|?\s*:?-{2,}/.test(lines[i + 1] ?? '')) {
			flush();
			const head = cells(line);
			const align = cells(lines[i + 1]).map((c) =>
				c.startsWith(':') && c.endsWith(':') ? 'center' : c.endsWith(':') ? 'right' : ''
			);
			const rows: string[][] = [];
			for (i += 2; i < lines.length && /^\s*\|/.test(lines[i]); i++) rows.push(cells(lines[i]));
			i--;
			const td = (tag: string, c: string, k: number) =>
				`<${tag}${align[k] ? ` class="align-${align[k]}"` : ''}>${inline(c, resolve, titleOf, file)}</${tag}>`;
			out.push(
				`<div class="table"><table><thead><tr>${head.map((c, k) => td('th', c, k)).join('')}</tr></thead><tbody>${rows
					.map((r) => `<tr>${r.map((c, k) => td('td', c, k)).join('')}</tr>`)
					.join('')}</tbody></table></div>`
			);
		} else if (/^\s*(---|\*\*\*)\s*$/.test(line)) {
			flush();
			out.push('<hr>');
		} else if (line.trim() === '') {
			flush();
		} else {
			if (list) flush();
			para.push(line);
		}
	}
	flush();
	return out.join('\n');
}
