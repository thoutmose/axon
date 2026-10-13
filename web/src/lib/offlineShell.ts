/**
 * What the service worker keeps for offline use (EP-55): the app's own code
 * and static files, except the API reference (Swagger UI, 1.8 MB) and the
 * home page teaser (EP-84, 2 MB), which every device would otherwise
 * download again after each update.
 */
export function offlineShell(paths: string[]): string[] {
	return paths.filter((p) => !p.startsWith('/api-docs/') && !p.startsWith('/teaser/'));
}
