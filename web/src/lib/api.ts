import type { components, paths } from './api-types';

/** Error carrying the API's RFC 9457 problem detail, safe to show to the user. */
export class ApiError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
	}
}

/** Calls the Axon API from the browser (same origin, session cookie included). */
export async function api<T = unknown>(
	path: string,
	init: { method?: string; body?: unknown; headers?: Record<string, string> } = {}
): Promise<T> {
	return (await apiResponse<T>(path, init)).data;
}

/** Like api(), but also returns the ETag needed for a later PATCH (If-Match). */
export async function apiResponse<T = unknown>(
	path: string,
	init: { method?: string; body?: unknown; headers?: Record<string, string> } = {}
): Promise<{ data: T; etag: string }> {
	let res: Response;
	try {
		res = await fetch(`/api${path}`, {
			method: init.method ?? 'GET',
			credentials: 'same-origin',
			headers: {
				...(init.body === undefined ? {} : { 'Content-Type': 'application/json' }),
				...init.headers
			},
			body: init.body === undefined ? undefined : JSON.stringify(init.body)
		});
	} catch {
		throw new ApiError(0, 'Axon is unreachable. Check your connection and try again.');
	}
	if (!res.ok) {
		const problem = await res.json().catch(() => null);
		throw new ApiError(res.status, problem?.detail ?? `Request failed (HTTP ${res.status}).`);
	}
	const etag = res.headers.get('ETag') ?? '';
	if (res.status === 204 || res.status === 201) {
		const text = await res.text();
		return { data: (text ? JSON.parse(text) : undefined) as T, etag };
	}
	return { data: (await res.json()) as T, etag };
}

export function message(err: unknown): string {
	if (err instanceof ApiError) return err.message;
	if (err instanceof Error && err.name === 'NotAllowedError') {
		return 'The passkey request was cancelled or timed out.';
	}
	return 'Something went wrong. Try again.';
}

/** Response shapes generated from api/openapi/openapi.yaml (npm run gen:api). */
export type Schemas = components['schemas'];

export type AuthStatus =
	paths['/v1/auth/status']['get']['responses'][200]['content']['application/json'];
