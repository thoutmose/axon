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
		kind: 'library';
		title: string;
		description: string;
		attribution: string;
		license_url: string | null;
		count: number;
		skills: number;
	};
	const sourceNames: Record<string, string> = $derived({
		library: t('from the library'),
		tutor: t('drafted by the Tutor')
	});
	let roadmaps = $state<Roadmap[] | null>(null);
	let radar = $state<{ hard: RadarKind; soft: RadarKind } | null>(null);
	const by = $derived({
		roadmap: t('One spoke per roadmap.'),
		section: t('One spoke per section.')
	});
	let sources = $state<Source[]>([]);
	let error = $state('');
	let title = $state('');
	let importing = $state(false);
	// The library is one roadmap with its plan (EP-95); once started, it opens.
	const started = (s: Source) =>
		roadmaps?.find((r) => r.source === 'library' && r.title === s.title);

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

	async function importSource(s: Source, place = false) {
		importing = true;
		try {
			const r = await api<Roadmap>('/v1/roadmaps/import', {
				method: 'POST',
				body: { source: s.id, place }
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
				{#each [{ title: t('Hard skills'), kind: radar.hard }, { title: t('Soft skills'), kind: radar.soft }] as r (r.title)}
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
			message={t('No roadmaps yet. Create one above, or start the roadmap below.')}
		/>
	{/if}

	{#each sources as s (s.id)}
		<section class="card glass library" aria-labelledby="library-{s.id}">
			<h2 id="library-{s.id}">{s.title}</h2>
			{#each s.description.split('\n\n') as para, i (i)}<p class="small">{para}</p>{/each}
			<p class="muted small">
				{t('{skills} skills · {count} resources', { skills: s.skills, count: s.count })} · {t(
					'Starting it adds the goal, a milestone and a project per phase with its exit check, the 42 projects, the habits, the first week’s tasks, its areas, resources, and note.'
				)}
			</p>
			<div class="row">
				{#if started(s)}
					<Button href="/roadmaps/{started(s)?.id}">{t('Open the roadmap')}</Button>
				{:else}
					<Button busy={importing} onclick={() => importSource(s)}>{t('Start this roadmap')}</Button
					>
					<Button variant="secondary" busy={importing} onclick={() => importSource(s, true)}
						>{t('Start at my level')}</Button
					>
				{/if}
			</div>
			<p class="small attribution">{s.attribution}</p>
		</section>
	{/each}
	<p class="muted small">
		{t(
			'roadmap.sh roadmaps are not imported: their licence allows personal viewing but not copying into other apps. Open them at'
		)}
		<a href="https://roadmap.sh" rel="noopener noreferrer" target="_blank">roadmap.sh</a>
		{t('and add the skills you choose here.')}
	</p>
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
	.row {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
		margin-top: var(--space-3);
	}
	.attribution {
		color: var(--muted);
	}
</style>
