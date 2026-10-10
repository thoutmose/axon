import { redirect } from '@sveltejs/kit';
import { api, type AuthStatus } from '$lib/api';
import type { LayoutLoad } from './$types';

// Signed-in pages render in the browser, which holds the session cookie.
export const ssr = false;

export const load: LayoutLoad = async ({ url }) => {
	const status = await api<AuthStatus>('/v1/auth/status');
	if (!status.signed_in) {
		redirect(307, `/signin?next=${encodeURIComponent(url.pathname)}`);
	}
	return { status };
};
