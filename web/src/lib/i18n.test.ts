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

	it('keeps placeholders in every French string', () => {
		for (const [en, f] of Object.entries(fr)) {
			expect(f.trim()).not.toBe('');
			for (const ph of en.match(/\{\w+\}/g) ?? []) expect(f).toContain(ph);
		}
	});
});

import { iconFor } from './preferences';

describe('iconFor', () => {
	it('follows the icon theme', () => {
		expect(iconFor('goal', 'emoji')).toBe('🎯');
		expect(iconFor('goal', 'symbols')).toBe('◎');
		expect(iconFor('goal', 'none')).toBe('');
		expect(iconFor('unknown', 'emoji')).toBe('');
	});
});
