/**
 * The budget's pies (EP-138). A category keeps its colour whatever the
 * period: the seven largest of the year get the first seven hues in order,
 * and everything else is folded into Other, in grey.
 */

export type CategoryTotal = { name: string; kind: 'income' | 'expense'; cents: number };
export type Slice = { label: string; cents: number; color: string; share: number };

const SLOTS = 7;

/** The year's categories of a kind, largest first: the order that gives colours. */
export function colourOrder(year: CategoryTotal[], kind: CategoryTotal['kind']): string[] {
	return year
		.filter((c) => c.kind === kind)
		.sort((a, b) => b.cents - a.cents)
		.map((c) => c.name);
}

/** A period's categories of a kind as slices, largest first, Other last. */
export function slices(
	period: CategoryTotal[],
	kind: CategoryTotal['kind'],
	order: string[],
	other = 'Other'
): Slice[] {
	const of = period.filter((c) => c.kind === kind && c.cents > 0);
	const total = of.reduce((n, c) => n + c.cents, 0);
	const out: Slice[] = [];
	let rest = 0;
	for (const c of [...of].sort((a, b) => b.cents - a.cents)) {
		const slot = order.indexOf(c.name);
		if (slot >= 0 && slot < SLOTS)
			out.push({
				label: c.name,
				cents: c.cents,
				color: `var(--series-${slot + 1})`,
				share: c.cents / total
			});
		else rest += c.cents;
	}
	if (rest) out.push({ label: other, cents: rest, color: 'var(--muted)', share: rest / total });
	return out;
}

/** An SVG path for a donut slice from a to b (fractions of a turn, 0 at the top). */
export function arc(a: number, b: number, r: number, inner: number, c: number): string {
	const at = (f: number, radius: number) => {
		const angle = 2 * Math.PI * f - Math.PI / 2;
		return `${(c + radius * Math.cos(angle)).toFixed(2)} ${(c + radius * Math.sin(angle)).toFixed(2)}`;
	};
	if (b - a >= 0.9999) b = a + 0.9999; // a whole ring still draws
	const large = b - a > 0.5 ? 1 : 0;
	return `M${at(a, r)}A${r} ${r} 0 ${large} 1 ${at(b, r)}L${at(b, inner)}A${inner} ${inner} 0 ${large} 0 ${at(a, inner)}Z`;
}
