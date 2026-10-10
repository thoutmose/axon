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
	init: { method?: string; body?: unknown } = {}
): Promise<T> {
	let res: Response;
	try {
		res = await fetch(`/api${path}`, {
			method: init.method ?? 'GET',
			credentials: 'same-origin',
			headers: init.body === undefined ? {} : { 'Content-Type': 'application/json' },
			body: init.body === undefined ? undefined : JSON.stringify(init.body)
		});
	} catch {
		throw new ApiError(0, 'Axon is unreachable. Check your connection and try again.');
	}
	if (!res.ok) {
		const problem = await res.json().catch(() => null);
		throw new ApiError(res.status, problem?.detail ?? `Request failed (HTTP ${res.status}).`);
	}
	if (res.status === 204 || res.status === 201) {
		const text = await res.text();
		return (text ? JSON.parse(text) : undefined) as T;
	}
	return (await res.json()) as T;
}

export function message(err: unknown): string {
	if (err instanceof ApiError) return err.message;
	if (err instanceof Error && err.name === 'NotAllowedError') {
		return 'The passkey request was cancelled or timed out.';
	}
	return 'Something went wrong. Try again.';
}

export type AuthStatus = {
	registration_open: boolean;
	signed_in: boolean;
	totp_available: boolean;
};
