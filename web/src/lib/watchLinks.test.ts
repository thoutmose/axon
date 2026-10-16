import { describe, expect, it } from 'vitest';
import { fr } from './i18n.fr';
import { host, originalLabel } from './watchLinks';

// The original and its source, a click away (EP-126).
describe('watch links', () => {
	it('names a link by its site, without www', () => {
		expect(host('https://www.youtube.com/watch?v=x')).toBe('youtube.com');
		expect(host('http://192.168.1.10:13378/item/li_1')).toBe('192.168.1.10:13378');
		expect(host('not a link')).toBe('');
	});
	it('says what opening it does: watch, listen, or read', () => {
		expect(originalLabel({ kind: 'video', url: 'https://youtube.com/watch?v=x' })).toEqual({
			key: 'Watch on {site}',
			site: 'youtube.com'
		});
		expect(originalLabel({ kind: 'podcast', url: 'https://changelog.com/news/185' }).key).toBe(
			'Open the episode on {site}'
		);
		expect(originalLabel({ kind: 'article', url: 'https://go.dev/blog/pgo' })).toEqual({
			key: 'Read on {site}',
			site: 'go.dev'
		});
	});
	it('has French for each label, which is translated by key', () => {
		for (const k of ['Watch on {site}', 'Open the episode on {site}', 'Read on {site}'])
			expect(fr).toHaveProperty([k]);
	});
});
