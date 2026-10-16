import { skyState } from './skyState.svelte';
import { applyTheme, type Theme } from './theme';

export type Appearance = {
	theme: Theme;
	font: 'system' | 'serif' | 'rounded' | 'mono';
	font_size: 'small' | 'normal' | 'large';
	icons: 'emoji' | 'symbols' | 'none';
	motion: 'system' | 'reduced' | 'full';
	/** Themed scenery per area; absent means on. */
	spaces?: '' | 'on' | 'off';
	/** The landscape under the sky that follows the sun; absent means mountains. */
	scenery?: string;
};

const key = 'axon-appearance';

/** Applies appearance to the page and remembers it for the next load. */
export function applyAppearance(a: Appearance) {
	const root = document.documentElement;
	applyTheme(a.theme);
	root.dataset.font = a.font;
	root.dataset.size = a.font_size;
	root.dataset.icons = a.icons;
	root.dataset.motion = a.motion;
	root.dataset.spaces = a.spaces === 'off' ? 'off' : 'on';
	skyState.landscape = a.scenery || 'mountains';
	try {
		localStorage.setItem(key, JSON.stringify(a));
	} catch {
		// Private browsing: settings apply to this page only.
	}
}

export function storedAppearance(): Appearance | null {
	try {
		const raw = localStorage.getItem(key);
		return raw ? (JSON.parse(raw) as Appearance) : null;
	} catch {
		return null;
	}
}

/** The icon theme currently applied to the page. */
export function iconTheme(): string | undefined {
	return typeof document === 'undefined' ? undefined : document.documentElement.dataset.icons;
}
