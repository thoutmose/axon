import { describe, expect, it } from 'vitest';
import { offlineShell } from './offlineShell';

describe('offlineShell', () => {
	it('keeps the app, its icons, and the offline page', () => {
		const kept = [
			'/_app/immutable/entry/app.js',
			'/icons/icon-192.png',
			'/manifest.webmanifest',
			'/offline'
		];
		expect(offlineShell(kept)).toEqual(kept);
	});
	it('leaves out the API reference: 1.8 MB no one needs offline', () => {
		expect(
			offlineShell([
				'/api-docs/vendor/swagger-ui-bundle.js',
				'/api-docs/index.html',
				'/favicon.svg'
			])
		).toEqual(['/favicon.svg']);
	});
});
