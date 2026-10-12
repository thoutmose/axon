import type { Schemas } from './api';

export type Skill = Schemas['Skill'];
export type Edge = { from: string; to: string };
export type Placed = { skill: Skill; x: number; y: number; layer: number };

export const NODE_W = 200;
export const NODE_H = 52;
const GAP_X = 70;
const GAP_Y = 18;

/**
 * Lays skills out in columns: a skill sits one column to the right of the
 * furthest skill it requires. Cycles are broken where they are found, so a
 * bad edge can never hang the page. Linear in nodes plus edges.
 */
export function layout(skills: Skill[], edges: Edge[]) {
	const requires = new Map<string, string[]>();
	for (const e of edges) requires.set(e.from, [...(requires.get(e.from) ?? []), e.to]);
	const layer = new Map<string, number>();
	const visiting = new Set<string>();
	const depth = (id: string): number => {
		const known = layer.get(id);
		if (known !== undefined) return known;
		if (visiting.has(id)) return 0; // a cycle: treat as a root here
		visiting.add(id);
		let d = 0;
		for (const r of requires.get(id) ?? []) d = Math.max(d, depth(r) + 1);
		visiting.delete(id);
		layer.set(id, d);
		return d;
	};
	const columns: Skill[][] = [];
	for (const s of skills) {
		const d = depth(s.id);
		(columns[d] ??= []).push(s);
	}
	const placed: Placed[] = [];
	columns.forEach((col, i) => {
		col.sort((a, b) => a.position - b.position); // curriculum order
		col.forEach((s, j) =>
			placed.push({ skill: s, layer: i, x: i * (NODE_W + GAP_X), y: j * (NODE_H + GAP_Y) })
		);
	});
	const rows = Math.max(1, ...columns.map((c) => c?.length ?? 0));
	return {
		nodes: placed,
		width: Math.max(1, columns.length) * (NODE_W + GAP_X) - GAP_X,
		height: rows * (NODE_H + GAP_Y) - GAP_Y
	};
}

/** Skills grouped by section, in position order. */
export function bySection(skills: Skill[]): [string, Skill[]][] {
	const m = new Map<string, Skill[]>();
	for (const s of [...skills].sort((a, b) => a.position - b.position)) {
		const k = s.section || 'Skills';
		m.set(k, [...(m.get(k) ?? []), s]);
	}
	return [...m];
}
