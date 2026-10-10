<script lang="ts">
	import { api, message, type Schemas } from './api';
	import { href, label, options, typeNames, type Link, type Option, type Relation } from './links';
	import Button from './ui/Button.svelte';
	import { toast } from './ui/toast.svelte';

	type Hit = Schemas['SearchHit'];

	let {
		id,
		type,
		onchange = () => {}
	}: { id: string; type: string; onchange?: () => void } = $props();

	let relations = $state<Relation[]>([]);
	let outgoing = $state<Link[]>([]);
	let incoming = $state<Link[]>([]);
	let query = $state('');
	let hits = $state<Hit[]>([]);
	let picked = $state<Hit | null>(null);
	let choice = $state(0);
	let timer: ReturnType<typeof setTimeout> | undefined;

	const choices = $derived<Option[]>(picked ? options(relations, type, picked.type) : []);
	const rows = $derived([
		...outgoing.map((l) => ({ link: l, out: true })),
		...incoming.map((l) => ({ link: l, out: false }))
	]);

	async function load() {
		const [r, o, i] = await Promise.all([
			api<{ items: Relation[] }>('/v1/relations'),
			api<{ items: Link[] }>(`/v1/entities/${id}/links?limit=100`),
			api<{ items: Link[] }>(`/v1/entities/${id}/links?direction=incoming&limit=100`)
		]);
		relations = r.items;
		outgoing = o.items;
		incoming = i.items;
	}

	$effect(() => {
		void id;
		load().catch((err) => toast(message(err), 'error'));
	});

	function search() {
		clearTimeout(timer);
		picked = null;
		const q = query.trim();
		if (!q) {
			hits = [];
			return;
		}
		timer = setTimeout(async () => {
			try {
				const res = await api<{ items: Hit[] }>(`/v1/search?q=${encodeURIComponent(q)}&limit=8`);
				hits = res.items.filter((h) => h.id !== id && typeNames[h.type]);
			} catch {
				hits = [];
			}
		}, 200);
	}

	function pick(h: Hit) {
		picked = h;
		choice = 0;
		hits = [];
		query = h.title;
	}

	async function connect(e: SubmitEvent) {
		e.preventDefault();
		const o = choices[choice];
		if (!picked || !o) return;
		try {
			await api('/v1/links', {
				method: 'POST',
				body: o.outgoing
					? { source_id: id, relation: o.relation, target_id: picked.id }
					: { source_id: picked.id, relation: o.relation, target_id: id }
			});
			toast(`Connected to “${picked.title}”.`);
			picked = null;
			query = '';
			await load();
			onchange();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove(l: Link) {
		try {
			await api(`/v1/links/${l.id}`, { method: 'DELETE' });
			await load();
			onchange();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<section class="card connections" aria-labelledby="connections-{id}">
	<h2 id="connections-{id}">Connections</h2>
	{#if rows.length === 0}
		<p class="muted">Not connected to anything yet.</p>
	{:else}
		<ul class="rows">
			{#each rows as { link, out } (link.id + (out ? 'o' : 'i'))}
				<li>
					<span class="rel">{label(link.relation, out)}</span>
					{#if href(link.entity)}
						<a class="grow" href={href(link.entity)}>{link.entity.title}</a>
					{:else}
						<span class="grow">{link.entity.title}</span>
					{/if}
					<span class="chip">{typeNames[link.entity.type] ?? link.entity.type}</span>
					{#if link.relation !== 'references'}
						<Button
							variant="ghost"
							onclick={() => remove(link)}
							aria-label="Remove connection to {link.entity.title}">Remove</Button
						>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	<form class="add" onsubmit={connect}>
		<label for="connect-{id}">Connect to</label>
		<input
			id="connect-{id}"
			type="search"
			placeholder="Search tasks, notes, goals, habits…"
			autocomplete="off"
			bind:value={query}
			oninput={search}
			aria-controls={hits.length ? `connect-results-${id}` : undefined}
		/>
		{#if hits.length}
			<ul class="results" id="connect-results-{id}" aria-label="Matches">
				{#each hits as h (h.id)}
					<li>
						<button type="button" onclick={() => pick(h)}>
							<span class="chip">{typeNames[h.type]}</span>
							{h.title}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
		{#if picked}
			{#if choices.length === 0}
				<p class="muted small">These two cannot be connected.</p>
			{:else}
				<div class="row">
					<label for="relation-{id}" class="visually-hidden">How they connect</label>
					<select id="relation-{id}" bind:value={choice}>
						{#each choices as c, i (c.relation + c.outgoing)}
							<option value={i}>This {c.label.toLowerCase()} “{picked.title}”</option>
						{/each}
					</select>
					<Button type="submit" variant="secondary">Connect</Button>
				</div>
			{/if}
		{/if}
	</form>
</section>

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	@media (max-width: 30rem) {
		.card {
			padding: var(--space-4);
		}
	}
	h2 {
		font-size: var(--text-lg);
	}
	.rows,
	.results {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-4);
	}
	.rows li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
	}
	.rel {
		color: var(--muted);
		font-size: var(--text-sm);
		min-width: 6.5rem;
	}
	.grow {
		flex: 1;
		min-width: 8rem;
		overflow-wrap: anywhere;
	}
	.chip {
		font-size: var(--text-sm);
		border-radius: 999px;
		padding: 0 var(--space-2);
		background: var(--accent-soft);
		color: var(--fg);
	}
	.add {
		display: grid;
		gap: var(--space-2);
	}
	.add label {
		font-weight: 600;
	}
	input,
	select {
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		width: 100%;
		max-width: 100%;
	}
	.results {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface-solid);
		overflow: hidden;
	}
	.results button {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		width: 100%;
		text-align: left;
		font: inherit;
		color: var(--fg);
		background: none;
		border: none;
		padding: var(--space-2) var(--space-3);
		cursor: pointer;
		overflow-wrap: anywhere;
	}
	.results button:hover,
	.results button:focus-visible {
		background: var(--accent-soft);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.row select {
		flex: 1;
		min-width: 0;
		width: auto;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
