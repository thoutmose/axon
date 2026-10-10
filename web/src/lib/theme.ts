export type Theme = 'system' | 'light' | 'dark';

const key = 'axon-theme';

export function storedTheme(): Theme {
	try {
		const t = localStorage.getItem(key);
		return t === 'light' || t === 'dark' ? t : 'system';
	} catch {
		return 'system';
	}
}

export function applyTheme(theme: Theme) {
	if (theme === 'system') document.documentElement.removeAttribute('data-theme');
	else document.documentElement.dataset.theme = theme;
	try {
		if (theme === 'system') localStorage.removeItem(key);
		else localStorage.setItem(key, theme);
	} catch {
		// Private browsing: the choice lasts for this page only.
	}
}
