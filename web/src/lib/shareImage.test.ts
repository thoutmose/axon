import { describe, expect, it } from 'vitest';
import { socialLinks, wrap } from './shareImage';

// Sharing an element (EP-121): the image's text fits its width, and the
// social links carry the words, never anything else.
describe('share image', () => {
	const width = (s: string) => s.length * 10; // 10 px a character
	it('wraps words into lines no wider than the width, at most n lines', () => {
		expect(wrap(width, 'Run a 10 km race in under an hour', 130, 3)).toEqual([
			'Run a 10 km',
			'race in under',
			'an hour'
		]);
		expect(wrap(width, 'one two three four five six seven', 100, 2)).toEqual([
			'one two',
			'three four…'
		]);
		expect(wrap(width, 'Supercalifragilistic', 100, 2)).toEqual(['Supercalifragilistic']);
	});
	it('builds the posting links of X, LinkedIn, and Bluesky from the text', () => {
		const links = socialLinks('Level 7 on Axon #axon');
		expect(links.map((l) => l.name)).toEqual(['X', 'LinkedIn', 'Bluesky']);
		expect(links[0].url).toBe('https://x.com/intent/post?text=Level%207%20on%20Axon%20%23axon');
		expect(links[2].url).toBe(
			'https://bsky.app/intent/compose?text=Level%207%20on%20Axon%20%23axon'
		);
		for (const l of links) expect(l.url).toMatch(/^https:\/\//);
	});
});
