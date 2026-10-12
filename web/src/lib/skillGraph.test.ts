import { describe, expect, it } from 'vitest';
import { bySection, layout, type Skill } from './skillGraph';

const skill = (id: string, section = '', position = 0) =>
	({
		id,
		title: id,
		section,
		position,
		status: 'todo',
		description: '',
		url: null,
		updated_at: ''
	}) as Skill;

describe('layout', () => {
	it('puts a skill right of what it requires', () => {
		const { nodes } = layout(
			[skill('a'), skill('b'), skill('c')],
			[
				{ from: 'b', to: 'a' },
				{ from: 'c', to: 'b' }
			]
		);
		const at = Object.fromEntries(nodes.map((n) => [n.skill.id, n.layer]));
		expect(at).toEqual({ a: 0, b: 1, c: 2 });
	});
	it('survives cycles', () => {
		const { nodes } = layout(
			[skill('a'), skill('b')],
			[
				{ from: 'a', to: 'b' },
				{ from: 'b', to: 'a' }
			]
		);
		expect(nodes).toHaveLength(2);
	});
	it('lays out 500 skills quickly', () => {
		const skills = Array.from({ length: 500 }, (_, i) => skill(`s${i}`, `p${i % 10}`, i));
		const edges = skills.slice(1).map((s, i) => ({ from: s.id, to: skills[Math.floor(i / 2)].id }));
		const t = performance.now();
		const { nodes, width, height } = layout(skills, edges);
		expect(performance.now() - t).toBeLessThan(50);
		expect(nodes).toHaveLength(500);
		expect(width).toBeGreaterThan(0);
		expect(height).toBeGreaterThan(0);
	});
});

describe('bySection', () => {
	it('groups in order', () => {
		const groups = bySection([
			skill('b', 'Core', 2),
			skill('a', 'Intro', 1),
			skill('c', 'Core', 3)
		]);
		expect(groups.map(([s, items]) => [s, items.map((i) => i.id)])).toEqual([
			['Intro', ['a']],
			['Core', ['b', 'c']]
		]);
	});
});
