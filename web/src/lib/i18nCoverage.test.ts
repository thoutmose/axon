import { describe, expect, it } from 'vitest';
import { fr } from './i18n.fr';
import * as entityTemplates from './entityTemplates';
import * as finance from './finance';
import * as flashcards from './flashcards';
import * as habits from './habits';
import { mealNames, restrictionNames } from './mealPlan';
import * as links from './links';
import * as notes from './notes';
import { templates as noteTemplates } from './noteTemplates';
import * as para from './para';
import * as tasks from './tasks';

// French for every page (IS-31.06): text in a template must go through t().
// `untranslated` finds what does not: text between tags, and words in the
// attributes people read (placeholder, aria-label, title, label, alt, message,
// and the empty, hint, and description props of Axon's components).

const templates = import.meta.glob('/src/**/*.svelte', {
	query: '?raw',
	import: 'default',
	eager: true
}) as Record<string, string>;

/** Names, units, and symbols that read the same in both languages. */
const same = new Set([
	'Axon',
	'OSSU',
	'roadmap.sh',
	'GitHub',
	'Anki',
	'YAML',
	'Markdown',
	'min',
	'km',
	'kg',
	'kcal',
	// Each language is named in its own.
	'English',
	'Français'
]);

/** Attributes whose values are code, not words. */
const technical =
	/^(?:class|style|href|src|id|for|type|name|value|key|d|points|transform|viewBox|role|lang|srclang|accept|autocomplete|inputmode|pattern|(?:on|bind:|use:|class:|style:|transition:|animate:|in:|out:)[\w-]*)$/;

/** Replaces every {…} expression, nested braces included, with ⟨⟩, and
 * collects the expressions that are not technical attribute values. */
function blankExpressions(src: string, expressions: string[] = []): string {
	let out = '';
	let depth = 0;
	let expr = '';
	let attr = '';
	for (const ch of src) {
		if (ch === '{') {
			if (depth++ === 0) {
				attr = out.match(/([\w:-]+)=$/)?.[1] ?? '';
				out += '⟨⟩';
				expr = '';
				continue;
			}
		} else if (ch === '}' && depth > 0) {
			if (--depth === 0) {
				if (!technical.test(attr)) expressions.push(expr);
				continue;
			}
		} else if (depth === 0) {
			out += ch;
			continue;
		}
		expr += ch;
	}
	return out;
}

/** Removes t(…) calls, so only words outside them are left. */
function withoutT(expr: string): string {
	let out = '';
	for (let i = 0; i < expr.length; i++) {
		if (/\bt\($/.test(expr.slice(Math.max(0, i - 2), i + 1)) && !/[\w$.]/.test(expr[i - 2] ?? '')) {
			out = out.slice(0, -1);
			let depth = 1;
			while (depth && ++i < expr.length) depth += expr[i] === '(' ? 1 : expr[i] === ')' ? -1 : 0;
			continue;
		}
		out += expr[i];
	}
	return out;
}

/** Text people read: words with a lower-case letter (not YYYY-MM-DD), not a path or URL. */
const sentence = (text: string) =>
	/[a-z]/.test(text) &&
	/[A-Za-z]{2,}/.test(text) &&
	/\s\S|^[A-Z]/.test(text.trim()) &&
	!/^[/#.]|^https?:/.test(text.trim());

/** String literals in an expression that people would read; a template
 * literal counts by its own text, and its \${…} parts are read in turn. */
function words(expr: string, found: string[] = []): string[] {
	const code = withoutT(expr);
	let i = 0;
	const skipTo = (end: string) => {
		let text = '';
		while (i < code.length && code[i] !== end) {
			if (code[i] === '\\') i++;
			text += code[i++];
		}
		i++;
		return text;
	};
	while (i < code.length) {
		const ch = code[i];
		const compared = /(===?|!==?)\s*$/.test(code.slice(0, i));
		if (ch === "'" || ch === '"') {
			i++;
			const text = skipTo(ch);
			if (!compared && sentence(text)) found.push(text.trim());
		} else if (ch === '`') {
			i++;
			let text = '';
			while (i < code.length && code[i] !== '`') {
				if (code.startsWith('\${', i)) {
					let depth = 1;
					const start = (i += 2);
					for (; depth && i < code.length; i++)
						depth += code[i] === '{' ? 1 : code[i] === '}' ? -1 : 0;
					if (code[i - 1] === '}' && depth === 0) words(code.slice(start, i - 1), found);
					text += ' ';
				} else text += code[i++];
			}
			i++;
			if (!compared && sentence(text)) found.push(text.replace(/\s+/g, ' ').trim());
		} else i++;
	}
	return found;
}

const readable = (s: string) => {
	const words = s
		.replace(/⟨⟩/g, ' ')
		.replace(/&[a-z]+;|&#\d+;/g, ' ')
		.split(/\s+/)
		.filter((w) => !same.has(w) && !/^https?:/.test(w));
	return /[A-Za-z]{2,}/.test(words.join(' ')) ? s.replace(/\s+/g, ' ').trim() : null;
};

/** The English left in a component's markup. */
export function untranslated(src: string): string[] {
	const expressions: string[] = [];
	const markup = blankExpressions(
		src
			.replace(/<script[\s\S]*?<\/script>/g, '')
			.replace(/<style[\s\S]*?<\/style>/g, '')
			.replace(/<!--[\s\S]*?-->/g, ''),
		expressions
	).replace(/<(code|kbd|pre)\b[^>]*>[\s\S]*?<\/\1>/g, '');
	const found: string[] = [];
	for (const m of markup.matchAll(/>([^<>]+)</g)) {
		const s = readable(m[1]);
		if (s) found.push(s);
	}
	for (const m of markup.matchAll(
		/\s(?:placeholder|aria-label|title|label|alt|message|empty|hint|description)="([^"]*)"/g
	)) {
		const s = readable(m[1]);
		if (s) found.push(s);
	}
	for (const e of expressions) found.push(...words(e).filter((w) => readable(w)));
	return found;
}

describe('French for every page', () => {
	it('finds text and readable attributes outside t()', () => {
		expect(
			untranslated(`<script>const a = 'x';</script>
<h1>{t('Tasks')}</h1>
<p>Plain words</p>
<input placeholder="Search…" aria-label={t('Search')} />
<span>{count} · {t('done')}</span>
<code>git status</code>
<p title="Progress of {r.title}">Axon</p>
<b class={\`card \${x}\`}>{done ? 'All done' : t('Not yet')}</b>
<i>{kind === 'Big one' ? t(a ? 'A' : 'B') : name}</i>
<style>p { color: red; }</style>`)
		).toEqual(['Plain words', 'Search…', 'Progress of ⟨⟩', 'All done']);
	});

	it('leaves no English in any page', () => {
		const left = Object.entries(templates)
			.map(([path, src]) => [path, untranslated(src)] as const)
			.filter(([, found]) => found.length);
		expect(Object.fromEntries(left)).toEqual({});
	});
});

// Names, codes, and examples that read the same in French.
const unchanged = new Set([
	'OpenAI',
	'Voyage AI',
	'Markdown',
	'Axon',
	'Zettelkasten',
	'fr',
	'Anthropic (Claude)',
	'Mistral',
	'Google Gemini',
	'DeepSeek',
	'RRULE',
	'Todoist',
	'TickTick',
	'Goodreads',
	'Letterboxd',
	'Firefly III',
	'Notion',
	'carrefour',
	'.apkg',
	'kcal',
	'Open Food Facts',
	'USDA FoodData Central',
	'ABCD-EFGH-JKLM-NPQR',
	'Open Library',
	'TMDB',
	'Open English WordNet',
	'PARA',
	'Zettel/permanent/proofs.md',
	'xxxx-xxxx-xxxx-xxxx',
	'2-Areas',
	'3-Resources',
	'Ctrl K',
	'Shift ← → ↑ ↓',
	'Ctrl B / I / E / L',
	'Esc · i a o',
	'h j k l · w b e · 0 $ · gg G',
	'd c y + motion · dd cc yy',
	'x p P u Ctrl R'
]);

const sources = import.meta.glob('/src/**/*.{svelte,ts}', {
	query: '?raw',
	import: 'default',
	eager: true
}) as Record<string, string>;

/** The literal keys code passes to t(), both sides of a ternary included. */
function keysInCode(): Set<string> {
	const keys = new Set<string>();
	const unescape = (s: string) => s.replace(/\\(['"])/g, '$1').replace(/\\n/g, '\n');
	for (const [path, src] of Object.entries(sources)) {
		if (path.includes('.test.') || path.endsWith('i18n.fr.ts')) continue;
		for (const m of src.matchAll(/(?<![\w$.])t\(\s*(['"])((?:\\.|(?!\1).)*)\1/g))
			keys.add(unescape(m[2]));
		for (const m of src.matchAll(/(?<![\w$.])t\(([^()]*\?[^()]*)\)/g))
			for (const l of m[1].matchAll(/(['"])((?:\\.|(?!\1).)*)\1/g)) keys.add(unescape(l[2]));
	}
	return keys;
}

/** Labels in Axon's own tables that pages show through t(). */
function keysInTables(): Set<string> {
	const keys = new Set<string>();
	const fields = [
		'label',
		'title',
		'short',
		'long',
		'law',
		'prompt',
		'hint',
		'outcome',
		'description',
		'empty',
		'intro',
		'path',
		'identity'
	];
	const walk = (v: unknown, field = '') => {
		if (field === 'raw') return; // a vault template's own file, written in its owner's language
		if (typeof v === 'string') {
			if (fields.includes(field) || (/[a-z]/.test(v) && /\s/.test(v) && !/^https?:|^\//.test(v)))
				keys.add(v);
		} else if (Array.isArray(v)) v.forEach((x) => walk(x, field));
		else if (v && typeof v === 'object') for (const [k, x] of Object.entries(v)) walk(x, k);
	};
	walk({
		noteTemplates,
		restrictionNames,
		mealNames,
		entityTemplates,
		finance,
		flashcards,
		habits,
		notes,
		para,
		types: links.typeNames,
		rec: tasks.recurrencePresets
	});
	return keys;
}

describe('French for every string', () => {
	it('has a translation for every key the code uses', () => {
		const missing = [...keysInCode(), ...keysInTables()].filter(
			(k) => /[A-Za-z]{2,}/.test(k) && !(k in fr) && !unchanged.has(k)
		);
		expect([...new Set(missing)]).toEqual([]);
	});
});
