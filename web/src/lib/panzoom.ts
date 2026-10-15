import { zoomAt, type View } from './graphView';

// Pan and zoom an SVG graph (EP-122): the wheel zooms about the pointer, a
// drag pans (after a few pixels, so a click on a node stays a click), two
// fingers pinch. The view lives in the page, which draws with it.
type Params = { view: () => View; set: (v: View) => void; width: number; height: number };

export function panzoom(svg: SVGSVGElement, params: Params) {
	let p = params;
	const pointers = new Map<number, { x: number; y: number }>();
	let start: { x: number; y: number; view: View } | null = null;
	let moved = false;
	let pinch: { d: number; view: View } | null = null;

	// From the page to the graph's frame (its viewBox).
	const local = (e: { clientX: number; clientY: number }) => {
		const r = svg.getBoundingClientRect();
		return {
			x: ((e.clientX - r.left) * p.width) / r.width,
			y: ((e.clientY - r.top) * p.height) / r.height
		};
	};
	const distance = () => {
		const [a, b] = [...pointers.values()];
		return Math.hypot(a.x - b.x, a.y - b.y);
	};

	function wheel(e: WheelEvent) {
		e.preventDefault();
		const at = local(e);
		p.set(zoomAt(p.view(), Math.exp(-e.deltaY * 0.0015), at.x, at.y));
	}
	function down(e: PointerEvent) {
		pointers.set(e.pointerId, local(e));
		if (pointers.size === 1) {
			start = { ...local(e), view: p.view() };
			moved = false;
		} else if (pointers.size === 2) {
			pinch = { d: distance(), view: p.view() };
			start = null;
		}
	}
	function move(e: PointerEvent) {
		if (!pointers.has(e.pointerId)) return;
		pointers.set(e.pointerId, local(e));
		if (pinch && pointers.size === 2) {
			const [a, b] = [...pointers.values()];
			const v = zoomAt(pinch.view, distance() / pinch.d, (a.x + b.x) / 2, (a.y + b.y) / 2);
			p.set(v);
			return;
		}
		if (!start) return;
		const at = local(e);
		const dx = at.x - start.x;
		const dy = at.y - start.y;
		if (!moved && Math.hypot(dx, dy) < 4) return;
		if (!moved) svg.setPointerCapture(e.pointerId);
		moved = true;
		p.set({ ...start.view, x: start.view.x + dx, y: start.view.y + dy });
	}
	function up(e: PointerEvent) {
		pointers.delete(e.pointerId);
		if (pointers.size < 2) pinch = null;
		if (pointers.size === 0) start = null;
	}
	// A drag is not a click on what it started on.
	function click(e: MouseEvent) {
		if (moved) {
			e.stopPropagation();
			e.preventDefault();
			moved = false;
		}
	}
	svg.addEventListener('wheel', wheel, { passive: false });
	svg.addEventListener('pointerdown', down);
	svg.addEventListener('pointermove', move);
	svg.addEventListener('pointerup', up);
	svg.addEventListener('pointercancel', up);
	svg.addEventListener('click', click, true);
	return {
		update(next: Params) {
			p = next;
		},
		destroy() {
			svg.removeEventListener('wheel', wheel);
			svg.removeEventListener('pointerdown', down);
			svg.removeEventListener('pointermove', move);
			svg.removeEventListener('pointerup', up);
			svg.removeEventListener('pointercancel', up);
			svg.removeEventListener('click', click, true);
		}
	};
}
