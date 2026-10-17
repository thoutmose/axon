/**
 * Geometry of the skills radar (EP-82): spokes start at the top and go
 * clockwise; a value from 0 to 1 sits that far along its spoke.
 */

export type Spoke = {
	label: string;
	roadmap: string;
	done: number;
	learning: number;
	total: number;
	/** The assessed level, 0 to 5, of a roadmap plan's domain (EP-163). */
	level?: number;
};

/** The share of an area acquired. */
export const acquired = (s: Spoke) => (s.total ? s.done / s.total : 0);

/** The share acquired or being learnt. */
export const reached = (s: Spoke) => (s.total ? (s.done + s.learning) / s.total : 0);

/** The point a value from 0 to 1 reaches on spoke i of n, around (c, c). */
export function point(i: number, n: number, value: number, radius: number, c: number) {
	const angle = (2 * Math.PI * i) / n - Math.PI / 2;
	const r = Math.max(0, Math.min(1, value)) * radius;
	return { x: c + r * Math.cos(angle), y: c + r * Math.sin(angle) };
}

/** An SVG polygon's points for one value per spoke. */
export function polygon(values: number[], radius: number, c: number): string {
	return values
		.map((v, i) => point(i, values.length, v, radius, c))
		.map((p) => `${p.x.toFixed(1)},${p.y.toFixed(1)}`)
		.join(' ');
}

/** How a spoke's label is aligned, so it reads away from the chart. */
export function anchor(i: number, n: number): 'start' | 'middle' | 'end' {
	const x = Math.cos((2 * Math.PI * i) / n - Math.PI / 2);
	return Math.abs(x) < 0.2 ? 'middle' : x > 0 ? 'start' : 'end';
}

/** A domain's assessed level as a share of the top level, 5. */
export const level = (s: Spoke) => (s.level ?? 0) / 5;

/** Whole percent, as people read it. */
export const percent = (v: number) => `${Math.round(v * 100)}%`;

/** A label in at most two lines of about `width` characters, cut at a word with "…". */
export function lines(label: string, width = 16): string[] {
	const out: string[] = [];
	let line = '';
	for (const word of label.split(/\s+/).filter(Boolean)) {
		if (line && (line + ' ' + word).length > width) {
			out.push(line);
			line = word;
		} else line = line ? line + ' ' + word : word;
	}
	if (line) out.push(line);
	if (out.length <= 2) return out;
	return [out[0], out[1] + '…'];
}

/** Whether spokes need their roadmap named: sections of more than one. */
export function namesRoadmaps(spokes: Spoke[]): boolean {
	const roadmaps = new Set(spokes.map((s) => s.roadmap).filter(Boolean));
	return roadmaps.size > 1 && spokes.some((s) => s.roadmap && s.label !== s.roadmap);
}

/** The first line of a label, with "…" when there is more. */
export function short(label: string, width = 18): string {
	const [first, ...rest] = lines(label, width);
	return rest.length ? first + '…' : (first ?? '');
}
