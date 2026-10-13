import { skyAt } from './sky';

export type Theme = 'system' | 'light' | 'dark' | 'sun';

const key = 'axon-theme';
const zoneKey = 'axon-zone';

export function storedTheme(): Theme {
	try {
		const t = localStorage.getItem(key);
		return t === 'light' || t === 'dark' || t === 'sun' ? t : 'system';
	} catch {
		return 'system';
	}
}

function zone(): string {
	try {
		return (
			localStorage.getItem(zoneKey) || Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
		);
	} catch {
		return 'UTC';
	}
}

/** Applies a theme; "sun" is light from dawn to dusk in the person's time zone (EP-96). */
export function applyTheme(theme: Theme, now = new Date()) {
	if (theme === 'system') document.documentElement.removeAttribute('data-theme');
	else if (theme === 'sun')
		document.documentElement.dataset.theme = skyAt(now, zone()).light ? 'light' : 'dark';
	else document.documentElement.dataset.theme = theme;
	try {
		if (theme === 'system') localStorage.removeItem(key);
		else localStorage.setItem(key, theme);
	} catch {
		// Private browsing: the choice lasts for this page only.
	}
}
