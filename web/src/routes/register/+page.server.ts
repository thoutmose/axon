import { redirect } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { PageServerLoad } from './$types';

// Registration closed and no invitation: an account here is the owner's to
// give, so a visitor is sent to Axon's source, to run their own.
export const load: PageServerLoad = async ({ fetch, url }) => {
	if (url.searchParams.has('invite')) return {};
	// A slow API (a fresh install still migrating) fails open, quickly.
	const res = await fetch(`${env.API_URL ?? 'http://localhost:8080'}/v1/auth/status`, {
		signal: AbortSignal.timeout(2000)
	}).catch(() => null);
	const status: { registration_open?: boolean } | null = await res?.json().catch(() => null);
	if (status?.registration_open === false) redirect(307, 'https://github.com/thoutmose/axon');
	return {};
};
