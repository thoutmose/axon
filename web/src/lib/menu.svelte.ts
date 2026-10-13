import { api } from '$lib/api';

// The pages the person left out of the menu (EP-94), remembered on the
// device so the menu does not flash every page before the settings answer.
const key = 'axon-menu-hidden';

export const menu = $state({ hidden: [] as string[] });

export function setHidden(hidden: string[]) {
	menu.hidden = hidden;
	try {
		localStorage.setItem(key, JSON.stringify(hidden));
	} catch {
		// Private browsing: the menu applies to this page only.
	}
}

export function storedHidden(): string[] {
	try {
		const raw = JSON.parse(localStorage.getItem(key) ?? '[]');
		return Array.isArray(raw) ? raw.filter((x) => typeof x === 'string') : [];
	} catch {
		return [];
	}
}

/** Reads the menu from the person's settings. */
export async function loadMenu() {
	const s = await api<{ menu?: { hidden: string[] } }>('/v1/me/settings');
	setHidden(s.menu?.hidden ?? []);
}
