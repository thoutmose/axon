import { describe, expect, it } from 'vitest';
import { neighbour, reorder, wip, type Card, type Column } from './projects';

const card = (id: string) => ({ task_id: id }) as Card;

describe('board moves', () => {
	it('steps between columns and stops at the edges', () => {
		expect(neighbour('todo', 1)).toBe('doing');
		expect(neighbour('backlog', -1)).toBeNull();
		expect(neighbour('done', 1)).toBeNull();
	});
	it('reorders by naming the card to go before', () => {
		const cards = [card('a'), card('b'), card('c')];
		expect(reorder(cards, 'b', -1)).toBe('a');
		expect(reorder(cards, 'a', -1)).toBeNull();
		expect(reorder(cards, 'a', 1)).toBe('c');
		expect(reorder(cards, 'b', 1)).toBeUndefined(); // to the end
		expect(reorder(cards, 'c', 1)).toBeNull();
	});
	it('reports WIP against the limit', () => {
		const col = { id: 'doing', label: 'Doing', limit: 2, cards: [card('a'), card('b')] } as Column;
		expect(wip(col)).toEqual({ label: '2 of 2', full: true });
		expect(wip({ ...col, limit: null })).toBeNull();
	});
});
