<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import type { ParaItem, ParaKind } from '$lib/para';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let { kind }: { kind: ParaKind } = $props();

	const status = $derived(page.url.searchParams.get('status') ?? 'active');
	let items = $state<ParaItem[] | null>(null);
	let error = $state('');
	let title = $state('');

	async function load() {
		error = '';
		try {
			items = (await api<{ items: ParaItem[] }>(`/v1/${kind.path}?status=${status}`)).items;
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void status;
		void kind;
		items = null;
		load();
	});

	async function create(body: { title: string; description?: string }) {
		try {
			const it = await api<ParaItem>(`/v1/${kind.path}`, { method: 'POST', body });
			await goto(`/${kind.path}/${it.id}`);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function submit(e: SubmitEvent) {
		e.preventDefault();
		create({ title });
		title = '';
	}

	const unused = $derived(
		kind.templates.filter(
			(t) => !items?.some((i) => i.title.toLowerCase() === t.title.toLowerCase())
		)
	);
</script>

<h1>{kind.title}</h1>
<p class="muted hint">{kind.intro}</p>

<nav aria-label="{kind.title} status" class="tabs">
	<a href="?status=active" aria-current={status === 'active' ? 'page' : undefined}>Active</a>
	<a href="?status=archived" aria-current={status === 'archived' ? 'page' : undefined}>Archived</a>
</nav>

{#if status === 'active'}
	<form class="quick" onsubmit={submit}>
		<label for="new-{kind.type}" class="visually-hidden">New {kind.one}</label>
		<input
			id="new-{kind.type}"
			placeholder={kind.placeholder}
			bind:value={title}
			maxlength="500"
			required
			autocomplete="off"
		/>
		<Button type="submit" variant="secondary">Create</Button>
	</form>
	{#if items && unused.length}
		<div class="templates" role="group" aria-label="Start from a template">
			<span class="muted small">Templates:</span>
			{#each unused as t (t.title)}
				<button onclick={() => create(t)} title={t.description}>＋ {t.title}</button>
			{/each}
		</div>
	{/if}
{/if}

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !items}
	<Status kind="loading" message="Loading {kind.path}…" />
{:else if items.length === 0}
	<Status kind="empty" message={status === 'active' ? kind.empty : `No archived ${kind.path}.`} />
{:else}
	<ul class="grid">
		{#each items as it (it.id)}
			<li>
				<a href="/{kind.path}/{it.id}">{it.title}</a>
				{#if it.description}<p class="muted small">{it.description}</p>{/if}
				<span class="muted small">{it.linked} {it.linked === 1 ? 'connection' : 'connections'}</span
				>
			</li>
		{/each}
	</ul>
{/if}

<style>
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.tabs {
		display: flex;
		gap: var(--space-1);
		margin: var(--space-4) 0;
	}
	.tabs a {
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		color: var(--muted);
		text-decoration: none;
	}
	.tabs a[aria-current='page'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
	}
	.quick {
		display: flex;
		gap: var(--space-2);
		margin-bottom: var(--space-3);
	}
	.quick input {
		flex: 1;
		min-width: 0;
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.templates {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
		margin-bottom: var(--space-6);
	}
	.templates button {
		font: inherit;
		font-size: var(--text-sm);
		border: 1px dashed var(--border);
		background: var(--surface);
		color: var(--fg);
		border-radius: 999px;
		padding: var(--space-1) var(--space-3);
		cursor: pointer;
		min-height: 2rem;
	}
	.templates button:hover {
		border-color: var(--accent);
	}
	.grid {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(18rem, 1fr));
		gap: var(--space-3);
	}
	.grid li {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		display: grid;
		gap: var(--space-1);
		align-content: start;
	}
	.grid a {
		font-weight: 600;
		font-size: var(--text-lg);
	}
	.grid p {
		margin: 0;
	}
</style>
