// A more practical knowledge graph (EP-122): pan and zoom, fit, find a
// node, and labels that never overlap. A view maps graph coordinates to
// the screen: screen = (x + k · gx, y + k · gy).

export type View = { x: number; y: number; k: number };

export const MIN_ZOOM = 0.25;
export const MAX_ZOOM = 8;

const clamp = (k: number) => Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, k));

/** Zooms by factor about the screen point (px, py), which stays put. */
export function zoomAt(v: View, factor: number, px: number, py: number): View {
	const k = clamp(v.k * factor);
	const f = k / v.k;
	return { x: px - (px - v.x) * f, y: py - (py - v.y) * f, k };
}

/** The view that shows every point in a width × height frame, with a margin. */
export function fit(
	points: { x: number; y: number }[],
	width: number,
	height: number,
	pad = 40
): View {
	if (!points.length) return { x: 0, y: 0, k: 1 };
	const xs = points.map((p) => p.x);
	const ys = points.map((p) => p.y);
	const [x0, x1, y0, y1] = [Math.min(...xs), Math.max(...xs), Math.min(...ys), Math.max(...ys)];
	const k = clamp(
		Math.min((width - 2 * pad) / Math.max(1, x1 - x0), (height - 2 * pad) / Math.max(1, y1 - y0))
	);
	return { x: width / 2 - (k * (x0 + x1)) / 2, y: height / 2 - (k * (y0 + y1)) / 2, k };
}

/** The same zoom, moved so (gx, gy) sits in the middle of the frame. */
export function centerOn(v: View, gx: number, gy: number, width: number, height: number): View {
	return { x: width / 2 - v.k * gx, y: height / 2 - v.k * gy, k: v.k };
}

type Labelled = { id: string; label: string; x: number; y: number; r: number; rank: number };

/**
 * The nodes whose labels show: those in focus first, then by rank, each
 * only if its box (above the node, in screen space, at a 13 px font)
 * overlaps none placed before. Zooming in spreads nodes, so more fit.
 */
export function placeLabels(
	nodes: Labelled[],
	v: View,
	width: (label: string) => number,
	focus: Set<string> = new Set()
): Set<string> {
	const order = [...nodes].sort(
		(a, b) => Number(focus.has(b.id)) - Number(focus.has(a.id)) || b.rank - a.rank
	);
	const boxes: [number, number, number, number][] = [];
	const shown = new Set<string>();
	for (const n of order) {
		const w = width(n.label);
		const cx = v.x + v.k * n.x;
		const top = v.y + v.k * n.y - v.k * n.r - 6 - 14;
		const box: [number, number, number, number] = [cx - w / 2 - 2, top, cx + w / 2 + 2, top + 16];
		if (boxes.some((b) => box[0] < b[2] && box[2] > b[0] && box[1] < b[3] && box[3] > b[1]))
			continue;
		boxes.push(box);
		shown.add(n.id);
	}
	return shown;
}
