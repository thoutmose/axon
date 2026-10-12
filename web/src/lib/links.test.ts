import { describe, expect, it } from 'vitest';
import { href, label, options, type Relation } from './links';

const relations: Relation[] = [
	{ name: 'relates_to', sources: [], targets: [] },
	{ name: 'blocks', sources: ['task'], targets: ['task'] },
	{ name: 'references', sources: ['note'], targets: [] },
	{ name: 'supports', sources: ['task', 'note', 'habit'], targets: ['goal', 'milestone'] }
];

describe('options', () => {
	it('offers both directions where the rules allow', () => {
		expect(options(relations, 'goal', 'task').map((o) => o.label)).toEqual([
			'Supported by',
			'Related to'
		]);
		expect(options(relations, 'task', 'goal').map((o) => o.label)).toEqual([
			'Supports',
			'Related to'
		]);
		expect(options(relations, 'task', 'task').map((o) => o.label)).toEqual([
			'Blocks',
			'Blocked by',
			'Related to'
		]);
	});
	it('never offers automatic wikilink references', () => {
		expect(options(relations, 'note', 'task').some((o) => o.relation === 'references')).toBe(false);
	});
});

describe('label and href', () => {
	it('reads relations from each end', () => {
		expect(label('supports', false)).toBe('Supported by');
		expect(label('references', true)).toBe('Mentions');
		expect(label('part_of', true)).toBe('part of');
	});
	it('links to entity pages', () => {
		expect(href({ id: 'x', type: 'task' })).toBe('/tasks/x');
		expect(href({ id: 'x', type: 'milestone' })).toBeNull();
	});
});
