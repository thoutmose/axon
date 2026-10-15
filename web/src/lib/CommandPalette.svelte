<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from './api';
	import { capturing } from './capture.svelte';
	import { href, typeNames } from './links';
	import { iconFor, iconTheme } from './preferences';
	import { openDaily } from './newNote';
	import { highlight } from './notes';
	import { searchPages } from './search.svelte';
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

	const commands: Item[] = $derived([
		{ id: 'c-note', label: t('New note…'), detail: t('Command'), run: () => goto('/notes?new=1') },
		{ id: 'c-daily', label: t("Open today's note"), detail: t('Command'), run: openDaily },
		{
			id: 'c-task',
			label: t('Capture a task or note'),
			detail: t('Command · n'),
			run: () => (capturing.open = true)
		},
		...searchPages.map((p) => ({
			id: `g-${p.href}`,
			label: t('Go to {l}', { l: t(p.label) }),
			detail: t('Page'),
			run: () => goto(p.href)
		}))
	]);

	const items = $derived.by<Item[]>(() => {
		const q = query.trim().toLowerCase();
		const cmds = commands.filter((c) => !q || c.label.toLowerCase().includes(q));
		const found = hits
			.filter((h) => href(h))
			.map((h) => ({
				id: h.id,
				label: h.title,
				detail: `${iconFor(h.type, iconTheme())} ${t(typeNames[h.type] ?? h.type)}`.trim(),
				snippet: h.snippet,
				run: () => goto(href(h)!)
			}));
		// Notes close in meaning (EP-62), when search by meaning is set up; not
		// repeated when the word search already found them.
		const seen = new Set(found.map((f) => f.id));
		const meant = meaning
			.filter((m) => !seen.has(m.id))
			.slice(0, 4)
			.map((m) => ({
				id: `meaning-${m.id}`,
				label: m.title,
				detail: t('Close in meaning'),
				snippet: '',
				run: () => goto(`/notes/${m.id}`)
			}));
		return [...found, ...meant, ...cmds].slice(0, 14);
	});

	$effect(() => {
		if (open && !dialog.open) {
			query = '';
			hits = [];
			meaning = [];
			active = 0;
			dialog.showModal();
			input?.focus();
		}
		if (!open && dialog.open) dialog.close();
	});

	let meaning = $state<{ id: string; title: string; score: number }[]>([]);
	let meaningOn = true; // off for this session once the API says it is not set up
	function search() {
		active = 0;
		clearTimeout(timer);
		const q = query.trim();
		if (!q) {
			hits = [];
			meaning = [];
			return;
		}
		timer = setTimeout(async () => {
			hits = (
				await api<{ items: Hit[] }>(`/v1/search?limit=8&q=${encodeURIComponent(q)}`).catch(() => ({
					items: []
				}))
			).items;
			if (meaningOn && q.length >= 3) {
				meaning = await api<{ items: { id: string; title: string; score: number }[] }>(
					`/v1/notes/semantic?q=${encodeURIComponent(q)}`
				)
					.then((r) => r.items.filter((m) => m.score >= 0.3))
					.catch(() => {
						meaningOn = false;
						return [];
					});
			}
		}, 250);
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
	aria-label={t('Command center')}
	onclose={() => (open = false)}
>
	<label for="palette-input" class="visually-hidden">{t('Search or type a command')}</label>
	<input
		id="palette-input"
		bind:this={input}
		bind:value={query}
		oninput={search}
		{onkeydown}
		placeholder={t('Search everything or type a command…')}
		autocomplete="off"
		role="combobox"
		aria-expanded={items.length > 0}
		aria-controls="palette-list"
		aria-activedescendant={items.length ? `palette-${active}` : undefined}
	/>
	<ul id="palette-list" role="listbox" aria-label={t('Results')}>
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
	<p class="hint">{t('↑↓ to choose · Enter to open · Esc to close')}</p>
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
