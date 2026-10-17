<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import { colors, hits, HEIGHT, shapeSVG, toSVG, WIDTH, type Point, type Shape } from './sketch';

	// Drawing in a note (EP-143): pen, marker, lines, arrows, boxes, ellipses,
	// and text on a white sheet, saved as an SVG picture the note embeds.
	let {
		open = $bindable(false),
		shapes: initial = [],
		onsave
	}: {
		open: boolean;
		shapes?: Shape[];
		onsave: (svg: string, title: string) => Promise<void> | void;
	} = $props();

	type Tool = 'pen' | 'marker' | 'line' | 'arrow' | 'rect' | 'ellipse' | 'text' | 'eraser';
	let dialog: HTMLDialogElement;
	let sheet = $state<SVGSVGElement>();
	let shapes = $state<Shape[]>([]);
	let undone = $state<Shape[][]>([]);
	let history = $state<Shape[][]>([]);
	let draft = $state<Shape | null>(null);
	let tool = $state<Tool>('pen');
	let color = $state(colors[0]);
	let width = $state(4);
	let text = $state('');
	let title = $state('');
	let saving = $state(false);

	$effect(() => {
		if (open && !dialog.open) {
			shapes = structuredClone($state.snapshot(initial)) as Shape[];
			history = [];
			undone = [];
			title = '';
			dialog.showModal();
		}
		if (!open && dialog.open) dialog.close();
	});

	const tools: { id: Tool; label: string; icon: string }[] = $derived([
		{ id: 'pen', label: t('Pen'), icon: '✎' },
		{ id: 'marker', label: t('Marker'), icon: '▬' },
		{ id: 'line', label: t('Line'), icon: '╱' },
		{ id: 'arrow', label: t('Arrow'), icon: '→' },
		{ id: 'rect', label: t('Box'), icon: '▭' },
		{ id: 'ellipse', label: t('Ellipse'), icon: '◯' },
		{ id: 'text', label: t('Text'), icon: 'T' },
		{ id: 'eraser', label: t('Eraser'), icon: '⌫' }
	]);
	const colorNames: Record<string, string> = $derived({
		'#1f2328': t('Black'),
		'#d1242f': t('Red'),
		'#0969da': t('Blue'),
		'#1a7f37': t('Green'),
		'#bc4c00': t('Orange'),
		'#8250df': t('Purple')
	});

	function commit(next: Shape[]) {
		history = [...history, $state.snapshot(shapes) as Shape[]];
		undone = [];
		shapes = next;
	}
	function undo() {
		if (!history.length) return;
		undone = [...undone, $state.snapshot(shapes) as Shape[]];
		shapes = history[history.length - 1];
		history = history.slice(0, -1);
	}
	function redo() {
		if (!undone.length) return;
		history = [...history, $state.snapshot(shapes) as Shape[]];
		shapes = undone[undone.length - 1];
		undone = undone.slice(0, -1);
	}

	function at(e: PointerEvent): Point {
		const box = sheet!.getBoundingClientRect();
		return [
			((e.clientX - box.left) / box.width) * WIDTH,
			((e.clientY - box.top) / box.height) * HEIGHT
		];
	}
	function erase(p: Point) {
		const kept = shapes.filter((s) => !hits(s, p));
		if (kept.length !== shapes.length) commit(kept);
	}

	function down(e: PointerEvent) {
		if (e.button !== 0) return;
		const p = at(e);
		if (tool === 'eraser') {
			erase(p);
		} else if (tool === 'text') {
			if (text.trim()) {
				commit([
					...shapes,
					{ kind: 'text', color, size: 12 + width * 4, x: p[0], y: p[1], text: text.trim() }
				]);
				text = '';
			}
			return;
		} else if (tool === 'pen' || tool === 'marker') {
			draft = {
				kind: 'pen',
				color,
				width: tool === 'marker' ? width * 4 : width,
				points: [p],
				...(tool === 'marker' ? { marker: true } : {})
			};
		} else {
			draft = { kind: tool, color, width, x1: p[0], y1: p[1], x2: p[0], y2: p[1] };
		}
		sheet!.setPointerCapture(e.pointerId);
	}
	function move(e: PointerEvent) {
		if (tool === 'eraser' && e.buttons === 1) return erase(at(e));
		if (!draft) return;
		const p = at(e);
		if (draft.kind === 'pen') {
			const last = draft.points[draft.points.length - 1];
			if (Math.hypot(p[0] - last[0], p[1] - last[1]) > 2) draft.points.push(p);
		} else if (draft.kind !== 'text') {
			draft.x2 = p[0];
			draft.y2 = p[1];
		}
	}
	function up() {
		if (!draft) return;
		const d = $state.snapshot(draft) as Shape;
		draft = null;
		const tiny = d.kind !== 'pen' && d.kind !== 'text' && Math.hypot(d.x2 - d.x1, d.y2 - d.y1) < 3;
		if (!tiny) commit([...shapes, d]);
	}

	function onkeydown(e: KeyboardEvent) {
		if (!(e.ctrlKey || e.metaKey) || e.key.toLowerCase() !== 'z') return;
		if ((e.target as HTMLElement).tagName === 'INPUT') return;
		e.preventDefault();
		if (e.shiftKey) redo();
		else undo();
	}

	async function save() {
		saving = true;
		try {
			const name = title.trim() || t('Drawing');
			await onsave(toSVG($state.snapshot(shapes) as Shape[], name), name);
			open = false;
		} finally {
			saving = false;
		}
	}
</script>

<dialog
	bind:this={dialog}
	class="sketch"
	aria-labelledby="sketch-title"
	onclose={() => (open = false)}
	{onkeydown}
>
	<header>
		<h2 id="sketch-title">{t('Draw')}</h2>
		<label class="name">
			<span class="visually-hidden">{t('Name of the drawing')}</span>
			<input bind:value={title} placeholder={t('Name of the drawing')} maxlength="80" />
		</label>
		<div class="actions">
			<Button variant="ghost" onclick={() => (open = false)}>{t('Cancel')}</Button>
			<Button busy={saving} disabled={!shapes.length} onclick={save}>{t('Save in the note')}</Button
			>
		</div>
	</header>

	<div class="bar" role="toolbar" aria-label={t('Drawing tools')}>
		{#each tools as x (x.id)}
			<button
				type="button"
				class="tool"
				aria-pressed={tool === x.id}
				aria-label={x.label}
				title={x.label}
				onclick={() => (tool = x.id)}>{x.icon}</button
			>
		{/each}
		<span class="sep"></span>
		{#each colors as c (c)}
			<button
				type="button"
				class="swatch"
				style:background={c}
				aria-pressed={color === c}
				aria-label={colorNames[c]}
				title={colorNames[c]}
				onclick={() => (color = c)}
			></button>
		{/each}
		<span class="sep"></span>
		<label class="width">
			{t('Thickness')}
			<select bind:value={width}>
				<option value={2}>{t('Fine')}</option>
				<option value={4}>{t('Medium')}</option>
				<option value={8}>{t('Bold')}</option>
			</select>
		</label>
		{#if tool === 'text'}
			<label class="text">
				<span class="visually-hidden">{t('Text to place')}</span>
				<input bind:value={text} placeholder={t('Type, then click the sheet')} />
			</label>
		{/if}
		<span class="sep"></span>
		<button
			type="button"
			class="tool"
			disabled={!history.length}
			onclick={undo}
			aria-label={t('Undo')}
			title={`${t('Undo')} (Ctrl+Z)`}>↶</button
		>
		<button
			type="button"
			class="tool"
			disabled={!undone.length}
			onclick={redo}
			aria-label={t('Redo')}
			title={`${t('Redo')} (Ctrl+Shift+Z)`}>↷</button
		>
		<button
			type="button"
			class="tool"
			disabled={!shapes.length}
			onclick={() => commit([])}
			aria-label={t('Clear the sheet')}
			title={t('Clear the sheet')}>✕</button
		>
	</div>

	<svg
		bind:this={sheet}
		class="sheet tool-{tool}"
		viewBox="0 0 {WIDTH} {HEIGHT}"
		role="img"
		aria-label={t('Drawing sheet: {n} shapes', { n: shapes.length })}
		onpointerdown={down}
		onpointermove={move}
		onpointerup={up}
		onpointercancel={up}
	>
		<rect width={WIDTH} height={HEIGHT} fill="#ffffff" />
		<!-- Shapes the person drew, serialised by $lib/sketch (escaped text). -->
		{@html shapes.map(shapeSVG).join('') + (draft ? shapeSVG(draft) : '')}
	</svg>
	<p class="muted small">
		{t(
			'Saved as an SVG picture in Attachments/: it shows in the note, in the vault (Obsidian), and opens here again for more drawing.'
		)}
	</p>
</dialog>

<style>
	.sketch {
		width: min(72rem, calc(100vw - 2 * var(--space-4)));
		max-width: none;
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--surface);
		color: var(--fg);
		padding: var(--space-4);
		box-shadow: var(--shadow);
	}
	.sketch::backdrop {
		background: rgb(0 0 0 / 0.45);
	}
	header {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		align-items: center;
		margin-bottom: var(--space-3);
	}
	h2 {
		margin: 0;
		font-size: var(--text-lg);
	}
	.name {
		flex: 1;
		min-width: 10rem;
	}
	.name input {
		width: 100%;
	}
	.actions {
		display: flex;
		gap: var(--space-2);
	}
	.bar {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		align-items: center;
		margin-bottom: var(--space-2);
	}
	.tool {
		min-width: 2.25rem;
		height: 2.25rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--bg);
		color: var(--fg);
		font-size: 1.05rem;
		cursor: pointer;
	}
	.tool[aria-pressed='true'] {
		background: var(--accent);
		border-color: var(--accent);
		color: #fff;
	}
	.tool:disabled {
		opacity: 0.4;
		cursor: default;
	}
	.swatch {
		width: 1.75rem;
		height: 1.75rem;
		border-radius: 50%;
		border: 2px solid var(--surface);
		outline: 1px solid var(--border);
		cursor: pointer;
	}
	.swatch[aria-pressed='true'] {
		outline: 3px solid var(--accent);
	}
	.sep {
		width: 1px;
		align-self: stretch;
		background: var(--border);
		margin: 0 var(--space-1);
	}
	.width,
	.text {
		display: flex;
		gap: var(--space-1);
		align-items: center;
		font-size: var(--text-sm);
	}
	.sheet {
		display: block;
		width: 100%;
		height: auto;
		aspect-ratio: 3 / 2;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		touch-action: none;
		cursor: crosshair;
		margin: 0 auto;
		max-width: calc(68vh * 1.5);
	}
	.sheet.tool-text {
		cursor: text;
	}
	.sheet.tool-eraser {
		cursor: cell;
	}
	.small {
		font-size: var(--text-sm);
		margin: var(--space-2) 0 0;
	}
</style>
