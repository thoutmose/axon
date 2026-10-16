// The original and its source, a click away (EP-126).

/** A link's site, without "www.": "youtube.com", or a LAN host and port. */
export function host(url: string): string {
	try {
		return new URL(url).host.replace(/^www\./, '');
	} catch {
		return '';
	}
}

/** What opening an item's original does, to say on its link (translated by key). */
export function originalLabel(it: { kind: string; url: string }): { key: string; site: string } {
	const site = host(it.url);
	if (it.kind === 'video') return { key: 'Watch on {site}', site };
	if (it.kind === 'podcast') return { key: 'Open the episode on {site}', site };
	return { key: 'Read on {site}', site };
}
