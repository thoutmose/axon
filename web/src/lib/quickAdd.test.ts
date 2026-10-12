import { describe, expect, it } from 'vitest';
import { action, describe as words, parse } from './quickAdd';

const today = new Date(2026, 9, 12); // Monday 12 October 2026

describe('quick add', () => {
	it('reads a task with a context and a date', () => {
		expect(parse('Call the plumber @phone tomorrow', today)).toEqual({
			kind: 'task',
			title: 'Call the plumber',
			context: '@phone',
			due: '2026-10-13',
			tags: []
		});
		expect(parse('Pay rent friday', today).due).toBe('2026-10-16');
		expect(parse('Pay rent monday', today).due).toBe('2026-10-19'); // the next one, not today
		expect(parse('Renew passport in 10 days', today).due).toBe('2026-10-22');
		expect(parse('Appeler maman demain', today).due).toBe('2026-10-13');
		expect(parse('Learn the cello someday', today)).toMatchObject({
			title: 'Learn the cello',
			status: 'someday'
		});
		expect(parse('Call Sam', today)).toEqual({ kind: 'task', title: 'Call Sam', tags: [] });
		expect(parse('Fix bug #12 today', today)).toMatchObject({
			title: 'Fix bug #12',
			due: '2026-10-12'
		});
	});

	it('reads the kind from a prefix, in English or French', () => {
		expect(parse('book: Atomic Habits').kind).toBe('book');
		expect(parse('livre : Le Petit Prince')).toMatchObject({
			kind: 'book',
			title: 'Le Petit Prince'
		});
		expect(parse('série: Dark').kind).toBe('series');
		expect(parse('habit: Read one page').title).toBe('Read one page');
		expect(parse('project: Garden shed').kind).toBe('project');
		expect(parse('Ratio: 3 to 1').kind).toBe('task'); // not a known prefix
	});

	it('reads a note with tags, a goal with a date, and a card with an answer and a deck', () => {
		expect(parse('note: idea for the garden #ideas #home')).toMatchObject({
			kind: 'note',
			title: 'idea for the garden',
			tags: ['ideas', 'home']
		});
		expect(parse('goal: Run a 10 km race by 2027-05-01', today)).toMatchObject({
			kind: 'goal',
			title: 'Run a 10 km race',
			due: '2027-05-01'
		});
		expect(parse('goal: Stand by me', today)).toMatchObject({ title: 'Stand by me' });
		expect(parse('card: What is spacing? :: Reviewing just before forgetting #Learning')).toEqual({
			kind: 'card',
			title: 'What is spacing?',
			back: 'Reviewing just before forgetting',
			tags: ['Learning']
		});
	});

	it('says what it will do', () => {
		expect(words(parse('Call the plumber @phone tomorrow', today))).toEqual([
			'Task',
			'@phone',
			'due 2026-10-13'
		]);
		expect(words(parse('card: Why space?'))).toEqual(['Flashcard', 'add “ :: answer”']);
		expect(action('book')).toBe('Add book');
	});
});
