import { describe, expect, it } from 'vitest';
import { fr } from './i18n.fr';
import { i18n, t } from './i18n.svelte';

describe('t', () => {
	it('returns English by default and French when chosen, with placeholders', () => {
		i18n.locale = 'en';
		expect(t('Today')).toBe('Today');
		i18n.locale = 'fr';
		expect(t('Today')).toBe('Aujourd’hui');
		expect(t('Theme: {theme}. Change theme', { theme: 'sombre' })).toBe(
			'Thème : sombre. Changer de thème'
		);
		expect(t('A string nobody translated')).toBe('A string nobody translated');
		i18n.locale = 'en';
	});

	it('shows a missing value as nothing, as markup does', () => {
		expect(t('Due {date}', { date: undefined })).toBe('Due ');
		expect(t('{n} min', { n: null })).toBe(' min');
	});

	it('keeps placeholders in every French string', () => {
		for (const [en, f] of Object.entries(fr)) {
			expect(f.trim()).not.toBe('');
			for (const ph of en.match(/\{\w+\}/g) ?? []) expect(f).toContain(ph);
		}
	});

	it('names the loss floor as the API does, in English and French (EP-191)', () => {
		const lose = 'Lose weight gently: 15 % under, never under your basal rate or 1,200 kcal';
		expect(t(lose)).toContain('never under your basal rate or 1,200 kcal');
		expect(fr[lose]).toContain('jamais sous votre métabolisme de base ni sous 1 200 kcal');
	});
});

import { kindIcon } from './icons';
import { typeNames } from './links';

describe('kindIcon', () => {
	it('draws every kind of item, and nothing for an unknown one', () => {
		for (const type of Object.keys(typeNames)) expect(kindIcon(type), type).not.toBeNull();
		expect(kindIcon('unknown')).toBeNull();
	});
});

describe('emojiIcon', () => {
	it('draws the digest’s emoji as line icons, and leaves others alone', async () => {
		const { emojiIcon } = await import('./icons');
		expect(emojiIcon('✅')).toBe('task');
		expect(emojiIcon('🗓️')).toBe('calendar');
		expect(emojiIcon('🦄')).toBeNull();
	});
});
