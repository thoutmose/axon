<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Roadmap = Schemas['Roadmap'];
	type Source = {
		id: string;
		title: string;
		description: string;
		attribution: string;
		license_url: string;
	};
	let roadmaps = $state<Roadmap[] | null>(null);
	let sources = $state<Source[]>([]);
	let error = $state('');
	let title = $state('');
	let importing = $state(false);

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
						{r.done} of {r.total} skills{r.source !== 'manual' ? ' · imported' : ''}
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

	<section class="card glass imports" aria-labelledby="import-title">
		<h2 id="import-title">Start from an open curriculum</h2>
		{#each sources as s (s.id)}
			<div class="source">
				<div>
					<strong>{s.title}</strong>
					<p class="muted small">{s.description}</p>
					<p class="small attribution">
						{s.attribution}
						<a href={s.license_url} rel="noopener noreferrer" target="_blank">Licence</a>
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
	.imports h2 {
		font-size: var(--text-lg);
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
