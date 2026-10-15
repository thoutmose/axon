// Coffee and tea (EP-117): what the day's caffeine means for tonight.

/** Drinks by kind, as the API names them (translated by name). */
export const drinkNames: Record<string, string> = {
	espresso: 'Espresso',
	coffee: 'Coffee',
	instant: 'Instant coffee',
	tea: 'Black tea',
	green_tea: 'Green tea',
	matcha: 'Matcha',
	cola: 'Cola',
	energy: 'Energy drink',
	decaf: 'Decaf',
	other: 'Other'
};

type Day = { total_mg: number; limit_mg: number; cutoff: string | null; at_bedtime_mg: number };

/**
 * What to say about the day: over the daily limit; a coffee still fits
 * before the cut-off; the cut-off has passed; or even one now would keep
 * too much in the body at bedtime.
 */
export function advice(d: Day, now: Date): { key: 'over' | 'before' | 'past' | 'none' } {
	if (d.total_mg > d.limit_mg) return { key: 'over' };
	if (!d.cutoff) return { key: 'none' };
	return { key: new Date(d.cutoff) > now ? 'before' : 'past' };
}
