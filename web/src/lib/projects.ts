import type { Schemas } from './api';

export type Project = Schemas['Project'];
export type Card = Schemas['Card'];
export type Column = { id: Card['column']; label: string; limit: number | null; cards: Card[] };

const order: Card['column'][] = ['backlog', 'todo', 'doing', 'done'];

/** The column to the left (-1) or right (+1), or null at the edge. */
export function neighbour(col: Card['column'], step: -1 | 1): Card['column'] | null {
	const i = order.indexOf(col) + step;
	return i >= 0 && i < order.length ? order[i] : null;
}

/**
 * Where a card lands when moved up (-1) or down (+1) in its column: the id
 * of the card it goes before, undefined for the end, or null if it cannot move.
 */
export function reorder(cards: Card[], taskId: string, step: -1 | 1): string | undefined | null {
	const i = cards.findIndex((c) => c.task_id === taskId);
	if (i < 0) return null;
	if (step === -1) return i === 0 ? null : cards[i - 1].task_id;
	if (i === cards.length - 1) return null;
	return i + 2 < cards.length ? cards[i + 2].task_id : undefined;
}

/** "2 of 3" with whether the WIP limit is reached. */
export function wip(col: Column): { label: string; full: boolean } | null {
	if (col.limit == null) return null;
	const n = col.cards.length;
	return { label: `${n} of ${col.limit}`, full: n >= col.limit };
}
