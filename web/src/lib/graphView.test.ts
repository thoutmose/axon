import { describe, expect, it } from 'vitest';
import { centerOn, fit, placeLabels, zoomAt } from './graphView';

// A more practical knowledge graph (EP-122): zoom about a point, fit, find,
// and labels that never overlap.
describe('graph view', () => {
	it('zooms about a point, which stays under the pointer', () => {
		const v = zoomAt({ x: 0, y: 0, k: 1 }, 2, 100, 50);
		expect(v).toEqual({ x: -100, y: -50, k: 2 });
		// The point (100, 50) maps to itself: x + k * 100 = 100.
		expect(v.x + v.k * 100).toBe(100);
		expect(zoomAt({ x: 0, y: 0, k: 7 }, 4, 0, 0).k).toBe(8); // clamped
		expect(zoomAt({ x: 0, y: 0, k: 0.3 }, 0.1, 0, 0).k).toBe(0.25);
	});
	it('fits the nodes in the frame, and centres on one', () => {
		const v = fit(
			[
				{ x: 100, y: 100 },
				{ x: 300, y: 300 }
			],
			800,
			400,
			0
		);
		expect(v.k).toBe(2); // 200 × 200 into 800 × 400: limited by height
		expect(v.x + v.k * 200).toBe(400); // the middle in the middle
		expect(v.y + v.k * 200).toBe(200);
		expect(centerOn({ x: 0, y: 0, k: 2 }, 50, 50, 800, 400)).toEqual({ x: 300, y: 100, k: 2 });
	});
	it('places labels most central first, skipping those that would overlap', () => {
		const width = (s: string) => s.length * 8;
		const nodes = [
			{ id: 'a', label: 'consensus', x: 100, y: 100, r: 10, rank: 1 },
			{ id: 'b', label: 'replication', x: 110, y: 104, r: 6, rank: 0.5 }, // on top of a
			{ id: 'c', label: 'rust', x: 400, y: 300, r: 6, rank: 0.2 }
		];
		expect([...placeLabels(nodes, { x: 0, y: 0, k: 1 }, width)].sort()).toEqual(['a', 'c']);
		// The one in focus always shows, first.
		expect([...placeLabels(nodes, { x: 0, y: 0, k: 1 }, width, new Set(['b']))].sort()).toEqual([
			'b',
			'c'
		]);
		// Zoomed in, they spread apart and both fit.
		expect(placeLabels(nodes, { x: 0, y: 0, k: 4 }, width).size).toBe(3);
	});
});
