// Layout of the news knowledge graph (owner request: the centre was too
// dense to read). Each type of node has its own pole: the largest type in
// the middle, the others on a ring around it, so organizations,
// technologies, databases, and the rest form regions instead of one knot;
// a class sits with the type of most of its members. Links pull by 1 / the
// smaller degree of their ends (as d3's force layout does), so a hub no
// longer drags its many neighbours into a ball; every pair repels; and
// collision passes at the end keep each node a label's width from the
// next. The space grows with the number of nodes: the page fits it to its
// frame with a zoom, so labels keep their size and more of them show as
// you zoom in.

export type KgIn = { id: string; type: string; rank: number; label: string };
export type KgPlaced<N> = N & { x: number; y: number; r: number };

const order = [
	'technology',
	'database',
	'organization',
	'cloud_service',
	'language',
	'standard',
	'vulnerability',
	'person',
	'product',
	'event',
	'location',
	'dataset',
	'publication',
	'concept'
];

export function radiusOf(n: { kind?: string; rank: number; count?: number }): number {
	if (n.kind === 'class') return 6;
	return 5 + Math.sqrt(n.rank) * 11;
}

export function kgLayout<N extends KgIn & { kind?: string; count?: number }>(
	nodes: N[],
	edges: { from: string; to: string; predicate: string }[],
	steps = 320
): { placed: KgPlaced<N>[]; width: number; height: number } {
	const n = nodes.length;
	const width = Math.max(1100, Math.round(Math.sqrt(n) * 150));
	const height = Math.round(width * 0.62);
	if (n === 0) return { placed: [], width, height };
	const index = new Map(nodes.map((node, i) => [node.id, i]));
	const links = edges
		.map((e) => ({ a: index.get(e.from), b: index.get(e.to), isA: e.predicate === 'is_a' }))
		.filter(
			(l): l is { a: number; b: number; isA: boolean } => l.a !== undefined && l.b !== undefined
		);
	const degree = new Array(n).fill(0);
	for (const l of links) {
		degree[l.a]++;
		degree[l.b]++;
	}

	// A class takes the type most of its members have.
	const typeOf = nodes.map((nd) => nd.type);
	nodes.forEach((nd, i) => {
		if (nd.type !== 'class') return;
		const votes = new Map<string, number>();
		for (const l of links) {
			if (l.b === i && l.isA) votes.set(nodes[l.a].type, (votes.get(nodes[l.a].type) ?? 0) + 1);
		}
		typeOf[i] = [...votes].sort((x, y) => y[1] - x[1])[0]?.[0] ?? 'concept';
	});
	const groups = [
		...order.filter((t) => typeOf.includes(t)),
		...new Set(typeOf.filter((t) => !order.includes(t)))
	];
	const size = (t: string) => typeOf.filter((x) => x === t).length;
	const centre = [...groups].sort((a, b) => size(b) - size(a))[0];
	const poles = new Map<string, { x: number; y: number }>([
		[centre, { x: width / 2, y: height / 2 }]
	]);
	groups
		.filter((t) => t !== centre)
		.forEach((t, i, ring) => {
			const a = (2 * Math.PI * i) / ring.length - Math.PI / 2;
			poles.set(t, {
				x: width / 2 + width * 0.4 * Math.cos(a),
				y: height / 2 + height * 0.4 * Math.sin(a)
			});
		});

	const r = nodes.map((nd) => radiusOf(nd));
	// Half the room a node needs: its disc or half its label, whichever is wider.
	const room = nodes.map((nd, i) => Math.max(r[i], Math.min(nd.label.length, 28) * 3.4) + 6);
	const pos = nodes.map((_, i) => {
		const p = poles.get(typeOf[i])!;
		const angle = i * 2.399963;
		const d = 14 * Math.sqrt(i + 1);
		return { x: p.x + d * Math.cos(angle), y: p.y + d * Math.sin(angle) };
	});
	const vel = nodes.map(() => ({ x: 0, y: 0 }));
	const k = Math.sqrt((width * height) / n) * 0.55;
	for (let step = 0; step < steps; step++) {
		const alpha = 1 - step / steps;
		const f = pos.map(() => ({ x: 0, y: 0 }));
		for (let i = 0; i < n; i++) {
			for (let j = i + 1; j < n; j++) {
				let dx = pos[i].x - pos[j].x;
				let dy = pos[i].y - pos[j].y;
				let d2 = dx * dx + dy * dy;
				if (d2 < 0.01) {
					dx = (i % 7) - 3 || 1;
					dy = (j % 5) - 2 || 1;
					d2 = dx * dx + dy * dy;
				}
				const d = Math.sqrt(d2);
				let push = (k * k * 0.15) / d2;
				const min = room[i] + room[j];
				if (d < min) push += ((min - d) / d) * 0.5; // collision, label included
				f[i].x += dx * push;
				f[i].y += dy * push;
				f[j].x -= dx * push;
				f[j].y -= dy * push;
			}
		}
		for (const l of links) {
			const dx = pos[l.b].x - pos[l.a].x;
			const dy = pos[l.b].y - pos[l.a].y;
			const d = Math.sqrt(dx * dx + dy * dy) || 1;
			const ideal = l.isA ? k * 0.7 : k;
			const s = ((d - ideal) / d) * (0.4 / Math.min(degree[l.a], degree[l.b]));
			const wa = degree[l.b] / (degree[l.a] + degree[l.b]); // the lighter end moves more
			f[l.a].x += dx * s * wa;
			f[l.a].y += dy * s * wa;
			f[l.b].x -= dx * s * (1 - wa);
			f[l.b].y -= dy * s * (1 - wa);
		}
		for (let i = 0; i < n; i++) {
			const p = poles.get(typeOf[i])!;
			f[i].x += (p.x - pos[i].x) * 0.08;
			f[i].y += (p.y - pos[i].y) * 0.08;
			vel[i].x = (vel[i].x + f[i].x) * 0.6;
			vel[i].y = (vel[i].y + f[i].y) * 0.6;
			const len = Math.hypot(vel[i].x, vel[i].y) || 1;
			const max = 30 * alpha + 1;
			pos[i].x += (vel[i].x / len) * Math.min(len, max);
			pos[i].y += (vel[i].y / len) * Math.min(len, max);
		}
	}
	// Collision passes: no node within a label's width of another.
	for (let pass = 0; pass < 80; pass++) {
		for (let i = 0; i < n; i++) {
			for (let j = i + 1; j < n; j++) {
				const dx = pos[i].x - pos[j].x;
				const dy = pos[i].y - pos[j].y;
				const d = Math.hypot(dx, dy) || 0.1;
				const min = room[i] + room[j];
				if (d < min) {
					const m = ((min - d) / d) * 0.5;
					pos[i].x += dx * m;
					pos[i].y += dy * m;
					pos[j].x -= dx * m;
					pos[j].y -= dy * m;
				}
			}
		}
	}
	// Fit to the space with a margin, keeping proportions, never stretched
	// (which would undo the collision passes).
	const xs = pos.map((p) => p.x);
	const ys = pos.map((p) => p.y);
	const minX = Math.min(...xs);
	const minY = Math.min(...ys);
	const spanX = Math.max(...xs) - minX || 1;
	const spanY = Math.max(...ys) - minY || 1;
	const m = 60;
	const scale = Math.min((width - 2 * m) / spanX, (height - 2 * m) / spanY, 1);
	const ox = (width - spanX * scale) / 2;
	const oy = (height - spanY * scale) / 2;
	return {
		placed: nodes.map((nd, i) => ({
			...nd,
			x: ox + (pos[i].x - minX) * scale,
			y: oy + (pos[i].y - minY) * scale,
			r: r[i]
		})),
		width,
		height
	};
}
