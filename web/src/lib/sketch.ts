// Drawings in notes (EP-143). A sketch is a plain SVG picture, so it shows
// in Axon, in Obsidian, and in any browser, and travels with the vault like
// any other attachment. Its shapes are also kept as JSON in the SVG's
// <metadata>, so Axon can open it again and go on drawing.

export const WIDTH = 1200;
export const HEIGHT = 800;

export type Point = [number, number];
export type Shape =
	| { kind: 'pen'; color: string; width: number; points: Point[]; marker?: boolean }
	| {
			kind: 'line' | 'arrow' | 'rect' | 'ellipse';
			color: string;
			width: number;
			x1: number;
			y1: number;
			x2: number;
			y2: number;
	  }
	| { kind: 'text'; color: string; size: number; x: number; y: number; text: string };

export const colors = ['#1f2328', '#d1242f', '#0969da', '#1a7f37', '#bc4c00', '#8250df'];

const r = (n: number) => Math.round(n * 10) / 10;
const esc = (s: string) =>
	s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
const unesc = (s: string) =>
	s
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
		.replace(/&quot;/g, '"')
		.replace(/&amp;/g, '&');

/** A smooth path through the points: quadratic curves between midpoints. */
export function penPath(points: Point[]): string {
	if (points.length === 0) return '';
	const [x0, y0] = points[0];
	if (points.length === 1) return `M${r(x0)} ${r(y0)}l0.1 0`;
	let d = `M${r(x0)} ${r(y0)}`;
	for (let i = 1; i < points.length - 1; i++) {
		const [x, y] = points[i];
		const [nx, ny] = points[i + 1];
		d += `Q${r(x)} ${r(y)} ${r((x + nx) / 2)} ${r((y + ny) / 2)}`;
	}
	const [lx, ly] = points[points.length - 1];
	return d + `L${r(lx)} ${r(ly)}`;
}

/** The SVG markup of one shape. */
export function shapeSVG(s: Shape): string {
	if (s.kind === 'text') {
		return `<text x="${r(s.x)}" y="${r(s.y)}" fill="${s.color}" font-size="${s.size}" font-family="system-ui, sans-serif">${esc(s.text)}</text>`;
	}
	const stroke = `stroke="${s.color}" stroke-width="${s.width}" stroke-linecap="round" stroke-linejoin="round" fill="none"`;
	switch (s.kind) {
		case 'pen':
			return `<path d="${penPath(s.points)}" ${stroke}${s.marker ? ' stroke-opacity="0.35"' : ''}/>`;
		case 'line':
			return `<line x1="${r(s.x1)}" y1="${r(s.y1)}" x2="${r(s.x2)}" y2="${r(s.y2)}" ${stroke}/>`;
		case 'arrow': {
			const a = Math.atan2(s.y2 - s.y1, s.x2 - s.x1);
			const head = 10 + s.width * 3;
			const side = (t: number) =>
				`${r(s.x2 - head * Math.cos(a + t))} ${r(s.y2 - head * Math.sin(a + t))}`;
			return `<path d="M${r(s.x1)} ${r(s.y1)}L${r(s.x2)} ${r(s.y2)}M${side(0.45)}L${r(s.x2)} ${r(s.y2)}L${side(-0.45)}" ${stroke}/>`;
		}
		case 'rect':
			return `<rect x="${r(Math.min(s.x1, s.x2))}" y="${r(Math.min(s.y1, s.y2))}" width="${r(Math.abs(s.x2 - s.x1))}" height="${r(Math.abs(s.y2 - s.y1))}" rx="6" ${stroke}/>`;
		case 'ellipse':
			return `<ellipse cx="${r((s.x1 + s.x2) / 2)}" cy="${r((s.y1 + s.y2) / 2)}" rx="${r(Math.abs(s.x2 - s.x1) / 2)}" ry="${r(Math.abs(s.y2 - s.y1) / 2)}" ${stroke}/>`;
	}
}

/** The whole drawing as an SVG file, on a white sheet so it reads in dark themes too. */
export function toSVG(shapes: Shape[], title = 'Drawing'): string {
	return (
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${WIDTH} ${HEIGHT}" width="${WIDTH}" height="${HEIGHT}">` +
		`<title>${esc(title)}</title>` +
		`<metadata id="axon-sketch">${esc(JSON.stringify({ version: 1, shapes }))}</metadata>` +
		`<rect width="${WIDTH}" height="${HEIGHT}" fill="#ffffff"/>` +
		shapes.map(shapeSVG).join('') +
		`</svg>\n`
	);
}

const num = (v: unknown) => typeof v === 'number' && Number.isFinite(v) && Math.abs(v) < 1e5;
const hex = (v: unknown) => typeof v === 'string' && /^#[0-9a-f]{6}$/i.test(v);

/**
 * A shape read from a file, rebuilt field by field: the file may come from
 * anywhere (the vault, an import), and shapes are drawn as markup.
 */
function clean(v: unknown): Shape | null {
	if (typeof v !== 'object' || v === null) return null;
	const s = v as Record<string, unknown>;
	if (!hex(s.color)) return null;
	const color = s.color as string;
	if (s.kind === 'text') {
		if (!num(s.size) || !num(s.x) || !num(s.y) || typeof s.text !== 'string') return null;
		return {
			kind: 'text',
			color,
			size: s.size as number,
			x: s.x as number,
			y: s.y as number,
			text: s.text.slice(0, 500)
		};
	}
	if (!num(s.width) || (s.width as number) > 100) return null;
	const width = s.width as number;
	if (s.kind === 'pen') {
		if (!Array.isArray(s.points) || s.points.length > 20000) return null;
		const points = s.points.filter(
			(p): p is Point => Array.isArray(p) && p.length === 2 && num(p[0]) && num(p[1])
		);
		return {
			kind: 'pen',
			color,
			width,
			points: points.map(([x, y]) => [x, y]),
			...(s.marker === true ? { marker: true } : {})
		};
	}
	if (s.kind === 'line' || s.kind === 'arrow' || s.kind === 'rect' || s.kind === 'ellipse') {
		if (![s.x1, s.y1, s.x2, s.y2].every(num)) return null;
		return {
			kind: s.kind,
			color,
			width,
			x1: s.x1 as number,
			y1: s.y1 as number,
			x2: s.x2 as number,
			y2: s.y2 as number
		};
	}
	return null;
}

/** The shapes of a drawing made in Axon, or null for any other SVG. */
export function fromSVG(svg: string): Shape[] | null {
	const m = svg.match(/<metadata id="axon-sketch">([\s\S]*?)<\/metadata>/);
	if (!m) return null;
	try {
		const data = JSON.parse(unesc(m[1]));
		if (!Array.isArray(data?.shapes)) return null;
		return data.shapes.map(clean).filter((x: Shape | null): x is Shape => x !== null);
	} catch {
		return null;
	}
}

/** Whether a point is within reach of a shape, for the eraser. */
export function hits(s: Shape, [x, y]: Point, reach = 12): boolean {
	const near = (ax: number, ay: number, bx: number, by: number) => {
		const dx = bx - ax;
		const dy = by - ay;
		const len = dx * dx + dy * dy;
		const t = len ? Math.max(0, Math.min(1, ((x - ax) * dx + (y - ay) * dy) / len)) : 0;
		return Math.hypot(x - (ax + t * dx), y - (ay + t * dy)) <= reach + ('width' in s ? s.width : 0);
	};
	switch (s.kind) {
		case 'pen':
			return s.points.some((p, i) => near(...(s.points[i - 1] ?? p), ...p));
		case 'line':
		case 'arrow':
			return near(s.x1, s.y1, s.x2, s.y2);
		case 'rect': {
			const [a, b, c, d] = [s.x1, s.y1, s.x2, s.y2];
			return near(a, b, c, b) || near(c, b, c, d) || near(c, d, a, d) || near(a, d, a, b);
		}
		case 'ellipse': {
			const rx = Math.abs(s.x2 - s.x1) / 2 || 1;
			const ry = Math.abs(s.y2 - s.y1) / 2 || 1;
			const nx = (x - (s.x1 + s.x2) / 2) / rx;
			const ny = (y - (s.y1 + s.y2) / 2) / ry;
			return Math.abs(Math.hypot(nx, ny) - 1) * Math.min(rx, ry) <= reach + s.width;
		}
		case 'text':
			return (
				x >= s.x - reach &&
				x <= s.x + s.text.length * s.size * 0.6 + reach &&
				y >= s.y - s.size - reach &&
				y <= s.y + reach
			);
	}
}

/** The embed of an SVG picture on a line of Markdown, if there is one. */
export function svgEmbed(line: string): { text: string; target: string } | null {
	const md = line.match(/!\[[^\]]*\]\(([^)\s]+\.svg)\)/i);
	if (md) return { text: md[0], target: md[1] };
	const wiki = line.match(/!\[\[([^\]|]+\.svg)(\|[^\]]*)?\]\]/i);
	return wiki ? { text: wiki[0], target: wiki[1] } : null;
}
