import { describe, expect, it } from 'vitest';
import { fromURL } from './practice';

describe('fromURL', () => {
	it('reads the platform and a title from a problem link', () => {
		expect(fromURL('https://leetcode.com/problems/two-sum/description/')).toEqual({
			platform: 'leetcode',
			title: 'Two Sum'
		});
		expect(fromURL('https://codeforces.com/problemset/problem/1352/C')).toEqual({
			platform: 'codeforces',
			title: 'Codeforces 1352C'
		});
		expect(fromURL('https://adventofcode.com/2024/day/5')).toEqual({
			platform: 'advent-of-code',
			title: 'Advent of Code 2024, day 5'
		});
		expect(fromURL('https://overthewire.org/wargames/bandit/bandit6.html')).toEqual({
			platform: 'overthewire',
			title: 'Bandit level 5 → 6'
		});
		expect(fromURL('https://www.kaggle.com/competitions/titanic')).toEqual({
			platform: 'kaggle',
			title: 'Titanic'
		});
		expect(fromURL('https://exercism.org/tracks/rust/exercises/reverse-string').platform).toBe(
			'exercism'
		);
		expect(
			fromURL('https://portswigger.net/web-security/sql-injection/lab-login-bypass').platform
		).toBe('portswigger');
	});
	it('leaves anything else alone', () => {
		expect(fromURL('https://example.org/some/puzzle')).toEqual({ platform: 'other', title: '' });
		expect(fromURL('not a link')).toEqual({ platform: '', title: '' });
	});
});
