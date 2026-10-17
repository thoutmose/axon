<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { tick } from 'svelte';
	import * as edit from './markdownEdit';
	import type { Edit } from './markdownEdit';
	import { canUpload, noteFiles, refreshFiles, resolveFile, upload } from './noteFiles.svelte';
	import MathHelper from './MathHelper.svelte';
	import SketchPad from './SketchPad.svelte';
	import { fromSVG, svgEmbed, type Shape } from './sketch';
	import { noteIndex } from './noteIndex.svelte';
	import { stem } from './notes';
	import { handle, newVim, type Mode } from './vim';

	let {
		value = $bindable(''),
		id,
		label = t('Markdown'),
		rows = 20,
		vim = false,
		pictures = false,
		onsave
	}: {
		value: string;
		id: string;
		label?: string;
		rows?: number;
		vim?: boolean;
		/** Paste, drop, or pick pictures and PDFs; they are uploaded and embedded. */
		pictures?: boolean;
		onsave?: () => void;
	} = $props();

	let picker = $state<HTMLInputElement>();
	let uploading = $state(0);
	let uploadError = $state('');

	/** Uploads files and embeds each at the caret, one per line. */
	async function addFiles(files: File[]) {
		const usable = files.filter(canUpload);
		if (!usable.length) {
			if (files.length)
				uploadError = t('Only PNG, JPEG, GIF, WebP, SVG pictures and PDFs can go in a note.');
			return;
		}
		uploadError = '';
		uploading += usable.length;
		for (const f of usable) {
			try {
				const md = await upload(f);
				const s = area.selectionStart;
				const before = value.slice(0, s);
				insert((before && !before.endsWith('\n') ? '\n' : '') + md + '\n');
			} catch (err) {
				uploadError = `${f.name}: ${err instanceof Error ? err.message : err}`;
			} finally {
				uploading--;
			}
		}
	}

	// Drawing (EP-143): a new sketch at the caret, or the one embedded on
	// the caret's line when Axon drew it, saved as a new SVG that replaces it.
	let drawing = $state(false);
	let writingMaths = $state(false);
	let sketch = $state<Shape[]>([]);
	let replacing = $state<string | null>(null);
	async function draw() {
		uploadError = '';
		sketch = [];
		replacing = null;
		const s = area.selectionStart;
		const line = value.slice(value.lastIndexOf('\n', s - 1) + 1).split('\n')[0];
		const embed = svgEmbed(line);
		if (embed) {
			try {
				if (!noteFiles.loaded) await refreshFiles();
				const f = resolveFile(embed.target);
				const res = f && (await fetch(`/api/v1/notes/file?path=${encodeURIComponent(f.path)}`));
				const shapes = res && res.ok ? fromSVG(await res.text()) : null;
				if (shapes) {
					sketch = shapes;
					replacing = embed.text;
				}
			} catch {
				// a new drawing instead
			}
		}
		drawing = true;
	}
	async function saveDrawing(svg: string, title: string) {
		const md = await upload(new File([svg], `${title}.svg`, { type: 'image/svg+xml' }));
		if (replacing && value.includes(replacing)) {
			value = value.replace(replacing, md);
		} else {
			const s = area.selectionStart;
			const before = value.slice(0, s);
			insert((before && !before.endsWith('\n') ? '\n' : '') + md + '\n');
		}
	}

	function onpaste(e: ClipboardEvent) {
		if (!pictures) return;
		const files = [...(e.clipboardData?.files ?? [])];
		if (files.length) {
			e.preventDefault();
			addFiles(files);
		}
	}

	function ondrop(e: DragEvent) {
		if (!pictures) return;
		const files = [...(e.dataTransfer?.files ?? [])];
		if (files.length) {
			e.preventDefault();
			area.focus();
			addFiles(files);
		}
	}

	const vimState = newVim();
	let mode = $state<Mode>('normal');
	let command = $state('');

	/** Vim keybindings: the engine decides, the textarea shows the result. */
	function vimKey(e: KeyboardEvent): boolean {
		if (e.altKey || ((e.ctrlKey || e.metaKey) && !(e.ctrlKey && e.key === 'r'))) return false;
		const key = e.ctrlKey && e.key === 'r' ? 'C-r' : e.key;
		const r = handle(vimState, key, value, area.selectionStart);
		mode = vimState.mode;
		command = vimState.pending.startsWith(':') ? vimState.pending : '';
		if (!r.handled) return false;
		e.preventDefault();
		value = r.text;
		const [s, end] = r.selection ?? [r.cursor, mode === 'insert' ? r.cursor : r.cursor + 1];
		tick().then(() => area.setSelectionRange(s, end));
		if (r.save) onsave?.();
		return true;
	}

	let area: HTMLTextAreaElement;
	let query = $state<string | null>(null); // text typed after "[["
	let active = $state(0);

	const suggestions = $derived.by(() => {
		if (query === null) return [];
		const q = query.toLowerCase();
		return noteIndex.notes
			.filter((n) => n.title.toLowerCase().includes(q) || stem(n.path).toLowerCase().includes(q))
			.slice(0, 8);
	});

	async function setSelection(start: number, end = start) {
		await tick();
		area.focus();
		area.setSelectionRange(start, end);
	}

	/** Applies a text change from markdownEdit and puts its selection back. */
	function apply(change: (t: Edit) => Edit) {
		const next = change({ value, start: area.selectionStart, end: area.selectionEnd });
		value = next.value;
		setSelection(next.start, next.end);
	}
	const wrap = (before: string, after = before, placeholder = 'text') =>
		apply((t) => edit.wrap(t, before, after, placeholder));
	const prefixLines = (prefix: string) => apply((t) => edit.prefixLines(t, prefix));
	const insert = (text: string) => apply((t) => edit.insert(t, text));

	const tools: { label: string; keys?: string; icon: string; run: () => void }[] = $derived([
		{ label: t('Bold'), keys: 'Ctrl+B', icon: 'B', run: () => wrap('**') },
		{ label: t('Italic'), keys: 'Ctrl+I', icon: 'I', run: () => wrap('*') },
		{ label: t('Strikethrough'), keys: 'Ctrl+Shift+X', icon: 'S', run: () => wrap('~~') },
		{ label: t('Highlight'), keys: 'Ctrl+Shift+H', icon: 'H̲', run: () => wrap('==') },
		{ label: t('Heading'), icon: '#', run: () => prefixLines('## ') },
		{ label: t('Bulleted list'), icon: '•', run: () => prefixLines('- ') },
		{ label: t('Checklist'), icon: '☐', run: () => prefixLines('- [ ] ') },
		{ label: t('Quote'), icon: '❝', run: () => prefixLines('> ') },
		{ label: t('Code'), keys: 'Ctrl+E', icon: '</>', run: () => wrap('`') },
		{
			label: t('Link to a note'),
			keys: 'Ctrl+L',
			icon: '[[ ]]',
			run: () => wrap('[[', ']]', 'note')
		},
		{
			label: t('Table'),
			icon: '▦',
			run: () => insert('\n| Column | Column |\n| --- | --- |\n| | |\n')
		},
		{ label: t('Callout'), icon: 'ⓘ', run: () => insert('\n> [!NOTE] Title\n> Text\n') },
		{ label: t('Write maths'), icon: '∑', run: () => (writingMaths = true) }
	]);

	function detect() {
		const before = value.slice(0, area.selectionStart);
		const m = before.match(/\[\[([^[\]\n|#]{0,60})$/);
		query = m ? m[1] : null;
		active = 0;
	}

	function accept(i: number) {
		const n = suggestions[i];
		if (!n || query === null) return;
		const caret = area.selectionStart;
		const start = caret - query.length;
		const name = stem(n.path);
		const closes = value.slice(caret, caret + 2) === ']]';
		value = value.slice(0, start) + name + (closes ? '' : ']]') + value.slice(caret);
		query = null;
		setSelection(start + name.length + 2);
	}

	function onkeydown(e: KeyboardEvent) {
		if (vim && !(query !== null && suggestions.length && mode === 'insert') && vimKey(e)) return;
		if (vim && mode !== 'insert') return;
		if (query !== null && suggestions.length) {
			if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
				e.preventDefault();
				active =
					(active + (e.key === 'ArrowDown' ? 1 : suggestions.length - 1)) % suggestions.length;
				return;
			}
			if (e.key === 'Enter' || e.key === 'Tab') {
				e.preventDefault();
				accept(active);
				return;
			}
			if (e.key === 'Escape') {
				e.preventDefault();
				query = null;
				return;
			}
		}
		const mod = e.ctrlKey || e.metaKey;
		const k = e.key.toLowerCase();
		const shortcut =
			mod && !e.shiftKey && k === 'b'
				? t('Bold')
				: mod && !e.shiftKey && k === 'i'
					? t('Italic')
					: mod && e.shiftKey && k === 'x'
						? t('Strikethrough')
						: mod && e.shiftKey && k === 'h'
							? t('Highlight')
							: mod && !e.shiftKey && k === 'e'
								? t('Code')
								: mod && !e.shiftKey && k === 'l'
									? t('Link to a note')
									: null;
		if (shortcut) {
			e.preventDefault();
			tools.find((t) => t.label === shortcut)?.run();
			return;
		}
		if (e.key === 'Enter' && !e.shiftKey && !mod) continueList(e);
	}

	/** Enter in a list item starts the next item; on an empty item it ends the list. */
	function continueList(e: KeyboardEvent) {
		const next = edit.continueList({ value, start: area.selectionStart, end: area.selectionEnd });
		if (!next) return;
		e.preventDefault();
		apply(() => next);
	}
</script>

<div class="editor">
	<div class="toolbar" role="toolbar" aria-label={t('Formatting')} aria-controls={id}>
		{#each tools as tool (tool.label)}
			<button
				type="button"
				onclick={tool.run}
				aria-label={tool.label}
				title={tool.keys ? `${tool.label} (${tool.keys})` : tool.label}>{tool.icon}</button
			>
		{/each}
		{#if pictures}
			<button
				type="button"
				onclick={() => picker?.click()}
				aria-label={t('Insert a picture or PDF')}
				title={t('Insert a picture or PDF (or paste, or drop it here)')}>🖼</button
			>
			<button
				type="button"
				onclick={draw}
				aria-label={t('Draw')}
				title={t('Draw a sketch (on a line with a drawing: edit it)')}>✎</button
			>
			<input
				bind:this={picker}
				type="file"
				accept="image/png,image/jpeg,image/gif,image/webp,image/svg+xml,application/pdf"
				multiple
				hidden
				onchange={(e) => {
					addFiles([...(e.currentTarget.files ?? [])]);
					e.currentTarget.value = '';
				}}
			/>
		{/if}
	</div>
	<!-- Mounted while open only, so the editor's own textarea comes first. -->
	{#if pictures && drawing}<SketchPad
			bind:open={drawing}
			shapes={sketch}
			onsave={saveDrawing}
		/>{/if}
	{#if writingMaths}<MathHelper bind:open={writingMaths} oninsert={insert} />{/if}
	<label for={id} class="visually-hidden">{label}</label>
	<textarea
		{id}
		bind:this={area}
		bind:value
		{rows}
		spellcheck="true"
		aria-autocomplete="list"
		aria-controls={suggestions.length ? `${id}-suggestions` : undefined}
		aria-activedescendant={suggestions.length ? `${id}-s-${active}` : undefined}
		oninput={detect}
		onclick={detect}
		{onkeydown}
		{onpaste}
		{ondrop}
		ondragover={(e) => {
			if (pictures && e.dataTransfer?.types.includes('Files')) e.preventDefault();
		}}
		class:vim-normal={vim && mode !== 'insert'}
		aria-describedby={vim ? `${id}-vim` : undefined}
	></textarea>
	{#if uploading || uploadError}
		<p class="upload-status" role={uploadError ? 'alert' : 'status'}>
			{uploading
				? t(uploading > 1 ? 'Uploading {n} files…' : 'Uploading {n} file…', { n: uploading })
				: uploadError}
		</p>
	{/if}
	{#if vim}
		<p class="vim-status" id="{id}-vim" aria-live="polite">
			{command || `-- ${mode === 'visual-line' ? 'VISUAL LINE' : mode.toUpperCase()} --`}
			<span class="muted">{t('Vim mode · Esc for normal, i to type, :w to save')}</span>
		</p>
	{/if}
	{#if suggestions.length}
		<ul class="suggestions" id="{id}-suggestions" role="listbox" aria-label={t('Notes to link')}>
			{#each suggestions as n, i (n.id)}
				<li
					id="{id}-s-{i}"
					role="option"
					aria-selected={i === active}
					onmousedown={(e) => {
						e.preventDefault();
						accept(i);
					}}
				>
					<strong>{n.title}</strong> <span class="muted">{n.path}</span>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.editor {
		position: relative;
		display: grid;
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--surface);
		overflow: visible;
	}
	.upload-status {
		margin: 0;
		padding: var(--space-1) var(--space-2);
		font-size: var(--text-sm);
		border-top: 1px solid var(--border);
	}
	.toolbar {
		display: flex;
		flex-wrap: wrap;
		gap: 2px;
		padding: var(--space-1);
		border-bottom: 1px solid var(--border);
	}
	.toolbar button {
		font: inherit;
		font-size: var(--text-sm);
		min-width: 2.25rem;
		min-height: 2.25rem;
		border: none;
		border-radius: var(--radius-sm);
		background: none;
		color: var(--fg);
		cursor: pointer;
	}
	.toolbar button:hover {
		background: var(--accent-soft);
	}
	textarea {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		line-height: 1.7;
		border: none;
		background: transparent;
		color: var(--fg);
		padding: var(--space-4);
		resize: vertical;
		min-height: 24rem;
		width: 100%;
		border-radius: 0 0 var(--radius-lg) var(--radius-lg);
	}
	textarea:focus-visible {
		box-shadow: inset 0 0 0 2px var(--accent);
	}
	.suggestions {
		position: absolute;
		left: var(--space-4);
		right: var(--space-4);
		top: 3.5rem;
		z-index: 5;
		list-style: none;
		margin: 0;
		padding: var(--space-1);
		background: var(--surface-solid);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		box-shadow: var(--shadow);
	}
	.suggestions li {
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius-sm);
		cursor: pointer;
		font-size: var(--text-sm);
		overflow-wrap: anywhere;
	}
	.suggestions li[aria-selected='true'] {
		background: var(--accent-soft);
	}
	.vim-normal {
		caret-color: transparent;
	}
	.vim-status {
		margin: var(--space-1) 0 0;
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		display: flex;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
</style>
