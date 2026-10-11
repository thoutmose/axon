import { fr } from './i18n.fr';

export type Locale = 'en' | 'fr';

// The interface language. English strings are the keys; a missing French
// string falls back to English, so the app never shows a blank.
export const i18n = $state({ locale: 'en' as Locale });

const key = 'axon-locale';

/** Translates an English string, filling {name} placeholders. */
export function t(s: string, vars?: Record<string, string | number>): string {
	let out = i18n.locale === 'fr' ? (fr[s] ?? s) : s;
	if (vars) for (const [k, v] of Object.entries(vars)) out = out.replaceAll(`{${k}}`, String(v));
	return out;
}

export function setLocale(l: Locale) {
	i18n.locale = l;
	if (typeof document !== 'undefined') document.documentElement.lang = l;
	try {
		localStorage.setItem(key, l);
	} catch {
		// Private browsing: the choice lasts for this page only.
	}
}

export function storedLocale(): Locale {
	try {
		return localStorage.getItem(key) === 'fr' ? 'fr' : 'en';
	} catch {
		return 'en';
	}
}
