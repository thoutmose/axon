import { t } from './i18n.svelte';

// The podcast's natural voices (EP-132), from the operator's speech server.
export type NaturalVoice = { id: string; lang: string };

const accents: Record<string, string> = {
	a: 'American',
	b: 'British',
	f: 'French',
	e: 'Spanish',
	i: 'Italian',
	p: 'Portuguese',
	j: 'Japanese',
	z: 'Chinese',
	h: 'Hindi'
};

/** "af_heart" → "Heart · American · woman"; any other name as it is. */
export function voiceLabel(id: string): string {
	const m = /^([a-z])([fm])_(.+)$/.exec(id);
	if (!m || !accents[m[1]]) return id;
	const name = m[3].charAt(0).toUpperCase() + m[3].slice(1);
	return `${name} · ${t(accents[m[1]])} · ${m[2] === 'f' ? t('woman') : t('man')}`;
}

// Kokoro's best-rated voices, then any of the language.
const preferred: Record<string, string[]> = {
	en: ['af_heart', 'am_michael', 'af_bella', 'bm_george', 'am_fenrir'],
	fr: ['ff_siwis']
};

/**
 * The two hosts' voices for a language: the chosen ones when the server has
 * them, else its best, the second one different from the first.
 */
export function pickVoices(
	lang: string,
	voices: NaturalVoice[],
	chosen: [string, string]
): [string, string] {
	const mine = voices.filter((v) => v.lang === lang).map((v) => v.id);
	const pool = [...(preferred[lang] ?? []).filter((id) => mine.includes(id)), ...mine];
	if (!pool.length) return ['', ''];
	const first = mine.includes(chosen[0]) ? chosen[0] : pool[0];
	const second =
		mine.includes(chosen[1]) && chosen[1] !== first
			? chosen[1]
			: (pool.find((id) => id !== first) ?? first);
	return [first, second];
}

/** Which host reads part i: the first opens and closes, then they take turns. */
export function hostOf(i: number, parts: number, hosts: number): 0 | 1 {
	if (hosts < 2 || i === 0 || i === parts - 1) return 0;
	return i % 2 === 1 ? 1 : 0;
}
