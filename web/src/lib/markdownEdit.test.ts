import { describe, expect, it } from 'vitest';
import { continueList, insert, prefixLines, wrap, type Edit } from './markdownEdit';

/** Text with the selection marked by [ and ], or a caret by |. */
function at(marked: string): Edit {
	const caret = marked.indexOf('|');
	if (caret >= 0) return { value: marked.replace('|', ''), start: caret, end: caret };
	const start = marked.indexOf('[');
	const end = marked.indexOf(']') - 1;
	return { value: marked.replace('[', '').replace(']', ''), start, end };
}

/** The edit written back with its selection marked the same way. */
function show(e: Edit | null): string | null {
	if (!e) return null;
	const v = e.value;
	if (e.start === e.end) return v.slice(0, e.start) + '|' + v.slice(e.start);
	return v.slice(0, e.start) + '[' + v.slice(e.start, e.end) + ']' + v.slice(e.end);
}

describe('wrap', () => {
	it('wraps the selection and keeps it selected', () => {
		expect(show(wrap(at('a [word] b'), '**'))).toBe('a **[word]** b');
	});
	it('inserts a placeholder to type over when nothing is selected', () => {
		expect(show(wrap(at('see |'), '[[', ']]', 'note'))).toBe('see [[[note]]]');
	});
});

describe('prefixLines', () => {
	it('prefixes every selected line, then removes the prefix again', () => {
		const once = prefixLines(at('x\n[one\ntw]o\ny'), '- ');
		expect(show(once)).toBe('x\n[- one\n- two]\ny');
		expect(show(prefixLines(once, '- '))).toBe('x\n[one\ntwo]\ny');
	});
	it('works on the line of a caret', () => {
		expect(show(prefixLines(at('a\nti|tle'), '## '))).toBe('a\n[## title]');
	});
});

describe('insert', () => {
	it('replaces the selection and puts the caret after', () => {
		expect(show(insert(at('a [b] c'), 'XY'))).toBe('a XY| c');
	});
});

describe('continueList', () => {
	it('continues bullets, checklists (unticked), and numbers', () => {
		expect(show(continueList(at('- milk|')))).toBe('- milk\n- |');
		expect(show(continueList(at('* milk|')))).toBe('* milk\n* |');
		expect(show(continueList(at('- [x] done|')))).toBe('- [x] done\n- [ ] |');
		expect(show(continueList(at('9. nine|')))).toBe('9. nine\n10. |');
	});
	it('keeps the indentation and the number style', () => {
		expect(show(continueList(at('  - sub|')))).toBe('  - sub\n  - |');
		expect(show(continueList(at('1) first|')))).toBe('1) first\n2) |');
	});
	it('ends the list on an empty item', () => {
		expect(show(continueList(at('- a\n- |')))).toBe('- a\n|');
	});
	it('leaves other lines, and selections, to the browser', () => {
		expect(continueList(at('plain text|'))).toBeNull();
		expect(continueList(at('- [milk]'))).toBeNull();
	});
});
