// One set of line icons (EP-129), drawn on a 24 × 24 grid with a round
// stroke, after Lucide (ISC licence, https://lucide.dev). Each is the inner
// markup of an <svg>; the stroke and colour come from the Icon component.

const circle = (cx: number, cy: number, r: number) => `<circle cx="${cx}" cy="${cy}" r="${r}"/>`;
const path = (...d: string[]) => d.map((x) => `<path d="${x}"/>`).join('');

export const icons = {
	task: '<rect x="3" y="3" width="18" height="18" rx="2"/>' + path('m9 12 2 2 4-4'),
	note: path(
		'M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z',
		'M14 2v4a2 2 0 0 0 2 2h4',
		'M16 13H8',
		'M16 17H8',
		'M10 9H8'
	),
	goal: circle(12, 12, 10) + circle(12, 12, 6) + circle(12, 12, 2),
	milestone: path('M4 22V4', 'M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1'),
	project:
		path(
			'M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z'
		) + path('M8 10v4', 'M12 10v2', 'M16 10v6'),
	habit: path(
		'm17 2 4 4-4 4',
		'M3 11v-1a4 4 0 0 1 4-4h14',
		'm7 22-4-4 4-4',
		'M21 13v1a4 4 0 0 1-4 4H3'
	),
	roadmap:
		circle(12, 12, 10) +
		path('m16.24 7.76-1.8 5.41a2 2 0 0 1-1.27 1.27L7.76 16.24l1.8-5.41a2 2 0 0 1 1.27-1.27z'),
	skill: path(
		'M21.42 10.92a1 1 0 0 0 0-1.84L12.83 5.18a2 2 0 0 0-1.66 0L2.6 9.08a1 1 0 0 0 0 1.83l8.57 3.91a2 2 0 0 0 1.66 0z',
		'M22 10v6',
		'M6 12.5V16a6 3 0 0 0 12 0v-3.5'
	),
	area: path(
		'M12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83Z',
		'm2 12.65 9.17 4.16a2 2 0 0 0 1.66 0L22 12.65',
		'm2 17.65 9.17 4.16a2 2 0 0 0 1.66 0L22 17.65'
	),
	resource: path(
		'M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z',
		'M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z'
	),
	media: path(
		'M20.2 6 3 11l-.9-2.4c-.3-1.1.3-2.2 1.3-2.5l13.5-4c1.1-.3 2.2.3 2.5 1.3Z',
		'm6.2 5.3 3.1 3.9',
		'm12.4 3.4 3.1 4',
		'M3 11h18v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z'
	),
	feed_item: path(
		'M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2',
		'M18 14h-8',
		'M15 18h-5',
		'M10 6h8v4h-8z'
	),
	graph:
		circle(18, 5, 3) +
		circle(6, 12, 3) +
		circle(18, 19, 3) +
		path('m8.59 13.51 6.83 3.98', 'm15.41 6.51-6.82 3.98'),
	analysis: path('M3 3v16a2 2 0 0 0 2 2h16', 'M18 17V9', 'M13 17V5', 'M8 17v-3'),
	templates:
		'<rect x="3" y="3" width="18" height="7" rx="1"/><rect x="3" y="14" width="9" height="7" rx="1"/><rect x="16" y="14" width="5" height="7" rx="1"/>',
	sparkles: path(
		'M9.94 15.5a2 2 0 0 0-1.44-1.44l-6.13-1.58a.5.5 0 0 1 0-.96L8.5 9.94A2 2 0 0 0 9.94 8.5l1.58-6.13a.5.5 0 0 1 .96 0l1.58 6.13a2 2 0 0 0 1.44 1.44l6.13 1.58a.5.5 0 0 1 0 .96l-6.13 1.58a2 2 0 0 0-1.44 1.44l-1.58 6.13a.5.5 0 0 1-.96 0z',
		'M20 3v4',
		'M22 5h-4'
	)
} as const;

export type IconName = keyof typeof icons;

/** The icon of a kind of item (task, note, goal…), if it has one. */
export function kindIcon(type: string): IconName | null {
	return type in icons ? (type as IconName) : null;
}
