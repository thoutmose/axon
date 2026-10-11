<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Roadmap = Schemas['Roadmap'];
	type Source = {
		id: string;
		kind: 'curriculum' | 'library';
		title: string;
		description: string;
		attribution: string;
		license_url: string | null;
		count: number;
	};
	const sourceNames: Record<string, string> = {
		ossu: 'OSSU',
		library: 'from the library',
		tutor: 'drafted by the Tutor'
	};
	let roadmaps = $state<Roadmap[] | null>(null);
	let sources = $state<Source[]>([]);
	let error = $state('');
	let title = $state('');
	let importing = $state(false);
	let filter = $state('');
	const curricula = $derived(sources.filter((s) => s.kind === 'curriculum'));
	const domains = $derived(
		sources.filter(
			(s) =>
				s.kind === 'library' &&
				`${s.title} ${s.description}`.toLowerCase().includes(filter.trim().toLowerCase())
		)
	);

	async function load() {
		try {
			const [r, s] = await Promise.all([
				api<{ items: Roadmap[] }>('/v1/roadmaps'),
				api<{ items: Source[] }>('/v1/roadmaps/sources')
			]);
			roadmaps = r.items;
			sources = s.items;
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function create(e: SubmitEvent) {
		e.preventDefault();
		try {
			const r = await api<Roadmap>('/v1/roadmaps', { method: 'POST', body: { title } });
			await goto(`/roadmaps/${r.id}`);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function importSource(s: Source) {
		importing = true;
		try {
			const r = await api<Roadmap>('/v1/roadmaps/import', {
				method: 'POST',
				body: { source: s.id }
			});
			await goto(`/roadmaps/${r.id}`);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			importing = false;
		}
	}
</script>

<svelte:head><title>Roadmaps · Axon</title></svelte:head>

<h1>Roadmaps</h1>
<p class="muted hint">
	Plan what to learn as skills with prerequisites, and watch the graph fill in.
</p>

<form class="quick" onsubmit={create}>
	<label for="new-roadmap" class="visually-hidden">New roadmap</label>
	<input
		id="new-roadmap"
		placeholder="New roadmap, e.g. Japanese N4…"
		bind:value={title}
		maxlength="500"
		required
		autocomplete="off"
	/>
	<Button type="submit" variant="secondary">Create</Button>
</form>

{#if error}
	<Status kind="error" message={error} />
{:else if !roadmaps}
	<Status kind="loading" message="Loading roadmaps…" />
{:else}
	{#if roadmaps.length}
		<ul class="grid">
			{#each roadmaps as r (r.id)}
				<li class="card glass">
					<a href="/roadmaps/{r.id}">{r.title}</a>
					<div
						class="bar"
						role="progressbar"
						aria-label="Progress of {r.title}"
						aria-valuemin="0"
						aria-valuemax={r.total || 1}
						aria-valuenow={r.done}
					>
						<span style:width="{r.total ? (r.done / r.total) * 100 : 0}%"></span>
					</div>
					<p class="muted small">
						{r.done} of {r.total} skills{sourceNames[r.source] ? ` · ${sourceNames[r.source]}` : ''}
					</p>
				</li>
			{/each}
		</ul>
	{:else}
		<Status
			kind="empty"
			message="No roadmaps yet. Create one above, or start from an open curriculum below."
		/>
	{/if}

	<section class="card glass library" aria-labelledby="library-title">
		<h2 id="library-title">The library</h2>
		<p class="muted small">
			Roadmaps built from the world's most recommended books, papers, university lectures, courses,
			and channels, one per domain. Import one, then make it yours: rename, reorder, add, or remove
			anything. Or ask the Tutor agent on the <a href="/agents">Agents</a> page to draft one for your
			goal.
		</p>
		<label for="library-filter" class="visually-hidden">Filter domains</label>
		<input
			id="library-filter"
			class="filter"
			type="search"
			placeholder="Filter domains, e.g. kubernetes, data, rust…"
			bind:value={filter}
		/>
		<ul class="domains">
			{#each domains as s (s.id)}
				<li>
					<strong>{s.title}</strong>
					<p class="muted small">{s.description}</p>
					<div class="row">
						<span class="muted small">{s.count} resources</span>
						<Button
							variant="secondary"
							busy={importing}
							onclick={() => importSource(s)}
							aria-label="Import {s.title}">Import</Button
						>
					</div>
				</li>
			{:else}
				<li class="muted">No domain matches “{filter}”.</li>
			{/each}
		</ul>
		{#if sources.find((s) => s.kind === 'library')}
			<p class="small attribution">{sources.find((s) => s.kind === 'library')?.attribution}</p>
		{/if}
	</section>

	<section class="card glass imports" aria-labelledby="import-title">
		<h2 id="import-title">Start from an open curriculum</h2>
		{#each curricula as s (s.id)}
			<div class="source">
				<div>
					<strong>{s.title}</strong>
					<p class="muted small">{s.description}</p>
					<p class="small attribution">
						{s.attribution}
						{#if s.license_url}<a href={s.license_url} rel="noopener noreferrer" target="_blank"
								>Licence</a
							>{/if}
					</p>
				</div>
				<Button variant="secondary" busy={importing} onclick={() => importSource(s)}>Import</Button>
			</div>
		{/each}
		<p class="muted small">
			roadmap.sh roadmaps are not imported: their licence allows personal viewing but not copying
			into other apps. Open them at
			<a href="https://roadmap.sh" rel="noopener noreferrer" target="_blank">roadmap.sh</a> and add the
			skills you choose here.
		</p>
	</section>
{/if}

<style>
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.quick {
		display: flex;
		gap: var(--space-2);
		margin: var(--space-4) 0 var(--space-6);
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
	.grid {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-6);
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 16rem), 1fr));
		gap: var(--space-3);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	.card a {
		font-weight: 650;
		color: var(--fg);
		text-decoration: none;
	}
	.bar {
		height: 0.4rem;
		border-radius: 999px;
		background: var(--accent-soft);
		overflow: hidden;
		margin-top: var(--space-3);
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--ok);
	}
	.card p {
		margin: var(--space-2) 0 0;
	}
	.imports h2,
	.library h2 {
		font-size: var(--text-lg);
	}
	.library {
		margin-bottom: var(--space-6);
	}
	.filter {
		font: inherit;
		width: 100%;
		max-width: 28rem;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		margin: var(--space-2) 0 var(--space-4);
	}
	.domains {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 17rem), 1fr));
		gap: var(--space-3);
	}
	.domains li {
		display: grid;
		gap: var(--space-1);
		align-content: space-between;
		padding: var(--space-3) var(--space-4);
		border: 1px solid var(--border);
		border-radius: var(--radius);
	}
	.domains p {
		margin: 0;
	}
	.row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
	}
	.source {
		display: flex;
		gap: var(--space-4);
		align-items: flex-start;
		justify-content: space-between;
		flex-wrap: wrap;
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--border);
	}
	.source > div {
		flex: 1;
		min-width: 14rem;
	}
	.attribution {
		color: var(--muted);
	}
</style>
