import { describe, expect, it } from 'vitest';
import { fr } from './i18n.fr';
import { noteType, noteTypes } from './noteTypes';
import { spaceFor, spaceInfo } from './spaces';

// Every page's picture and every note type's drawing comes with words (EP-98),
// in both languages: these strings are passed by name, so the coverage test
// cannot see them.
describe('illustrations', () => {
	it('give every Reflect page its own scenery', () => {
		expect(spaceFor('/journal')).toBe('diary');
		expect(spaceFor('/insights')).toBe('observatory');
		expect(spaceFor('/review')).toBe('lighthouse');
		expect(spaceFor('/agents/runs')).toBe('automata');
	});

	it('describe each scenery and note type, in French too', () => {
		const texts = [
			...Object.values(spaceInfo).flatMap((s) => [s.title, s.about]),
			...Object.values(noteTypes).flatMap((n) => [n.title, n.about])
		];
		const missing = texts.filter((s) => !(s in fr));
		expect(missing).toEqual([]);
	});

	it('fall back to a sheet of paper for an unknown type', () => {
		expect(noteType('zine')).toBe(noteTypes.note);
		expect(noteType('book')).toBe(noteTypes.media);
	});
});
