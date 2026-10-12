/**
 * Whether the person's recall matches the retention they chose (EP-77): more
 * than 5 points below means reviews come too late, more than 5 above means
 * they could come later. Under 20 reviews, it is too early to say.
 */
export function verdict(c: {
	reviews: number;
	remembered: number;
	target: number;
}): 'few' | 'below' | 'on' | 'above' {
	if (c.reviews < 20) return 'few';
	if (c.remembered < c.target - 0.05) return 'below';
	if (c.remembered > c.target + 0.05) return 'above';
	return 'on';
}
