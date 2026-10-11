import type { paths } from './api-types';

export type Overview =
	paths['/v1/dashboard/overview']['get']['responses'][200]['content']['application/json'];
export type NetNode = Overview['skills']['nodes'][number];
export type NetEdge = Overview['skills']['edges'][number];
export type DayCount = { day: string; value: number };

/** The change from the previous period, as text and a direction. */
export function trend(value: number, previous: number | null | undefined, unit: string) {
	if (previous === null || previous === undefined) return null;
	const diff = value - previous;
	const dir = diff > 0 ? 'up' : diff < 0 ? 'down' : 'flat';
	const amount = unit === 'percent' ? `${Math.abs(diff)} pts` : `${Math.abs(diff)}`;
	const text =
		diff === 0 ? 'same as before' : `${diff > 0 ? '+' : '−'}${amount} vs previous period`;
	return { dir, text };
}

/** Intensity 0–4 for a heatmap cell, by quartiles of the non-zero days. */
export function levels(days: DayCount[]): number[] {
	const nonZero = days
		.map((d) => d.value)
		.filter((v) => v > 0)
		.sort((a, b) => a - b);
	if (nonZero.length === 0) return days.map(() => 0);
	const q = (p: number) => nonZero[Math.min(nonZero.length - 1, Math.floor(p * nonZero.length))];
	const [q1, q2, q3] = [q(0.25), q(0.5), q(0.75)];
	return days.map((d) =>
		d.value === 0 ? 0 : d.value <= q1 ? 1 : d.value <= q2 ? 2 : d.value <= q3 ? 3 : 4
	);
}

export type Positioned = NetNode & { x: number; y: number };

/**
 * A small force-directed layout: nodes repel, edges pull, and everything is
 * drawn to the centre. It is deterministic (the start positions come from the
 * node order), so the same data always gives the same picture, and it runs a
 * fixed number of steps, so a large network can never hang the page.
 */
export function forceLayout(
	nodes: NetNode[],
	edges: NetEdge[],
	width = 900,
	height = 560,
	steps = 220
): Positioned[] {
	const n = nodes.length;
	if (n === 0) return [];
	const index = new Map(nodes.map((node, i) => [node.id, i]));
	const pos = nodes.map((_, i) => {
		const angle = i * 2.399963; // golden angle spreads the start evenly
		const r = 10 * Math.sqrt(i + 1);
		return { x: width / 2 + r * Math.cos(angle), y: height / 2 + r * Math.sin(angle) };
	});
	const links = edges
		.map((e) => [index.get(e.from), index.get(e.to)])
		.filter((l): l is [number, number] => l[0] !== undefined && l[1] !== undefined);
	const k = Math.sqrt((width * height) / n) * 0.4;
	for (let step = 0; step < steps; step++) {
		const cool = 1 - step / steps;
		const disp = pos.map(() => ({ x: 0, y: 0 }));
		for (let i = 0; i < n; i++) {
			for (let j = i + 1; j < n; j++) {
				let dx = pos[i].x - pos[j].x;
				let dy = pos[i].y - pos[j].y;
				let d2 = dx * dx + dy * dy;
				if (d2 < 0.01) {
					dx = 0.1;
					dy = 0.1;
					d2 = 0.02;
				}
				if (d2 > 9 * k * k) continue; // only neighbours repel: an even disc, not a ring
				const f = (k * k) / d2;
				disp[i].x += dx * f;
				disp[i].y += dy * f;
				disp[j].x -= dx * f;
				disp[j].y -= dy * f;
			}
		}
		for (const [a, b] of links) {
			const dx = pos[a].x - pos[b].x;
			const dy = pos[a].y - pos[b].y;
			const d = Math.sqrt(dx * dx + dy * dy) || 0.1;
			const f = d / k;
			disp[a].x -= dx * f;
			disp[a].y -= dy * f;
			disp[b].x += dx * f;
			disp[b].y += dy * f;
		}
		for (let i = 0; i < n; i++) {
			// Gravity keeps unconnected skills from drifting to the edges.
			disp[i].x += (width / 2 - pos[i].x) * 0.15;
			disp[i].y += (height / 2 - pos[i].y) * 0.15;
			const len = Math.sqrt(disp[i].x ** 2 + disp[i].y ** 2) || 1;
			const max = 18 * cool + 1;
			pos[i].x += (disp[i].x / len) * Math.min(len, max);
			pos[i].y += (disp[i].y / len) * Math.min(len, max);
			pos[i].x = Math.max(20, Math.min(width - 20, pos[i].x));
			pos[i].y = Math.max(20, Math.min(height - 20, pos[i].y));
		}
	}
	// Fit the result to the frame, keeping its proportions.
	const xs = pos.map((p) => p.x);
	const ys = pos.map((p) => p.y);
	const [minX, maxX, minY, maxY] = [
		Math.min(...xs),
		Math.max(...xs),
		Math.min(...ys),
		Math.max(...ys)
	];
	const margin = 40;
	const scale = Math.min(
		(width - 2 * margin) / Math.max(1, maxX - minX),
		(height - 2 * margin) / Math.max(1, maxY - minY),
		3
	);
	const ox = (width - (maxX - minX) * scale) / 2;
	const oy = (height - (maxY - minY) * scale) / 2;
	return nodes.map((node, i) => ({
		...node,
		x: ox + (pos[i].x - minX) * scale,
		y: oy + (pos[i].y - minY) * scale
	}));
}

/** Where a network node opens. */
export function nodeHref(node: NetNode): string {
	switch (node.type) {
		case 'goal':
			return `/goals/${node.id}`;
		case 'project':
			return `/projects/${node.id}`;
		case 'note':
			return `/notes/${node.id}`;
		default:
			return `/roadmaps/${node.roadmap_id}?view=graph&skill=${node.id}`;
	}
}
