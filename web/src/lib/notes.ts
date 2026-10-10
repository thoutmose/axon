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

/** Maps a lower-cased link target to a note id; unresolved targets are absent. */
export type Resolver = (target: string) => string | undefined;

// Inline Markdown on already-escaped text. Code spans are cut out first so
// their content is never interpreted.
function inline(text: string, resolve: Resolver): string {
	return text
		.split(/(`[^`]+`)/)
		.map((part) => {
			if (part.startsWith('`') && part.endsWith('`') && part.length > 1) {
				return `<code>${escape(part.slice(1, -1))}</code>`;
			}
			let s = escape(part);
			s = s.replace(wikilink, (_, target: string, heading = '', alias = '') => {
				const label = alias ? alias.slice(1) : target + heading;
				const id = resolve(target.trim().toLowerCase());
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
			return s;
		})
		.join('');
}

/**
 * Renders a note for reading. It covers what notes mostly use (front matter,
 * headings, lists, quotes, code, links, emphasis) and escapes everything
 * else, so a note can never inject markup.
 */
export function render(md: string, resolve: Resolver): string {
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
							resolve
						)}</dd>`
					: `<dd>${inline(l, resolve)}</dd>`;
			});
			out.push(`<dl class="props">${props.join('')}</dl>`);
			i = end + 1;
		}
	}
	let list: 'ul' | 'ol' | null = null;
	let para: string[] = [];
	const flush = () => {
		if (para.length) out.push(`<p>${para.map((l) => inline(l, resolve)).join('<br>')}</p>`);
		para = [];
		if (list) out.push(`</${list}>`);
		list = null;
	};
	for (; i < lines.length; i++) {
		const line = lines[i];
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
			out.push(`<h${level}>${inline(heading[2], resolve)}</h${level}>`);
		} else if (item) {
			const kind = item[1] ? 'ul' : 'ol';
			if (para.length || list !== kind) flush();
			if (!list) out.push(`<${kind}>`);
			list = kind;
			const task = item[3].match(/^\[([ xX])\]\s+(.*)$/);
			out.push(
				task
					? `<li>${task[1] === ' ' ? '☐' : '☑'} ${inline(task[2], resolve)}</li>`
					: `<li>${inline(item[3], resolve)}</li>`
			);
		} else if (quote) {
			flush();
			out.push(`<blockquote>${inline(quote[1], resolve)}</blockquote>`);
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
