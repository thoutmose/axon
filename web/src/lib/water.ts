// Water drunk (EP-101): the day as glasses of 250 ml.
export const GLASS = 250;

/** How many glasses the day holds and how many are drunk. */
export function glasses(
	totalML: number,
	targetML: number | null
): { filled: number; total: number } {
	const drunk = Math.round(totalML / GLASS);
	const total = Math.min(
		16,
		Math.max(targetML ? Math.ceil(targetML / GLASS) : 8, targetML ? 0 : 8, drunk)
	);
	return { filled: Math.min(drunk, total), total };
}

/** Millilitres as litres, up to two decimals. */
export const litres = (ml: number, locale: string) =>
	(ml / 1000).toLocaleString(locale, { maximumFractionDigits: 2 });
