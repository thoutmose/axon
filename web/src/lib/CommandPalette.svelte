<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from './api';
	import { capturing } from './capture.svelte';
	import { href, typeNames } from './links';
	import { openDaily } from './newNote';
	import { highlight } from './notes';
	import { toast } from './ui/toast.svelte';

	type Hit = Schemas['SearchHit'];
	type Item = { id: string; label: string; detail?: string; snippet?: string; run: () => unknown };

	let { open = $bindable(false) }: { open: boolean } = $props();
	let dialog: HTMLDialogElement;
	let input = $state<HTMLInputElement>();
	let query = $state('');
	let hits = $state<Hit[]>([]);
	let active = $state(0);
	let timer: ReturnType<typeof setTimeout> | undefined;

	const commands: Item[] = [
		{ id: 'c-note', label: 'New note…', detail: 'Command', run: () => goto('/notes?new=1') },
		{ id: 'c-daily', label: "Open today's note", detail: 'Command', run: openDaily },
		{
			id: 'c-task',
			label: 'Capture a task or note',
			detail: 'Command · n',
			run: () => (capturing.open = true)
		},
		...[
			['Today', '/today'],
			['Tasks', '/tasks'],
			['Notes', '/notes'],
			['Goals', '/goals'],
			['Projects', '/projects'],
			['Habits', '/habits'],
			['Calendar', '/calendar'],
			['Agents', '/agents'],
			['Account', '/account']
		].map(([l, h]) => ({ id: `g-${h}`, label: `Go to ${l}`, detail: 'Page', run: () => goto(h) }))
	];

	const items = $derived.by<Item[]>(() => {
		const q = query.trim().toLowerCase();
		const cmds = commands.filter((c) => !q || c.label.toLowerCase().includes(q));
		const found = hits
			.filter((h) => href(h))
			.map((h) => ({
				id: h.id,
				label: h.title,
				detail: typeNames[h.type] ?? h.type,
				snippet: h.snippet,
				run: () => goto(href(h)!)
			}));
		return [...found, ...cmds].slice(0, 12);
	});

	$effect(() => {
		if (open && !dialog.open) {
			query = '';
			hits = [];
			active = 0;
			dialog.showModal();
			input?.focus();
		}
		if (!open && dialog.open) dialog.close();
	});

	function search() {
		active = 0;
		clearTimeout(timer);
		const q = query.trim();
		if (!q) {
			hits = [];
			return;
		}
		timer = setTimeout(async () => {
			hits = (
				await api<{ items: Hit[] }>(`/v1/search?limit=8&q=${encodeURIComponent(q)}`).catch(() => ({
					items: []
				}))
			).items;
		}, 150);
	}

	async function run(i: number) {
		const item = items[i];
		if (!item) return;
		open = false;
		try {
			await item.run();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function onkeydown(e: KeyboardEvent) {
		if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
			e.preventDefault();
			const n = items.length || 1;
			active = (active + (e.key === 'ArrowDown' ? 1 : n - 1)) % n;
		} else if (e.key === 'Enter') {
			e.preventDefault();
			run(active);
		}
	}
</script>

<dialog
	bind:this={dialog}
	class="palette"
	aria-label="Command center"
	onclose={() => (open = false)}
>
	<label for="palette-input" class="visually-hidden">Search or type a command</label>
	<input
		id="palette-input"
		bind:this={input}
		bind:value={query}
		oninput={search}
		{onkeydown}
		placeholder="Search everything or type a command…"
		autocomplete="off"
		role="combobox"
		aria-expanded={items.length > 0}
		aria-controls="palette-list"
		aria-activedescendant={items.length ? `palette-${active}` : undefined}
	/>
	<ul id="palette-list" role="listbox" aria-label="Results">
		{#each items as item, i (item.id)}
			<li
				id="palette-{i}"
				role="option"
				aria-selected={i === active}
				onclick={() => run(i)}
				onkeydown={() => {}}
				onmousemove={() => (active = i)}
			>
				<span class="label">{item.label}</span>
				<span class="detail">{item.detail}</span>
				{#if item.snippet}
					<!-- highlight() escapes the snippet before marking matches. -->
					<span class="snippet">{@html highlight(item.snippet)}</span>
				{/if}
			</li>
		{/each}
	</ul>
	<p class="hint">↑↓ to choose · Enter to open · Esc to close</p>
</dialog>

<style>
	.palette {
		width: min(40rem, calc(100vw - 2rem));
		margin-top: 12vh;
		padding: 0;
		border: 1px solid var(--border);
		border-radius: var(--radius-xl);
		background: var(--surface-glass);
		color: var(--fg);
		box-shadow: var(--shadow);
		overflow: hidden;
	}
	input {
		width: 100%;
		font: inherit;
		font-size: var(--text-lg);
		padding: var(--space-4) var(--space-6);
		border: none;
		border-bottom: 1px solid var(--border);
		background: transparent;
		color: var(--fg);
	}
	input:focus-visible {
		box-shadow: none;
		outline: none;
	}
	ul {
		list-style: none;
		margin: 0;
		padding: var(--space-2);
		max-height: 55vh;
		overflow-y: auto;
	}
	li {
		display: grid;
		grid-template-columns: 1fr auto;
		gap: 0 var(--space-3);
		padding: var(--space-2) var(--space-4);
		border-radius: var(--radius);
		cursor: pointer;
	}
	li[aria-selected='true'] {
		background: var(--accent-soft);
	}
	.label {
		font-weight: 550;
		overflow-wrap: anywhere;
	}
	.detail {
		color: var(--muted);
		font-size: var(--text-sm);
	}
	.snippet {
		grid-column: 1 / -1;
		color: var(--muted);
		font-size: var(--text-sm);
		overflow-wrap: anywhere;
	}
	.snippet :global(mark) {
		background: var(--accent-soft);
		color: var(--fg);
	}
	.hint {
		margin: 0;
		padding: var(--space-2) var(--space-6);
		border-top: 1px solid var(--border);
		color: var(--muted);
		font-size: var(--text-sm);
	}
</style>
