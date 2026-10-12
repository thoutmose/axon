import { describe, expect, it } from 'vitest';
import { handle, newVim, type Vim } from './vim';

/** Types keys into the engine; "|" in the text marks the cursor. */
function run(start: string, keys: string[], v: Vim = newVim()) {
	let cursor = start.indexOf('|');
	let text = start.replace('|', '');
	let save = false;
	for (const k of keys) {
		const r = handle(v, k, text, cursor);
		if (!r.handled) {
			// Insert mode: the textarea types the character.
			text = text.slice(0, cursor) + k + text.slice(cursor);
			cursor += k.length;
			continue;
		}
		text = r.text;
		cursor = r.cursor;
		save ||= !!r.save;
	}
	return { out: text.slice(0, cursor) + '|' + text.slice(cursor), v, save };
}
const keys = (s: string) => [...s];

describe('vim motions', () => {
	it('moves by characters, words, and lines', () => {
		expect(run('|hello world', keys('w')).out).toBe('hello |world');
		expect(run('hello |world', keys('b')).out).toBe('|hello world');
		expect(run('|hello world', keys('e')).out).toBe('hell|o world');
		expect(run('|hello world', keys('$')).out).toBe('hello worl|d');
		expect(run('hello wor|ld', keys('0')).out).toBe('|hello world');
		expect(run('|abc', keys('3l')).out).toBe('ab|c'); // never past the last character
		expect(run('a|bc\ndef\nghi', keys('j')).out).toBe('abc\nd|ef\nghi');
		expect(run('abc\ndef\ng|hi', keys('2k')).out).toBe('a|bc\ndef\nghi');
		expect(run('abc\ndef\ng|hi', keys('gg')).out).toBe('|abc\ndef\nghi');
		expect(run('|abc\ndef\nghi', keys('G')).out).toBe('abc\ndef\n|ghi');
		expect(run('  |  indented', keys('^')).out).toBe('    |indented');
	});
});

describe('vim editing', () => {
	it('deletes, changes, and yanks with motions', () => {
		expect(run('|one two three', keys('dw')).out).toBe('|two three');
		expect(run('|one two three', keys('2dw')).out).toBe('|three');
		expect(run('one |two three', keys('d$')).out).toBe('one| ');
		expect(run('|one two', [...'cw', ...'ONE', 'Escape']).out).toBe('ON|E two');
		expect(run('a\n|b\nc', keys('dd')).out).toBe('a\n|c');
		expect(run('a\nb\n|c', keys('dd')).out).toBe('a\n|b');
		expect(run('|a\nb\nc', keys('2dd')).out).toBe('|c');
		expect(run('|a\nb', keys('yyp')).out).toBe('a\n|a\nb');
		expect(run('|abc', keys('xp')).out).toBe('b|ac');
		expect(run('|one\ntwo', keys('J')).out).toBe('one| two');
		expect(run('a|bc', keys('D')).out).toBe('|a');
	});

	it('enters insert mode and returns with Escape', () => {
		expect(run('|bc', ['i', 'a', 'Escape']).out).toBe('|abc');
		expect(run('|ab', ['A', 'c', 'Escape']).out).toBe('ab|c');
		expect(run('|a\nc', ['o', 'b', 'Escape']).out).toBe('a\n|b\nc');
		expect(run('a|', ['O', 'z', 'Escape']).out).toBe('|z\na');
		expect(run('|one\ntwo', [...'cc', 'x', 'Escape']).out).toBe('|x\ntwo');
	});

	it('undoes and redoes', () => {
		const r = run('|one two', keys('dwdw'));
		expect(r.out).toBe('|');
		expect(run('', ['u'], r.v).out).toBe('|two');
		const back = run('|two', ['u'], r.v);
		expect(back.out).toBe('|one two');
		expect(run('|one two', ['C-r'], r.v).out).toBe('|two');
	});

	it('selects in visual mode', () => {
		expect(run('|hello world', [...'vee', 'd']).out).toBe('|');
		expect(run('|hello world', [...'vey', '$', 'p']).out).toBe('hello worldhell|o');
		expect(run('a\n|b\nc', [...'Vj', 'd']).out).toBe('|a');
	});

	it('saves with :w and ignores other commands', () => {
		expect(run('|x', [':', 'w', 'Enter']).save).toBe(true);
		expect(run('|x', [':', 'q', 'Enter']).save).toBe(false);
		expect(run('|x', [':', 'w', 'Escape']).save).toBe(false);
	});

	it('lets the textarea handle keys it does not know', () => {
		expect(handle(newVim(), 'ArrowDown', 'ab', 0).handled).toBe(false);
		const v = newVim();
		handle(v, 'i', 'ab', 0);
		expect(handle(v, 'q', 'ab', 0).handled).toBe(false);
	});
});
