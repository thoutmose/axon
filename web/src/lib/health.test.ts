import { describe, expect, it } from 'vitest';
import { fetchApiStatus } from './health';

const respond = (status: number, body: string) => async () => new Response(body, { status });

describe('fetchApiStatus', () => {
	it('reports the version when the API is up', async () => {
		const got = await fetchApiStatus(
			respond(200, '{"status":"ok","version":"1.2.3"}'),
			'http://api'
		);
		expect(got).toEqual({ state: 'up', version: '1.2.3' });
	});

	it('calls the healthz endpoint of the given base URL', async () => {
		let called = '';
		const fetchFn = async (url: string | URL | Request) => {
			called = String(url);
			return new Response('{"status":"ok","version":"x"}');
		};
		await fetchApiStatus(fetchFn as typeof fetch, 'http://api:8080');
		expect(called).toBe('http://api:8080/healthz');
	});

	it('reports down on an HTTP error', async () => {
		const got = await fetchApiStatus(respond(503, '{}'), 'http://api');
		expect(got).toEqual({ state: 'down', reason: 'The API responded with HTTP 503.' });
	});

	it('reports down when the API is unreachable', async () => {
		const unreachable = async () => {
			throw new TypeError('fetch failed');
		};
		const got = await fetchApiStatus(unreachable, 'http://api');
		expect(got.state).toBe('down');
	});

	it('reports down on a malformed body', async () => {
		expect((await fetchApiStatus(respond(200, 'not json'), 'http://api')).state).toBe('down');
		expect((await fetchApiStatus(respond(200, '{"status":"ok"}'), 'http://api')).state).toBe(
			'down'
		);
	});
});
