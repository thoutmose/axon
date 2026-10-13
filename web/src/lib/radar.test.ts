import { describe, expect, it } from 'vitest';
import {
	acquired,
	anchor,
	lines,
	namesRoadmaps,
	percent,
	point,
	polygon,
	reached,
	short
} from './radar';

describe('skills radar', () => {
	it('measures an area by its skills done, then done or being learnt', () => {
		const s = { label: 'Python', roadmap: 'Python', done: 2, learning: 1, total: 4 };
		expect(acquired(s)).toBe(0.5);
		expect(reached(s)).toBe(0.75);
		expect(acquired({ ...s, done: 0, learning: 0, total: 0 })).toBe(0);
	});

	it('starts at the top and goes clockwise', () => {
		const top = point(0, 4, 1, 10, 50);
		const right = point(1, 4, 1, 10, 50);
		expect(top.x).toBeCloseTo(50);
		expect(top.y).toBeCloseTo(40);
		expect(right.x).toBeCloseTo(60);
		expect(right.y).toBeCloseTo(50);
	});

	it('keeps values on their spoke', () => {
		expect(point(0, 3, 2, 10, 0).y).toBeCloseTo(-10);
		expect(point(0, 3, -1, 10, 0).y).toBeCloseTo(0);
	});

	it('draws one point per spoke', () => {
		expect(polygon([1, 0, 0.5, 0], 10, 50)).toBe('50.0,40.0 50.0,50.0 50.0,55.0 50.0,50.0');
	});

	it('aligns labels away from the centre', () => {
		expect([0, 1, 2, 3].map((i) => anchor(i, 4))).toEqual(['middle', 'start', 'middle', 'end']);
		expect(anchor(1, 3)).toBe('start');
		expect(anchor(2, 3)).toBe('end');
	});

	it('rounds to whole percents', () => {
		expect(percent(2 / 3)).toBe('67%');
	});

	it('wraps long labels on two lines at most', () => {
		expect(lines('Python')).toEqual(['Python']);
		expect(lines('Linux and Bash scripting', 16)).toEqual(['Linux and Bash', 'scripting']);
		expect(lines('Database architecture and administration', 16)).toEqual([
			'Database',
			'architecture and…'
		]);
	});

	it('names the roadmap only when sections of several share a radar', () => {
		const spoke = (label: string, roadmap: string) => ({
			label,
			roadmap,
			done: 0,
			learning: 0,
			total: 1
		});
		expect(namesRoadmaps([spoke('Python', 'Python'), spoke('Go', 'Go')])).toBe(false);
		expect(namesRoadmaps([spoke('Learn', 'Soft'), spoke('Lead', 'Soft')])).toBe(false);
		expect(namesRoadmaps([spoke('Start', 'Python'), spoke('Go', 'Go')])).toBe(true);
		expect(short('System design and data-intensive applications')).toBe('System design and…');
		expect(short('Go')).toBe('Go');
	});
});
