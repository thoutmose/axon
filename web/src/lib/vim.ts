/**
 * A small Vim engine for the note editor (EP-32). It is pure: it takes the
 * text and cursor and returns the new ones, so it is unit-tested without a
 * browser. In insert mode the textarea edits natively; the engine only
 * handles Escape. Supported: counts; h j k l w b e 0 ^ $ gg G; i a I A o O;
 * x X D C J p P u Ctrl-r; d c y with a motion, dd cc yy; v and V with d c y x;
 * and :w, :wq, :x to save.
 */

export type Mode = 'normal' | 'insert' | 'visual' | 'visual-line';

export type Vim = {
	mode: Mode;
	pending: string; // operator, count, or command line typed so far
	anchor: number; // visual start
	register: string;
	linewise: boolean;
	undo: Snapshot[];
	redo: Snapshot[];
};

type Snapshot = { text: string; cursor: number };

export type Result = {
	text: string;
	cursor: number;
	selection?: [number, number]; // visual mode highlight, end exclusive
	handled: boolean; // the key was consumed; the textarea must not see it
	save?: boolean;
};

export function newVim(): Vim {
	return {
		mode: 'normal',
		pending: '',
		anchor: 0,
		register: '',
		linewise: false,
		undo: [],
		redo: []
	};
}

const lineStart = (t: string, i: number) => t.lastIndexOf('\n', i - 1) + 1;
const lineEnd = (t: string, i: number) => {
	const n = t.indexOf('\n', i);
	return n === -1 ? t.length : n;
};
/** The last character of the line (normal mode never rests on the newline). */
const lastChar = (t: string, i: number) => Math.max(lineStart(t, i), lineEnd(t, i) - 1);
const clampNormal = (t: string, i: number) =>
	Math.min(Math.max(0, i), lastChar(t, Math.min(i, t.length)));

function vertical(t: string, i: number, lines: number): number {
	const col = i - lineStart(t, i);
	let start = lineStart(t, i);
	for (let n = 0; n < Math.abs(lines); n++) {
		if (lines > 0) {
			const end = lineEnd(t, start);
			if (end >= t.length) break;
			start = end + 1;
		} else {
			if (start === 0) break;
			start = lineStart(t, start - 1);
		}
	}
	return Math.min(start + col, lastChar(t, start));
}

const isSpace = (c: string | undefined) => c === undefined || /\s/.test(c);

function wordForward(t: string, i: number): number {
	let j = i;
	while (j < t.length && !isSpace(t[j])) j++;
	while (j < t.length && isSpace(t[j])) j++;
	return j;
}
function wordBack(t: string, i: number): number {
	let j = i - 1;
	while (j > 0 && isSpace(t[j])) j--;
	while (j > 0 && !isSpace(t[j - 1])) j--;
	return Math.max(0, j);
}
function wordEnd(t: string, i: number): number {
	let j = i + 1;
	while (j < t.length && isSpace(t[j])) j++;
	while (j < t.length - 1 && !isSpace(t[j + 1])) j++;
	return Math.min(j, t.length - 1);
}

type Motion = { to: number; inclusive?: boolean; linewise?: boolean };

function motion(t: string, i: number, m: string, count: number): Motion | null {
	let to = i;
	switch (m) {
		case 'h':
			return { to: Math.max(lineStart(t, i), i - count) };
		case 'l':
			return { to: Math.min(lineEnd(t, i), i + count) };
		case 'j':
			return { to: vertical(t, i, count), linewise: true };
		case 'k':
			return { to: vertical(t, i, -count), linewise: true };
		case 'w':
			for (let n = 0; n < count; n++) to = wordForward(t, to);
			return { to };
		case 'b':
			for (let n = 0; n < count; n++) to = wordBack(t, to);
			return { to };
		case 'e':
			for (let n = 0; n < count; n++) to = wordEnd(t, to);
			return { to, inclusive: true };
		case '0':
			return { to: lineStart(t, i) };
		case '^': {
			let j = lineStart(t, i);
			while (j < lineEnd(t, i) && /[ \t]/.test(t[j])) j++;
			return { to: j };
		}
		case '$':
			return { to: lastChar(t, i), inclusive: true };
		case 'gg':
			return { to: 0, linewise: true };
		case 'G':
			return { to: lineStart(t, t.length), linewise: true };
	}
	return null;
}

const motions = ['h', 'l', 'j', 'k', 'w', 'b', 'e', '0', '^', '$', 'G'];

function snapshot(v: Vim, text: string, cursor: number) {
	v.undo.push({ text, cursor });
	if (v.undo.length > 200) v.undo.shift();
	v.redo = [];
}

/** The [start, end) range of whole lines from a to b, newline included. */
function lineRange(t: string, a: number, b: number): [number, number] {
	const s = lineStart(t, Math.min(a, b));
	let e = lineEnd(t, Math.max(a, b));
	if (e < t.length) e++;
	else if (s > 0) return [s - 1, e]; // last line: take the newline before it
	return [s, e];
}

export function handle(v: Vim, key: string, text: string, cursor: number): Result {
	const done = (t: string, c: number, extra: Partial<Result> = {}): Result => ({
		text: t,
		cursor: v.mode === 'insert' ? c : clampNormal(t, c),
		handled: true,
		...extra
	});

	if (v.mode === 'insert') {
		if (key !== 'Escape') return { text, cursor, handled: false };
		v.mode = 'normal';
		return done(text, cursor > lineStart(text, cursor) ? cursor - 1 : cursor);
	}

	// Command line: ":w" saves.
	if (v.pending.startsWith(':')) {
		if (key === 'Escape') v.pending = '';
		else if (key === 'Enter') {
			const cmd = v.pending.slice(1).trim();
			v.pending = '';
			return done(text, cursor, { save: ['w', 'wq', 'x'].includes(cmd) });
		} else if (key === 'Backspace') v.pending = v.pending.slice(0, -1);
		else if (key.length === 1) v.pending += key;
		return done(text, cursor);
	}

	if (key === 'Escape') {
		v.pending = '';
		v.mode = 'normal';
		return done(text, cursor);
	}
	if (key.length > 1 && key !== 'C-r') return { text, cursor, handled: false }; // arrows, Tab…

	// Visual mode.
	if (v.mode === 'visual' || v.mode === 'visual-line') {
		const m = motion(text, cursor, key === 'g' ? '' : key, 1);
		if (key === 'g' || v.pending === 'g') {
			v.pending = v.pending === 'g' ? '' : 'g';
			if (key === 'g' && v.pending === '') return visualResult(v, text, 0);
			return done(text, cursor, { selection: sel(v, text, cursor) });
		}
		if (m) return visualResult(v, text, m.to);
		if (['d', 'x', 'y', 'c'].includes(key)) {
			const [s, e] = sel(v, text, cursor);
			v.register = text.slice(s, e);
			v.linewise = v.mode === 'visual-line';
			if (key === 'y') {
				v.mode = 'normal';
				return done(text, s);
			}
			snapshot(v, text, cursor);
			const t = text.slice(0, s) + text.slice(e);
			v.mode = key === 'c' ? 'insert' : 'normal';
			return done(t, s);
		}
		return done(text, cursor, { selection: sel(v, text, cursor) });
	}

	// Normal mode: counts, then operators and motions.
	if (/[1-9]/.test(key) || (key === '0' && /\d$/.test(v.pending))) {
		v.pending += key;
		return done(text, cursor);
	}
	// pending is [count][operator][count][g]; the key completes it.
	const parsed = v.pending.match(/^(\d*)([dcy]?)(\d*)(g?)$/);
	if (!parsed) {
		v.pending = '';
		return done(text, cursor);
	}
	const count = Math.max(1, Number(parsed[1] || 1) * Number(parsed[3] || 1));
	const op = parsed[2];
	const rest = parsed[4] + key;
	const seq = v.pending + key;
	if (!op && !parsed[4] && ['d', 'c', 'y'].includes(key)) {
		v.pending = seq; // an operator waits for its motion
		return done(text, cursor);
	}

	if (op) {
		if (rest === 'g') {
			v.pending = seq;
			return done(text, cursor);
		}
		v.pending = '';
		let range: [number, number] | null = null;
		let linewise = false;
		if (rest === op) {
			range = lineRange(text, cursor, vertical(text, cursor, count - 1));
			linewise = true;
		} else {
			const m = motion(text, cursor, rest, count);
			if (!m) return done(text, cursor);
			if (m.linewise) {
				range = lineRange(text, cursor, m.to);
				linewise = true;
			} else {
				const a = Math.min(cursor, m.to);
				let b = Math.max(cursor, m.to) + (m.inclusive ? 1 : 0);
				if (op === 'c' && rest === 'w') b = Math.min(b, wordEnd(text, cursor - 1) + 1); // cw acts like ce
				range = [a, Math.min(b, text.length)];
			}
		}
		const [s, e] = range;
		v.register = text.slice(s, e);
		v.linewise = linewise;
		if (op === 'y') return done(text, linewise ? lineStart(text, Math.min(cursor, s + 1)) : s);
		snapshot(v, text, cursor);
		if (op === 'c' && linewise) {
			// cc keeps an empty line to type on.
			const keep =
				text.slice(0, s) +
				(s > 0 && e === text.length ? '\n' : '') +
				(e < text.length ? '\n' : '') +
				text.slice(e);
			v.mode = 'insert';
			const at = s > 0 && e === text.length ? s + 1 : s;
			return done(keep, at);
		}
		const t = text.slice(0, s) + text.slice(e);
		if (op === 'c') v.mode = 'insert';
		return done(t, linewise ? Math.min(s, t.length) : s);
	}

	if (rest === 'g') {
		v.pending = seq;
		return done(text, cursor);
	}
	v.pending = '';
	if (rest === 'gg' || motions.includes(rest)) {
		const m = motion(text, cursor, rest, count)!;
		return done(text, rest === 'G' && parsed[1] ? vertical(text, 0, count - 1) : m.to);
	}

	const enterInsert = (t: string, c: number) => {
		snapshot(v, text, cursor);
		v.mode = 'insert';
		return { text: t, cursor: c, handled: true } as Result;
	};
	switch (rest) {
		case 'i':
			return enterInsert(text, cursor);
		case 'a':
			return enterInsert(text, Math.min(cursor + 1, lineEnd(text, cursor)));
		case 'I':
			return enterInsert(text, motion(text, cursor, '^', 1)!.to);
		case 'A':
			return enterInsert(text, lineEnd(text, cursor));
		case 'o': {
			const e = lineEnd(text, cursor);
			return enterInsert(text.slice(0, e) + '\n' + text.slice(e), e + 1);
		}
		case 'O': {
			const s = lineStart(text, cursor);
			return enterInsert(text.slice(0, s) + '\n' + text.slice(s), s);
		}
		case 'x':
		case 'X': {
			const s = rest === 'x' ? cursor : Math.max(lineStart(text, cursor), cursor - count);
			const e = rest === 'x' ? Math.min(lineEnd(text, cursor), cursor + count) : cursor;
			if (s === e) return done(text, cursor);
			snapshot(v, text, cursor);
			v.register = text.slice(s, e);
			v.linewise = false;
			return done(text.slice(0, s) + text.slice(e), s);
		}
		case 'D':
		case 'C': {
			const e = lineEnd(text, cursor);
			snapshot(v, text, cursor);
			v.register = text.slice(cursor, e);
			v.linewise = false;
			if (rest === 'C') v.mode = 'insert';
			return done(text.slice(0, cursor) + text.slice(e), cursor);
		}
		case 'J': {
			const e = lineEnd(text, cursor);
			if (e >= text.length) return done(text, cursor);
			snapshot(v, text, cursor);
			let n = e + 1;
			while (n < text.length && /[ \t]/.test(text[n])) n++;
			return done(text.slice(0, e) + ' ' + text.slice(n), e);
		}
		case 'p':
		case 'P': {
			if (!v.register) return done(text, cursor);
			snapshot(v, text, cursor);
			const r = v.register.repeat(count);
			if (v.linewise) {
				const block = r.endsWith('\n') ? r : r + '\n';
				const at =
					rest === 'p' ? Math.min(lineEnd(text, cursor) + 1, text.length) : lineStart(text, cursor);
				const pre = rest === 'p' && at === text.length && !text.endsWith('\n') ? '\n' : '';
				const body = pre ? block.replace(/\n$/, '') : block;
				return done(text.slice(0, at) + pre + body + text.slice(at), at + pre.length);
			}
			const at = rest === 'p' ? Math.min(cursor + 1, lineEnd(text, cursor)) : cursor;
			return done(text.slice(0, at) + r + text.slice(at), at + r.length - 1);
		}
		case 'u': {
			const prev = v.undo.pop();
			if (!prev) return done(text, cursor);
			v.redo.push({ text, cursor });
			return done(prev.text, prev.cursor);
		}
		case 'C-r': {
			const next = v.redo.pop();
			if (!next) return done(text, cursor);
			v.undo.push({ text, cursor });
			return done(next.text, next.cursor);
		}
		case 'v':
		case 'V':
			v.mode = rest === 'v' ? 'visual' : 'visual-line';
			v.anchor = cursor;
			return done(text, cursor, { selection: sel(v, text, cursor) });
		case ':':
			v.pending = ':';
			return done(text, cursor);
	}
	return done(text, cursor); // unknown keys do nothing in normal mode
}

function sel(v: Vim, t: string, cursor: number): [number, number] {
	if (v.mode === 'visual-line') return lineRange(t, v.anchor, cursor);
	return [Math.min(v.anchor, cursor), Math.min(t.length, Math.max(v.anchor, cursor) + 1)];
}

function visualResult(v: Vim, t: string, to: number): Result {
	const c = clampNormal(t, to);
	return { text: t, cursor: c, handled: true, selection: sel(v, t, c) };
}
