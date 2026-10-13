<script lang="ts">
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Radar from '$lib/charts/Radar.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import { t } from '$lib/i18n.svelte';

	type Roadmap = Schemas['Roadmap'];
	type RadarKind = Schemas['RadarKind'];
	type Source = {
		id: string;
		kind: 'curriculum' | 'library';
		title: string;
		description: string;
		attribution: string;
		license_url: string | null;
		count: number;
		skills: number;
	};
	const sourceNames: Record<string, string> = {
		ossu: 'OSSU',
		library: 'from the library',
		tutor: 'drafted by the Tutor'
	};
	let roadmaps = $state<Roadmap[] | null>(null);
	let radar = $state<{ hard: RadarKind; soft: RadarKind } | null>(null);
	const by = { roadmap: 'One spoke per roadmap.', section: 'One spoke per section.' };
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
			const [r, s, k] = await Promise.all([
				api<{ items: Roadmap[] }>('/v1/roadmaps'),
				api<{ items: Source[] }>('/v1/roadmaps/sources'),
				api<{ hard: RadarKind; soft: RadarKind }>('/v1/roadmaps/radar')
			]);
			roadmaps = r.items;
			sources = s.items;
			radar = k;
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

<svelte:head><title>{t('Roadmaps')} · Axon</title></svelte:head>

<h1>{t('Roadmaps')}</h1>
<LearnTabs />
<AgentDock {...pageAgents.roadmaps} onapplied={() => load()} />
<p class="muted hint">
	{t('Plan what to learn as skills with prerequisites, and watch the graph fill in.')}
</p>

<aside class="assess glass" aria-labelledby="assess-title">
	<div>
		<h2 id="assess-title">{t('New here? Start with the skills assessment')}</h2>
		<p class="muted small">
			{t(
				'Your situation, goals, and a few quick checks per domain. Roadmaps then start at your level, and the Tutor agent knows where you are.'
			)}
		</p>
	</div>
	<Button href="/roadmaps/assessment" variant="secondary">{t('Take the assessment')}</Button>
</aside>

<form class="quick" onsubmit={create}>
	<label for="new-roadmap" class="visually-hidden">{t('New roadmap')}</label>
	<input
		id="new-roadmap"
		placeholder={t('New roadmap, e.g. Japanese N4…')}
		bind:value={title}
		maxlength="500"
		required
		autocomplete="off"
	/>
	<Button type="submit" variant="secondary">{t('Create')}</Button>
</form>

{#if error}
	<Status kind="error" message={error} />
{:else if !roadmaps}
	<Status kind="loading" message={t('Loading roadmaps…')} />
{:else}
	{#if radar && radar.hard.spokes.length + radar.soft.spokes.length}
		<section class="card glass skills" aria-labelledby="skills-title">
			<h2 id="skills-title">{t('Skills acquired')}</h2>
			<p class="muted small">
				{t(
					'How much of each area you have learnt: the filled shape is skills done, the dashed one adds those you are learning. Skipped skills are not counted.'
				)}
			</p>
			<div class="radars">
				{#each [{ title: 'Hard skills', kind: radar.hard }, { title: 'Soft skills', kind: radar.soft }] as r (r.title)}
					{#if r.kind.spokes.length}
						<div>
							<Radar title={t(r.title)} spokes={r.kind.spokes} />
							<p class="muted small">{t(by[r.kind.by])}</p>
						</div>
					{/if}
				{/each}
			</div>
		</section>
	{/if}

	{#if roadmaps.length}
		<ul class="grid">
			{#each roadmaps as r (r.id)}
				<li class="card glass">
					<a href="/roadmaps/{r.id}">{r.title}</a>
					<div
						class="bar"
						role="progressbar"
						aria-label={t('Progress of {title}', { title: r.title })}
						aria-valuemin="0"
						aria-valuemax={r.total || 1}
						aria-valuenow={r.done}
					>
						<span style:width="{r.total ? (r.done / r.total) * 100 : 0}%"></span>
					</div>
					<p class="muted small">
						{t('{done} of {total} skills', { done: r.done, total: r.total })}{sourceNames[r.source]
							? ` · ${t(sourceNames[r.source])}`
							: ''}
					</p>
				</li>
			{/each}
		</ul>
	{:else}
		<Status
			kind="empty"
			message={t('No roadmaps yet. Create one above, or start from an open curriculum below.')}
		/>
	{/if}

	<section class="card glass library" aria-labelledby="library-title">
		<h2 id="library-title">{t('The library')}</h2>
		<p class="muted small">
			{t(
				"Roadmaps built from the world's most recommended books, papers, university lectures, courses, and channels, one per domain. Import one, then make it yours: rename, reorder, add, or remove anything. Or ask the Tutor agent on the"
			)}
			<a href="/agents">{t('Agents')}</a>
			{t('page to draft one for your goal.')}
		</p>
		<label for="library-filter" class="visually-hidden">{t('Filter domains')}</label>
		<input
			id="library-filter"
			class="filter"
			type="search"
			placeholder={t('Filter domains, e.g. kubernetes, data, rust…')}
			bind:value={filter}
		/>
		<ul class="domains">
			{#each domains as s (s.id)}
				<li>
					<strong>{s.title}</strong>
					<p class="muted small">{s.description}</p>
					<div class="row">
						<span class="muted small"
							>{t('{skills} skills · {count} resources', {
								skills: s.skills,
								count: s.count
							})}</span
						>
						<Button
							variant="secondary"
							busy={importing}
							onclick={() => importSource(s)}
							aria-label={t('Import {title}', { title: s.title })}>{t('Import')}</Button
						>
					</div>
				</li>
			{:else}
				<li class="muted">{t('No domain matches “{filter}”.', { filter })}</li>
			{/each}
		</ul>
		{#if sources.find((s) => s.kind === 'library')}
			<p class="small attribution">{sources.find((s) => s.kind === 'library')?.attribution}</p>
		{/if}
	</section>

	<section class="card glass imports" aria-labelledby="import-title">
		<h2 id="import-title">{t('Start from an open curriculum')}</h2>
		{#each curricula as s (s.id)}
			<div class="source">
				<div>
					<strong>{s.title}</strong>
					<p class="muted small">{s.description}</p>
					<p class="small attribution">
						{s.attribution}
						{#if s.license_url}<a href={s.license_url} rel="noopener noreferrer" target="_blank"
								>{t('Licence')}</a
							>{/if}
					</p>
				</div>
				<Button variant="secondary" busy={importing} onclick={() => importSource(s)}
					>{t('Import')}</Button
				>
			</div>
		{/each}
		<p class="muted small">
			{t(
				'roadmap.sh roadmaps are not imported: their licence allows personal viewing but not copying into other apps. Open them at'
			)}
			<a href="https://roadmap.sh" rel="noopener noreferrer" target="_blank">roadmap.sh</a>
			{t('and add the skills you choose here.')}
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
	.skills {
		margin-bottom: var(--space-6);
	}
	.radars {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 22rem), 1fr));
		gap: var(--space-6);
	}
	.skills h2,
	.imports h2,
	.library h2 {
		font-size: var(--text-lg);
	}
	.library {
		margin-bottom: var(--space-6);
	}
	.assess {
		display: flex;
		gap: var(--space-4);
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--surface);
		padding: var(--space-4);
		margin-top: var(--space-4);
	}
	.assess h2 {
		font-size: var(--text-lg);
		margin: 0;
	}
	.assess p {
		margin: var(--space-1) 0 0;
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
