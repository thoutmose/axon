<script lang="ts">
	import { tick } from 'svelte';
	import { canUpload, upload } from './noteFiles.svelte';
	import { noteIndex } from './noteIndex.svelte';
	import { stem } from './notes';
	import { handle, newVim, type Mode } from './vim';

	let {
		value = $bindable(''),
		id,
		label = 'Markdown',
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
				uploadError = 'Only PNG, JPEG, GIF, WebP, SVG pictures and PDFs can go in a note.';
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

	/** Wraps the selection (or inserts a placeholder) in before…after. */
	function wrap(before: string, after = before, placeholder = 'text') {
		const { selectionStart: s, selectionEnd: e } = area;
		const picked = value.slice(s, e) || placeholder;
		value = value.slice(0, s) + before + picked + after + value.slice(e);
		setSelection(s + before.length, s + before.length + picked.length);
	}

	/** Toggles a prefix on every selected line. */
	function prefixLines(prefix: string) {
		const { selectionStart: s, selectionEnd: e } = area;
		const start = value.lastIndexOf('\n', s - 1) + 1;
		const end = value.indexOf('\n', e) === -1 ? value.length : value.indexOf('\n', e);
		const lines = value.slice(start, end).split('\n');
		const all = lines.every((l) => l.startsWith(prefix));
		const next = lines.map((l) => (all ? l.slice(prefix.length) : prefix + l)).join('\n');
		value = value.slice(0, start) + next + value.slice(end);
		setSelection(start, start + next.length);
	}

	function insert(text: string) {
		const { selectionStart: s, selectionEnd: e } = area;
		value = value.slice(0, s) + text + value.slice(e);
		setSelection(s + text.length);
	}

	const tools: { label: string; keys?: string; icon: string; run: () => void }[] = [
		{ label: 'Bold', keys: 'Ctrl+B', icon: 'B', run: () => wrap('**') },
		{ label: 'Italic', keys: 'Ctrl+I', icon: 'I', run: () => wrap('*') },
		{ label: 'Strikethrough', keys: 'Ctrl+Shift+X', icon: 'S', run: () => wrap('~~') },
		{ label: 'Highlight', keys: 'Ctrl+Shift+H', icon: 'H̲', run: () => wrap('==') },
		{ label: 'Heading', icon: '#', run: () => prefixLines('## ') },
		{ label: 'Bulleted list', icon: '•', run: () => prefixLines('- ') },
		{ label: 'Checklist', icon: '☐', run: () => prefixLines('- [ ] ') },
		{ label: 'Quote', icon: '❝', run: () => prefixLines('> ') },
		{ label: 'Code', keys: 'Ctrl+E', icon: '</>', run: () => wrap('`') },
		{ label: 'Link to a note', keys: 'Ctrl+L', icon: '[[ ]]', run: () => wrap('[[', ']]', 'note') },
		{
			label: 'Table',
			icon: '▦',
			run: () => insert('\n| Column | Column |\n| --- | --- |\n| | |\n')
		},
		{ label: 'Callout', icon: 'ⓘ', run: () => insert('\n> [!NOTE] Title\n> Text\n') }
	];

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
				? 'Bold'
				: mod && !e.shiftKey && k === 'i'
					? 'Italic'
					: mod && e.shiftKey && k === 'x'
						? 'Strikethrough'
						: mod && e.shiftKey && k === 'h'
							? 'Highlight'
							: mod && !e.shiftKey && k === 'e'
								? 'Code'
								: mod && !e.shiftKey && k === 'l'
									? 'Link to a note'
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
		const s = area.selectionStart;
		if (s !== area.selectionEnd) return;
		const lineStart = value.lastIndexOf('\n', s - 1) + 1;
		const line = value.slice(lineStart, s);
		const m = line.match(/^(\s*)(- \[[ xX]\] |[-*+] |(\d+)[.)] )(.*)$/);
		if (!m) return;
		e.preventDefault();
		if (m[4].trim() === '') {
			value = value.slice(0, lineStart) + value.slice(s);
			setSelection(lineStart);
			return;
		}
		const marker = m[3] ? `${Number(m[3]) + 1}. ` : m[2].startsWith('- [') ? '- [ ] ' : m[2];
		insert(`\n${m[1]}${marker}`);
	}
</script>

<div class="editor">
	<div class="toolbar" role="toolbar" aria-label="Formatting" aria-controls={id}>
		{#each tools as t (t.label)}
			<button
				type="button"
				onclick={t.run}
				aria-label={t.label}
				title={t.keys ? `${t.label} (${t.keys})` : t.label}>{t.icon}</button
			>
		{/each}
		{#if pictures}
			<button
				type="button"
				onclick={() => picker?.click()}
				aria-label="Insert a picture or PDF"
				title="Insert a picture or PDF (or paste, or drop it here)">🖼</button
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
			{uploading ? `Uploading ${uploading} file${uploading > 1 ? 's' : ''}…` : uploadError}
		</p>
	{/if}
	{#if vim}
		<p class="vim-status" id="{id}-vim" aria-live="polite">
			{command || `-- ${mode === 'visual-line' ? 'VISUAL LINE' : mode.toUpperCase()} --`}
			<span class="muted">Vim mode · Esc for normal, i to type, :w to save</span>
		</p>
	{/if}
	{#if suggestions.length}
		<ul class="suggestions" id="{id}-suggestions" role="listbox" aria-label="Notes to link">
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
