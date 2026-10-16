/** The hue (0–359) of a generated cover: the same title always gets the same colour. */
export function coverHue(title: string): number {
	let h = 0;
	for (const ch of title) h = (h * 31 + ch.codePointAt(0)!) % 360;
	return h;
}

/** A rating out of 10 as five stars, halves included: 7 is "★★★½☆". */
export function stars(rating: number): string {
	const full = Math.floor(rating / 2);
	const half = rating % 2;
	return '★'.repeat(full) + (half ? '½' : '') + '☆'.repeat(5 - full - half);
}

/** What is left to read, watch, or see, or null when the total is unknown. */
export function left(it: { progress: number; total: number | null }): number | null {
	return it.total == null ? null : Math.max(0, it.total - it.progress);
}
