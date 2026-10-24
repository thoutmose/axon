import { describe, expect, it } from 'vitest';
import { message } from './api';
import { fr } from './i18n.fr';
import { i18n } from './i18n.svelte';
import { lowOnRecoveryCodes, requirePasskeys } from './recovery.svelte';

describe('recovery codes running low', () => {
	it('warns from 3 codes down, and not before the count is known', () => {
		expect(lowOnRecoveryCodes(null)).toBe(false);
		expect(lowOnRecoveryCodes(10)).toBe(false);
		expect(lowOnRecoveryCodes(4)).toBe(false);
		expect(lowOnRecoveryCodes(3)).toBe(true);
		expect(lowOnRecoveryCodes(0)).toBe(true);
	});
});

describe('passkey ceremony errors', () => {
	const named = (name: string) => Object.assign(new Error('x'), { name });

	it('explains each failure, in English and French', () => {
		for (const name of ['NotAllowedError', 'InvalidStateError', 'NotSupportedError']) {
			i18n.locale = 'en';
			const en = message(named(name));
			expect(en, name).not.toBe('Something went wrong. Try again.');
			expect(fr[en], name).toBeTruthy();
			i18n.locale = 'fr';
			expect(message(named(name)), name).toBe(fr[en]);
		}
		i18n.locale = 'en';
	});

	it('refuses to start without WebAuthn support', () => {
		expect(() => requirePasskeys()).toThrow(/not supported/);
		try {
			requirePasskeys();
		} catch (err) {
			expect((err as Error).name).toBe('NotSupportedError');
		}
	});
});
