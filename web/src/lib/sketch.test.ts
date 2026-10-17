import { describe, expect, it } from 'vitest';
import { fromSVG, hits, penPath, svgEmbed, toSVG, type Shape } from './sketch';

const shapes: Shape[] = [
	{
		kind: 'pen',
		color: '#1f2328',
		width: 4,
		points: [
			[10, 10],
			[20, 15],
			[30, 10]
		]
	},
	{ kind: 'arrow', color: '#d1242f', width: 2, x1: 100, y1: 100, x2: 300, y2: 100 },
	{ kind: 'rect', color: '#0969da', width: 2, x1: 400, y1: 300, x2: 200, y2: 200 },
	{ kind: 'text', color: '#1a7f37', size: 28, x: 50, y: 500, text: 'Raft <leader> & "votes"' }
];

describe('sketches', () => {
	it('are SVG pictures that Axon can open again', () => {
		const svg = toSVG(shapes, 'Consensus');
		expect(svg.startsWith('<svg xmlns="http://www.w3.org/2000/svg"')).toBe(true);
		expect(svg).toContain('<title>Consensus</title>');
		expect(svg).toContain('&lt;leader&gt; &amp; &quot;votes&quot;');
		expect(svg).not.toContain('<leader>');
		expect(svg).toContain('<rect x="200" y="200" width="200" height="100"');
		expect(fromSVG(svg)).toEqual(shapes);
	});

	it('drop what a crafted file adds to its shapes', () => {
		const bad = JSON.stringify({
			version: 1,
			shapes: [
				{
					kind: 'line',
					color: '"/><script>alert(1)</script>',
					width: 2,
					x1: 0,
					y1: 0,
					x2: 1,
					y2: 1
				},
				{
					kind: 'pen',
					color: '#000000',
					width: 2,
					points: [
						[1, 2],
						['x', 3]
					],
					onload: 'x'
				},
				{ kind: 'image', color: '#000000', width: 2 }
			]
		});
		const svg = `<svg><metadata id="axon-sketch">${bad.replace(/</g, '&lt;')}</metadata></svg>`;
		expect(fromSVG(svg)).toEqual([{ kind: 'pen', color: '#000000', width: 2, points: [[1, 2]] }]);
	});

	it('cannot open an SVG made elsewhere', () => {
		expect(fromSVG('<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0"/></svg>')).toBeNull();
		expect(fromSVG('<svg><metadata id="axon-sketch">{oops</metadata></svg>')).toBeNull();
	});

	it('smooth a pen stroke, and keep a dot visible', () => {
		expect(penPath([[1, 2]])).toBe('M1 2l0.1 0');
		expect(
			penPath([
				[0, 0],
				[10, 10],
				[20, 0]
			])
		).toBe('M0 0Q10 10 15 5L20 0');
	});

	it('erase what the eraser touches', () => {
		expect(hits(shapes[0], [20, 16])).toBe(true);
		expect(hits(shapes[0], [200, 200])).toBe(false);
		expect(hits(shapes[2], [300, 200])).toBe(true); // on the rectangle's edge
		expect(hits(shapes[2], [300, 250])).toBe(false); // inside it
		expect(hits(shapes[3], [60, 490])).toBe(true);
	});

	it('find the drawing on a line of Markdown', () => {
		expect(svgEmbed('See ![sketch](Attachments/sketch-ab12.svg) here')).toEqual({
			text: '![sketch](Attachments/sketch-ab12.svg)',
			target: 'Attachments/sketch-ab12.svg'
		});
		expect(svgEmbed('![[raft.svg|400]]')).toEqual({
			text: '![[raft.svg|400]]',
			target: 'raft.svg'
		});
		expect(svgEmbed('![photo](Attachments/a.png)')).toBeNull();
	});
});
