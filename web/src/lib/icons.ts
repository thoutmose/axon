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
	),
	play: path('M6 3l14 9-14 9z'),
	mic: path(
		'M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z',
		'M19 10v2a7 7 0 0 1-14 0v-2',
		'M12 19v3'
	),
	trophy: path(
		'M6 9H4.5a2.5 2.5 0 0 1 0-5H6',
		'M18 9h1.5a2.5 2.5 0 0 0 0-5H18',
		'M4 22h16',
		'M10 14.66V17c0 .55-.47.98-.97 1.21C7.85 18.75 7 20.24 7 22',
		'M14 14.66V17c0 .55.47.98.97 1.21C16.15 18.75 17 20.24 17 22',
		'M18 2H6v7a6 6 0 0 0 12 0V2Z'
	),
	coffee: path(
		'M10 2v2',
		'M14 2v2',
		'M6 2v2',
		'M16 8a1 1 0 0 1 1 1v8a4 4 0 0 1-4 4H7a4 4 0 0 1-4-4V9a1 1 0 0 1 1-1h14a4 4 0 1 1 0 8h-1'
	),
	sun:
		circle(12, 12, 4) +
		path(
			'M12 2v2',
			'M12 20v2',
			'm4.93 4.93 1.41 1.41',
			'm17.66 17.66 1.41 1.41',
			'M2 12h2',
			'M20 12h2',
			'm6.34 17.66-1.41 1.41',
			'm19.07 4.93-1.41 1.41'
		),
	mail:
		'<rect x="2" y="4" width="20" height="16" rx="2"/>' +
		path('m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7'),
	pie: path(
		'M21 12c.55 0 1-.45.95-1a10 10 0 0 0-8.95-8.95c-.55-.05-1 .4-1 .95v8a1 1 0 0 0 1 1z',
		'M21.21 15.89A10 10 0 1 1 8 2.83'
	),
	web: path('M12 2l8.66 5v10L12 22l-8.66-5V7z', 'M12 7l4.33 2.5v5L12 17l-4.33-2.5v-5z', 'M12 2v20')
} as const;

export type IconName = keyof typeof icons;

/** The icon of a kind of item (task, note, goal…), if it has one. */
export function kindIcon(type: string): IconName | null {
	return type in icons ? (type as IconName) : null;
}
