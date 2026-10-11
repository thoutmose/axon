// Money is integer cents end to end; text is parsed with string arithmetic,
// never through floating point.

/** Parses "12", "12.5", "1 234,56", "1,234.56" into cents; null if not an amount. */
export function toCents(text: string): number | null {
	let s = text.trim().replace(/[\s  ']/g, '');
	if (!/^\d([\d.,]*\d)?$/.test(s)) return null;
	// The last separator followed by one or two digits is the decimal mark.
	const m = /^(.*?)[.,](\d{1,2})$/.exec(s);
	let whole = s,
		frac = '';
	if (m) {
		whole = m[1];
		frac = m[2];
	}
	whole = whole.replace(/[.,]/g, '');
	if (!/^\d+$/.test(whole) || whole.length > 10) return null;
	s = whole + frac.padEnd(2, '0');
	const cents = Number(s);
	return Number.isSafeInteger(cents) ? cents : null;
}

/** "€1,234.56" in the reader's locale. */
export function formatCents(cents: number, currency: string): string {
	const sign = cents < 0 ? '-' : '';
	const abs = Math.abs(cents);
	const units = Math.trunc(abs / 100);
	const rest = abs % 100;
	try {
		const parts = new Intl.NumberFormat(undefined, { style: 'currency', currency }).formatToParts(
			Number(`${units}.${String(rest).padStart(2, '0')}`)
		);
		return sign + parts.map((p) => p.value).join('');
	} catch {
		return `${sign}${units}.${String(rest).padStart(2, '0')} ${currency}`;
	}
}

/** Cents as an editable "1234.56". */
export function centsText(cents: number): string {
	return `${Math.trunc(cents / 100)}.${String(cents % 100).padStart(2, '0')}`;
}
