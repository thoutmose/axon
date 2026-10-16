import { describe, expect, it } from 'vitest';
import { hostOf, pickVoices, voiceLabel } from './podcastVoices';

const voices = [
	{ id: 'af_bella', lang: 'en' },
	{ id: 'af_heart', lang: 'en' },
	{ id: 'am_michael', lang: 'en' },
	{ id: 'ff_siwis', lang: 'fr' }
];

describe('podcast voices', () => {
	it('name Kokoro voices by accent and voice', () => {
		expect(voiceLabel('af_heart')).toBe('Heart · American · woman');
		expect(voiceLabel('bm_george')).toBe('George · British · man');
		expect(voiceLabel('my-voice')).toBe('my-voice');
	});
	it('keep the chosen voices, else take the best two of the language', () => {
		expect(pickVoices('en', voices, ['', ''])).toEqual(['af_heart', 'am_michael']);
		expect(pickVoices('en', voices, ['af_bella', 'af_bella'])).toEqual(['af_bella', 'af_heart']);
		expect(pickVoices('fr', voices, ['af_heart', ''])).toEqual(['ff_siwis', 'ff_siwis']);
		expect(pickVoices('de', voices, ['', ''])).toEqual(['', '']);
	});
	it('let the first host open and close, and the two take turns between', () => {
		expect([0, 1, 2, 3, 4].map((i) => hostOf(i, 5, 2))).toEqual([0, 1, 0, 1, 0]);
		expect([0, 1, 2].map((i) => hostOf(i, 3, 1))).toEqual([0, 0, 0]);
	});
});
