<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api, message } from '$lib/api';
	import Status from '$lib/ui/Status.svelte';

	// The A of PARA: everything archived, from every kind, in one place.
	// Archiving keeps links and history; open an item to restore it.
	type Row = { id: string; title: string };
	const groups: { title: string; url: string; href: (id: string) => string; pick?: string }[] = [
		{
			title: 'Projects',
			url: '/v1/projects?status=archived',
			href: (id) => `/projects/${id}`
		},
		{ title: 'Areas', url: '/v1/areas?status=archived', href: (id) => `/areas/${id}` },
		{
			title: 'Resources',
			url: '/v1/resources?status=archived',
			href: (id) => `/resources/${id}`
		},
		{ title: 'Goals', url: '/v1/goals?status=archived', href: (id) => `/goals/${id}` },
		{ title: 'Habits', url: '/v1/habits?status=archived', href: (id) => `/habits/${id}` },
		{
			title: 'Notes in the archive folder',
			url: '/v1/notes?para=archive&limit=100',
			href: (id) => `/notes/${id}`
		}
	];
	let lists = $state<(Row[] | null)[]>(groups.map(() => null));
	let error = $state('');

	async function load() {
		error = '';
		try {
			lists = await Promise.all(
				groups.map(async (g) => (await api<{ items: Row[] }>(g.url)).items)
			);
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		load();
	});
	const total = $derived(lists.reduce((n, l) => n + (l?.length ?? 0), 0));
</script>

<svelte:head><title>{t('Archive · Axon')}</title></svelte:head>

<h1>{t('Archive')}</h1>
<p class="muted hint">
	{t(
		'Inactive projects, areas, resources, goals, habits, and notes. Nothing here is lost: links and history stay, and you can restore any item from its page.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if lists.some((l) => l === null)}
	<Status kind="loading" message={t('Loading the archive…')} />
{:else if total === 0}
	<Status kind="empty" message={t('Nothing archived yet.')} />
{:else}
	<div class="grid">
		{#each groups as g, i (g.title)}
			{#if lists[i]?.length}
				<section class="card" aria-labelledby="archive-{i}">
					<h2 id="archive-{i}">{t(g.title)} <span class="muted">{lists[i]?.length}</span></h2>
					<ul>
						{#each lists[i] ?? [] as row (row.id)}
							<li><a href={g.href(row.id)}>{row.title}</a></li>
						{/each}
					</ul>
				</section>
			{/if}
		{/each}
	</div>
{/if}

<style>
	.hint {
		font-size: var(--text-sm);
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(18rem, 1fr));
		gap: var(--space-4);
		align-items: start;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	h2 {
		font-size: var(--text-lg);
		margin-top: 0;
	}
	ul {
		margin: 0;
		padding-left: var(--space-4);
		display: grid;
		gap: var(--space-1);
	}
</style>
