// The Markdown editor's text changes, as pure functions of the text and
// its selection, so they can be tested without a browser.

/** Text and its selection (start === end for a caret). */
export type Edit = { value: string; start: number; end: number };

/** Wraps the selection in before/after, or a placeholder to type over. */
export function wrap(t: Edit, before: string, after = before, placeholder = 'text'): Edit {
	const picked = t.value.slice(t.start, t.end) || placeholder;
	const start = t.start + before.length;
	return {
		value: t.value.slice(0, t.start) + before + picked + after + t.value.slice(t.end),
		start,
		end: start + picked.length
	};
}

/** Toggles a prefix on every selected line; the lines stay selected. */
export function prefixLines(t: Edit, prefix: string): Edit {
	const start = t.value.lastIndexOf('\n', t.start - 1) + 1;
	const nl = t.value.indexOf('\n', t.end);
	const end = nl === -1 ? t.value.length : nl;
	const lines = t.value.slice(start, end).split('\n');
	const all = lines.every((l) => l.startsWith(prefix));
	const next = lines.map((l) => (all ? l.slice(prefix.length) : prefix + l)).join('\n');
	return {
		value: t.value.slice(0, start) + next + t.value.slice(end),
		start,
		end: start + next.length
	};
}

/** Replaces the selection with text; the caret goes after it. */
export function insert(t: Edit, text: string): Edit {
	const caret = t.start + text.length;
	return {
		value: t.value.slice(0, t.start) + text + t.value.slice(t.end),
		start: caret,
		end: caret
	};
}

/**
 * Enter at the end of a list item starts the next one (same indentation,
 * bullet, or next number; a ticked box starts unticked); Enter on an empty
 * item ends the list. null when Enter should do what it always does.
 */
export function continueList(t: Edit): Edit | null {
	if (t.start !== t.end) return null;
	const lineStart = t.value.lastIndexOf('\n', t.start - 1) + 1;
	const m = t.value
		.slice(lineStart, t.start)
		.match(/^(\s*)(- \[[ xX]\] |[-*+] |(\d+)([.)]) )(.*)$/);
	if (!m) return null;
	const [, indent, marker, number, delimiter, rest] = m;
	if (rest.trim() === '') {
		return {
			value: t.value.slice(0, lineStart) + t.value.slice(t.start),
			start: lineStart,
			end: lineStart
		};
	}
	const next = number
		? `${Number(number) + 1}${delimiter} `
		: marker.startsWith('- [')
			? '- [ ] '
			: marker;
	return insert(t, `\n${indent}${next}`);
}
