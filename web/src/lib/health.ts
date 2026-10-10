export type ApiStatus = { state: 'up'; version: string } | { state: 'down'; reason: string };

/** Asks the API whether it is alive. Never throws: failures become a 'down' status. */
export async function fetchApiStatus(
	fetchFn: typeof fetch,
	baseUrl: string,
	timeoutMs = 2000
): Promise<ApiStatus> {
	let res: Response;
	try {
		res = await fetchFn(`${baseUrl}/healthz`, { signal: AbortSignal.timeout(timeoutMs) });
	} catch {
		return { state: 'down', reason: 'The API is unreachable.' };
	}
	if (!res.ok) {
		return { state: 'down', reason: `The API responded with HTTP ${res.status}.` };
	}
	const body: unknown = await res.json().catch(() => null);
	if (
		typeof body === 'object' &&
		body !== null &&
		'status' in body &&
		body.status === 'ok' &&
		'version' in body &&
		typeof body.version === 'string'
	) {
		return { state: 'up', version: body.version };
	}
	return { state: 'down', reason: 'The API sent an unexpected response.' };
}
