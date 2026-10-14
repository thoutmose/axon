// The practice log (EP-103): reading a pasted problem link into its
// platform and a title, so logging a problem is a paste and a click.

const words = (slug: string) =>
	slug
		.split('-')
		.filter(Boolean)
		.map((w) => (/^\d/.test(w) ? w : w[0].toUpperCase() + w.slice(1)))
		.join(' ');

const hosts: [RegExp, string][] = [
	[/(^|\.)leetcode\.(com|cn)$/, 'leetcode'],
	[/(^|\.)codeforces\.com$/, 'codeforces'],
	[/(^|\.)hackerrank\.com$/, 'hackerrank'],
	[/(^|\.)adventofcode\.com$/, 'advent-of-code'],
	[/(^|\.)exercism\.org$/, 'exercism'],
	[/(^|\.)projecteuler\.net$/, 'project-euler'],
	[/(^|\.)overthewire\.org$/, 'overthewire'],
	[/(^|\.)portswigger\.net$/, 'portswigger'],
	[/(^|\.)hackthebox\.com$/, 'hackthebox'],
	[/(^|\.)tryhackme\.com$/, 'tryhackme'],
	[/(^|\.)kaggle\.com$/, 'kaggle']
];

/** The platform and a title for a problem link; nothing for text that is not a link. */
export function fromURL(raw: string): { platform: string; title: string } {
	let u: URL;
	try {
		u = new URL(raw.trim());
	} catch {
		return { platform: '', title: '' };
	}
	const platform = hosts.find(([re]) => re.test(u.hostname))?.[1] ?? 'other';
	const path = u.pathname.split('/').filter(Boolean);
	let m: RegExpMatchArray | null;
	let title = '';
	switch (platform) {
		case 'leetcode':
			if (path[0] === 'problems' && path[1]) title = words(path[1]);
			break;
		case 'codeforces':
			if ((m = u.pathname.match(/problem\/(\d+)\/([A-Z]\d?)/))) title = `Codeforces ${m[1]}${m[2]}`;
			break;
		case 'advent-of-code':
			if ((m = u.pathname.match(/^\/(\d{4})\/day\/(\d+)/)))
				title = `Advent of Code ${m[1]}, day ${m[2]}`;
			break;
		case 'overthewire':
			if ((m = u.pathname.match(/\/(\w+?)(\d+)\.html$/))) {
				const level = Number(m[2]);
				title = `${m[1][0].toUpperCase()}${m[1].slice(1)} level ${level - 1} → ${level}`;
			}
			break;
		case 'project-euler':
			if (u.searchParams.get('id')) title = `Project Euler ${u.searchParams.get('id')}`;
			break;
		case 'other':
			break;
		default:
			if (path.length) title = words(path[path.length - 1].replace(/^lab-/, ''));
	}
	return { platform, title };
}
