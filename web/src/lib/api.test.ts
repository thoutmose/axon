import { afterEach, describe, expect, it, vi } from 'vitest';
import { api, ApiError, message } from './api';
import { connection } from './connection.svelte';

afterEach(() => vi.unstubAllGlobals());

const respond = (status: number, body: string | null, type = 'application/json') =>
	vi.stubGlobal(
		'fetch',
		vi.fn(async () => new Response(body, { status, headers: { 'Content-Type': type } }))
	);

describe('api', () => {
	it('prefixes /api and returns JSON', async () => {
		respond(200, '{"ok":true}');
		await expect(api('/v1/me')).resolves.toEqual({ ok: true });
		expect(vi.mocked(fetch).mock.calls[0][0]).toBe('/api/v1/me');
	});

	it('surfaces the problem detail', async () => {
		respond(403, '{"detail":"Registration is closed on this Axon."}', 'application/problem+json');
		await expect(api('/v1/auth/register/begin', { method: 'POST', body: {} })).rejects.toThrow(
			'Registration is closed on this Axon.'
		);
	});

	it('handles empty 204 responses', async () => {
		respond(204, null);
		await expect(api('/v1/auth/logout', { method: 'POST' })).resolves.toBeUndefined();
	});

	it('explains network failures', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => {
				throw new TypeError('fetch failed');
			})
		);
		const err = await api('/v1/me').catch((e) => e);
		expect(err).toBeInstanceOf(ApiError);
		expect(message(err)).toMatch(/unreachable/);
	});

	it('says Axon is down only when Axon itself does not answer', async () => {
		// The API answering 502 about another server (Audiobookshelf, Anki) is up.
		respond(502, '{"detail":"Audiobookshelf did not answer."}', 'application/problem+json');
		await expect(api('/v1/audiobookshelf', { method: 'PUT', body: {} })).rejects.toThrow(
			'Audiobookshelf did not answer.'
		);
		expect(connection.apiDown).toBe(false);
		// Caddy's own 502, with the API stopped, is not.
		respond(502, '', 'text/plain');
		await expect(api('/v1/me')).rejects.toBeInstanceOf(ApiError);
		expect(connection.apiDown).toBe(true);
	});
});
