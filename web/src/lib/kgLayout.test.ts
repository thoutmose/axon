import { describe, expect, it } from 'vitest';
import { kgLayout } from './kgLayout';

// A hub with many neighbours of several types: the shape that knotted the
// old layout's centre (owner request).
function star(n: number) {
	const types = ['technology', 'organization', 'database', 'standard'];
	const nodes = [{ id: 'hub', type: 'technology', rank: 1, label: 'Kubernetes' }];
	const edges = [];
	for (let i = 0; i < n; i++) {
		nodes.push({
			id: `n${i}`,
			type: types[i % types.length],
			rank: 0.1,
			label: `Entity number ${i}`
		});
		edges.push({ from: 'hub', to: `n${i}`, predicate: 'uses' });
		if (i % 3 === 0)
			edges.push({ from: `n${i}`, to: `n${(i + 1) % n}`, predicate: 'mentioned_with' });
	}
	return { nodes, edges };
}

describe('kgLayout', () => {
	it('keeps every node clear of the others and inside its space', () => {
		const { nodes, edges } = star(120);
		const { placed, width, height } = kgLayout(nodes, edges);
		expect(placed).toHaveLength(121);
		for (const p of placed) {
			expect(p.x).toBeGreaterThanOrEqual(0);
			expect(p.x).toBeLessThanOrEqual(width);
			expect(p.y).toBeGreaterThanOrEqual(0);
			expect(p.y).toBeLessThanOrEqual(height);
		}
		let close = 0;
		for (let i = 0; i < placed.length; i++)
			for (let j = i + 1; j < placed.length; j++)
				if (Math.hypot(placed[i].x - placed[j].x, placed[i].y - placed[j].y) < 2 * placed[i].r)
					close++;
		expect(close).toBe(0);
	});

	it('gives each type its own region', () => {
		const { nodes, edges } = star(80);
		const { placed } = kgLayout(nodes, edges);
		const centre = (type: string) => {
			const of = placed.filter((p) => p.type === type);
			return {
				x: of.reduce((s, p) => s + p.x, 0) / of.length,
				y: of.reduce((s, p) => s + p.y, 0) / of.length
			};
		};
		const a = centre('organization');
		const b = centre('database');
		expect(Math.hypot(a.x - b.x, a.y - b.y)).toBeGreaterThan(150);
	});
});
