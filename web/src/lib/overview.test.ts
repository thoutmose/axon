import { describe, expect, it } from 'vitest';
import { forceLayout, levels, trend, type NetNode } from './overview';

describe('trend', () => {
	it('compares with the previous period', () => {
		expect(trend(12, 8, 'count')).toEqual({ dir: 'up', text: '+4 vs previous period' });
		expect(trend(60, 75, 'percent')).toEqual({ dir: 'down', text: '−15 pts vs previous period' });
		expect(trend(3, 3, 'count')?.dir).toBe('flat');
		expect(trend(3, null, 'count')).toBeNull();
	});
});

describe('levels', () => {
	it('puts empty days at 0 and splits the rest by quartile', () => {
		const days = [0, 1, 2, 3, 4, 10].map((value, i) => ({ day: `2026-01-0${i + 1}`, value }));
		const l = levels(days);
		expect(l[0]).toBe(0);
		expect(l[5]).toBe(4);
		expect(l.every((x) => x >= 0 && x <= 4)).toBe(true);
	});
});

describe('forceLayout', () => {
	const nodes: NetNode[] = Array.from({ length: 300 }, (_, i) => ({
		id: `n${i}`,
		type: 'skill',
		title: `Skill ${i}`
	}));
	const edges = nodes
		.slice(1)
		.map((node, i) => ({ from: node.id, to: nodes[i].id, relation: 'requires' }));

	it('is deterministic and stays inside the frame', () => {
		const a = forceLayout(nodes.slice(0, 20), edges.slice(0, 19));
		const b = forceLayout(nodes.slice(0, 20), edges.slice(0, 19));
		expect(a).toEqual(b);
		expect(a.every((p) => p.x >= 39 && p.x <= 861 && p.y >= 39 && p.y <= 521)).toBe(true);
	});

	it('keeps linked nodes closer than unlinked ones', () => {
		const p = forceLayout(nodes.slice(0, 30), edges.slice(0, 10));
		const dist = (i: number, j: number) => Math.hypot(p[i].x - p[j].x, p[i].y - p[j].y);
		expect(dist(1, 0)).toBeLessThan(dist(29, 0));
	});

	it('does not pile unconnected nodes on the edges', () => {
		const p = forceLayout(nodes.slice(0, 40), []);
		// An even spread puts a fair share in the middle half of the frame, not a ring.
		const inner = p.filter((q) => q.x > 225 && q.x < 675 && q.y > 140 && q.y < 420);
		expect(inner.length).toBeGreaterThan(6);
	});

	it('lays out 300 nodes quickly', () => {
		const start = performance.now();
		forceLayout(nodes, edges);
		expect(performance.now() - start).toBeLessThan(2000);
	});
});
