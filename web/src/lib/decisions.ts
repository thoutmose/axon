// The decision journal (EP-114): how to read a calibration.

/** A Brier score against the 0.25 of always saying 50%. */
export function verdict(brier: number): 'sharp' | 'better' | 'coin' | 'worse' {
	if (brier < 0.1) return 'sharp';
	if (brier < 0.24) return 'better';
	if (brier <= 0.26) return 'coin';
	return 'worse';
}

/** Over- or under-confident, once ten decisions are reviewed and the gap passes ten points. */
export function confidence(c: {
	count: number;
	predicted: number;
	happened: number;
}): 'over' | 'under' | 'calibrated' | null {
	if (c.count < 10) return null;
	const gap = c.predicted - c.happened;
	if (gap > 0.1) return 'over';
	if (gap < -0.1) return 'under';
	return 'calibrated';
}

type Bucket = { from: number; count: number; predicted: number; happened: number };

/**
 * A reliability chart's dots in a square of the given size and padding: the
 * chance given across, the share that happened up (SVG y grows down, so the
 * caller flips it), each dot as large as its count.
 */
export function plot(buckets: Bucket[], size: number, pad: number) {
	const span = size - 2 * pad;
	const most = Math.max(1, ...buckets.map((b) => b.count));
	return buckets.map((b) => ({
		...b,
		x: pad + b.predicted * span,
		y: pad + b.happened * span,
		r: 5 + 7 * Math.sqrt(b.count / most)
	}));
}

/** A day some months later, kept within the month (31 January + 1 is 28 February). */
export function addMonths(day: string, months: number): string {
	const [y, m, d] = day.split('-').map(Number);
	const last = new Date(Date.UTC(y, m - 1 + months + 1, 0)).getUTCDate();
	return new Date(Date.UTC(y, m - 1 + months, Math.min(d, last))).toISOString().slice(0, 10);
}

/** What a verdict and a confidence say (translated by name). */
export const verdicts = {
	sharp: 'Sharp: your forecasts are close to what happens.',
	better: 'Better than always saying 50%.',
	coin: 'About as good as always saying 50%.',
	worse: 'Worse than always saying 50%: your chances point the wrong way.'
};
export const confidences = {
	over: 'You are overconfident: things you expect happen less often than you say.',
	under: 'You are underconfident: things you expect happen more often than you say.',
	calibrated: 'Well calibrated: things happen about as often as you say.'
};
