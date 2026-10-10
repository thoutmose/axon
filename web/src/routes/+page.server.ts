import { env } from '$env/dynamic/private';
import { fetchApiStatus } from '$lib/health';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	return { api: await fetchApiStatus(fetch, env.API_URL ?? 'http://localhost:8080') };
};
