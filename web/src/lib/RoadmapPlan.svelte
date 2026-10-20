<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, apiResponse, message } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import { formatDate } from '$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import Status from '$lib/ui/Status.svelte';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// The plan on the roadmap's page (EP-142): each phase as the milestone it
	// became, with its dates, project and board, exit check, 42 projects, and
	// the certifications taken at its end. With a library source instead of
	// a roadmap, the plan as starting it today would lay it out, so its
	// milestones can be read before it is started.
	let { roadmapId, source }: { roadmapId?: string; source?: string } = $props();
	const preview = $derived(!roadmapId);

	type Plan =
		paths['/v1/roadmaps/{id}/plan']['get']['responses'][200]['content']['application/json'];
	type Phase = Plan['phases'][number];
	let plan = $state<Plan | null>(null);
	let error = $state('');

	async function load() {
		error = '';
		try {
			plan = await api<Plan>(
				roadmapId ? `/v1/roadmaps/${roadmapId}/plan` : `/v1/roadmaps/sources/${source}/plan`
			);
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void roadmapId;
		void source;
		load();
	});

	const current = $derived(plan?.phases.find((p) => p.state === 'current') ?? null);
	const done = $derived(plan?.phases.filter((p) => p.milestone?.done).length ?? 0);
	const long = (d: string) =>
		formatDate(d, { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });
	const short = (d: string) => formatDate(d, { day: 'numeric', month: 'short', year: 'numeric' });
	const stateLabel: Record<string, string> = $derived({
		past: t('Past'),
		current: t('Now'),
		future: t('Coming')
	});
	const certStatus: Record<string, string> = $derived({
		planned: t('Planned'),
		booked: t('Booked'),
		passed: t('Passed'),
		failed: t('Failed')
	});

	// A roadmap started from an earlier version of the plan starts the
	// current one; the earlier one stays until the person removes it.
	let starting = $state(false);
	async function startNew() {
		starting = true;
		try {
			const r = await api<{ id: string }>('/v1/roadmaps/import', {
				method: 'POST',
				body: { source: 'top5-engineer' }
			});
			await goto(`/roadmaps/${r.id}`);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			starting = false;
		}
	}
	const certCount = $derived(plan?.phases.reduce((n, p) => n + p.certifications.length, 0) ?? 0);

	// A certification the plan gained after the roadmap was started (EP-146)
	// is planned for the end of its phase in one click.
	async function planCert(p: Phase, c: Phase['certifications'][number]) {
		try {
			await api('/v1/certifications', {
				method: 'POST',
				body: {
					name: c.name,
					code: c.code,
					provider: c.provider,
					url: c.url || null,
					status: 'planned',
					target_on: p.end,
					cost_cents: c.cost_cents,
					currency: 'USD',
					validity_months: c.validity_months
				}
			});
			await load();
			toast(t('{code} is planned for {date}.', { code: c.code, date: short(p.end) }));
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// The plan's resources as notes (EP-154): written at the start; this
	// adds the ones a roadmap started earlier lacks.
	let writing = $state(false);
	async function resourceNotes() {
		writing = true;
		try {
			const r = await api<{ written: number }>(`/v1/roadmaps/${roadmapId}/resource-notes`, {
				method: 'POST'
			});
			toast(
				r.written
					? t('{n} resource notes written in 3-Resources/.', { n: r.written })
					: t('Every resource already has its note.')
			);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			writing = false;
		}
	}

	// What the plan has gained since it was started (EP-166): its goals per
	// group of domains, its habits, and its areas, added where missing.
	let completing = $state(false);
	async function complete() {
		completing = true;
		try {
			const r = await api<{
				areas: number;
				goals: number;
				habits: number;
				milestones: number;
				projects: number;
			}>(`/v1/roadmaps/${roadmapId}/plan/complete`, { method: 'POST' });
			toast(
				r.areas + r.goals + r.habits + r.milestones + r.projects
					? t(
							'Added {goals} goals, {milestones} milestones, {projects} projects, {habits} habits, and {areas} areas.',
							r
						)
					: t('You already have every goal, milestone, project, habit, and area of the plan.')
			);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			completing = false;
		}
	}

	async function toggle(p: Phase) {
		if (!p.milestone) return;
		try {
			const { etag } = await apiResponse(`/v1/milestones/${p.milestone.id}`);
			await api(`/v1/milestones/${p.milestone.id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: { done: !p.milestone.done }
			});
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !plan}
	<Status kind="loading" message={t('Loading the plan…')} />
{:else if !plan.started && !preview}
	<Status
		kind="empty"
		message={t('This roadmap has no plan: its skills are in the list and the skill graph.')}
	/>
{:else}
	{#if plan.newer}
		<aside class="notice" role="status">
			<p>
				{t('A newer version of this plan is started: this one is kept for its history.')}
				<a href="/roadmaps/{plan.newer}">{t('Open the new version')}</a>
			</p>
		</aside>
	{:else if plan.outdated}
		<aside class="notice" role="status">
			<p>
				<strong>{t('This roadmap was started from an earlier version of the plan.')}</strong>
				{t(
					'The plan below is the current one: 18 phases from Monday 19 October 2026, a certification for every domain, and an assessment retaken at each phase’s end. Starting it creates its goal, milestones, and projects; exams and habits you already have are kept, and the calendar shows only the new version. Move this roadmap to the trash afterwards if you no longer need it.'
				)}
			</p>
			<Button busy={starting} onclick={startNew}>{t('Start the new version')}</Button>
		</aside>
	{/if}
	<section class="card summary" aria-labelledby="plan-title{preview ? '-preview' : ''}">
		<h2 id="plan-title{preview ? '-preview' : ''}">
			{preview ? t('The plan, phase by phase') : t('The plan')}
		</h2>
		<p class="lead">
			{#if preview}
				{t('Starts on {date}', { date: long(plan.start) })}
			{:else if current}
				{t('Phase {n} of {total}: {title}', {
					n: current.n,
					total: plan.phases.length,
					title: current.title.replace(/^Phase \d+: /, '')
				})}
			{:else if plan.phases[0] && plan.phases[0].state === 'future'}
				{t('Starts on {date}', { date: long(plan.start) })}
			{:else}
				{t('Plan finished on {date}', { date: long(plan.end) })}
			{/if}
		</p>
		<p class="muted">
			{#if preview}
				{t('{start} → {end} · {total} milestones · {certs} certifications', {
					start: short(plan.start),
					end: short(plan.end),
					total: plan.phases.length,
					certs: certCount
				})}
			{:else}
				{t('{start} → {end} · {done} of {total} milestones reached', {
					start: short(plan.start),
					end: short(plan.end),
					done,
					total: plan.phases.length
				})}
			{/if}
			{#if plan.goal_id}· <a href="/goals/{plan.goal_id}">{t('Open the goal')}</a>{/if}
			{#if !preview}· <a href="/notes?para=resources">{t('Its resources as notes')}</a>{/if}
		</p>
		{#if plan.behind_days > 0}
			<p class="behind">
				{t('{n} days behind: a phase is over and its milestone is not reached.', {
					n: plan.behind_days
				})}
			</p>
		{/if}
		<ol class="strip" aria-label={t('Phases')}>
			{#each plan.phases as p (p.n)}
				<li
					class={p.state}
					class:reached={p.milestone?.done}
					title="{p.title} · {short(p.start)} → {short(p.end)}"
				>
					<a href="#{preview ? 'preview-' : ''}phase-{p.n}"
						><span class="visually-hidden">{p.title}</span>{p.n}</a
					>
				</li>
			{/each}
		</ol>
		{#if !preview}
			<div class="tools">
				<Button variant="secondary" busy={writing} onclick={resourceNotes}
					>{t('Write its resources as notes')}</Button
				>
				<Button variant="secondary" busy={completing} onclick={complete}
					>{t('Add its goals, milestones, projects, habits, and areas')}</Button
				>
			</div>
		{/if}
	</section>

	<ol class="phases">
		{#each plan.phases as p (p.n)}
			<li id="{preview ? 'preview-' : ''}phase-{p.n}" class={p.state}>
				<details open={!preview && p.state === 'current'}>
					<summary>
						<span class="chip {p.state}">{stateLabel[p.state]}</span>
						<span class="title">{p.title}</span>
						<span class="dates muted">{short(p.start)} → {short(p.end)}</span>
						{#if p.milestone?.done}<span class="chip reached">{t('Milestone reached')}</span>{/if}
					</summary>
					<div class="body">
						<p class="muted small">{p.tracks}</p>
						<p class="small">{t('Domains')}: {p.domains.join(', ')}</p>

						<div class="grid">
							<section aria-label={t('Project')}>
								<h3>{t('Project')}</h3>
								{#if p.project && !p.project.id}
									<p><strong>{p.project.title}</strong></p>
									<p class="small">{p.project.outcome}</p>
								{:else if p.project}
									<p>
										<a href="/projects/{p.project.id}">{p.project.title}</a>
										<span class="muted small">· {p.project.status.replace('_', ' ')}</span>
									</p>
									<p class="small">{p.project.outcome}</p>
									{#if p.project.total}
										<div
											class="bar"
											role="progressbar"
											aria-label={t('{title}: cards done', { title: p.project.title })}
											aria-valuemin="0"
											aria-valuemax={p.project.total}
											aria-valuenow={p.project.done}
										>
											<span style:width="{(p.project.done / p.project.total) * 100}%"></span>
										</div>
										<p class="muted small">
											{t('{done} of {total} cards done', {
												done: p.project.done,
												total: p.project.total
											})}
										</p>
									{/if}
								{:else}
									<p class="muted small">{t('Its project was renamed or removed.')}</p>
								{/if}
							</section>

							<section aria-label={t('Certifications')}>
								<h3>{t('Certifications')}</h3>
								{#if p.certifications.length === 0}
									<p class="muted small">
										{t('None at the end of this phase: its domains are proven by the exit check.')}
									</p>
								{:else}
									<ul class="certs">
										{#each p.certifications as c (c.code + c.name)}
											<li>
												<a href={c.url} rel="noopener noreferrer" target="_blank">{c.code}</a>
												<span class="small">{c.name}</span>
												{#if !preview && c.status}<span class="chip">{certStatus[c.status]}</span
													>{:else if !preview}<Button
														variant="secondary"
														onclick={() => planCert(p, c)}
														aria-label={t('Plan {code}', { code: c.code })}>{t('Plan it')}</Button
													>{/if}
											</li>
										{/each}
									</ul>
									<a class="small" href="/learn/certifications">{t('All certifications')}</a>
								{/if}
							</section>
						</div>

						<section aria-label={t('Exit check')}>
							<h3>{t('Exit check')}</h3>
							<ul class="exit">
								{#each p.exit as e (e.title)}
									<li>
										{e.title}{#if e.parts?.length}<span class="muted small">
												({e.parts.join(', ')})</span
											>{/if}
									</li>
								{/each}
							</ul>
						</section>

						{#if p.fortytwo.length}
							<section aria-label={t('42 projects')}>
								<h3>{t('42 projects')}</h3>
								<ul class="exit">
									{#each p.fortytwo as f (f.title)}
										<li>
											{f.title}
											<span class="muted small"
												>~{f.hours} h{f.optional ? `, ${t('optional')}` : ''}</span
											>
											{#if f.note}<span class="muted small">· {f.note}</span>{/if}
										</li>
									{/each}
								</ul>
							</section>
						{/if}

						{#if p.milestone}
							<label class="reach">
								<input type="checkbox" checked={p.milestone.done} onchange={() => toggle(p)} />
								{t('Milestone reached: the exit check is done')}
							</label>
						{/if}
					</div>
				</details>
			</li>
		{/each}
	</ol>
{/if}

<style>
	.tools {
		margin-top: var(--space-3);
	}
	.notice {
		border: 1px solid var(--warn);
		border-radius: var(--radius-lg);
		padding: var(--space-3) var(--space-4);
		margin-bottom: var(--space-4);
		background: var(--surface);
	}
	.notice p {
		margin: 0 0 var(--space-2);
	}
	.summary {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.lead {
		font-size: var(--text-xl);
		font-weight: 650;
		margin: 0 0 var(--space-1);
	}
	.strip {
		list-style: none;
		padding: 0;
		margin: var(--space-3) 0 0;
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(2rem, 1fr));
		gap: 3px;
	}
	.strip a {
		display: block;
		text-align: center;
		padding: var(--space-1) 0;
		border-radius: var(--radius);
		background: var(--accent-soft);
		font-size: var(--text-sm);
		text-decoration: none;
		color: var(--fg);
	}
	.behind {
		color: var(--bad);
		font-size: var(--text-sm);
	}
	.strip .past a {
		opacity: 0.7;
	}
	.strip .current a {
		background: var(--accent);
		color: #fff;
		font-weight: 650;
	}
	.strip .reached a {
		background: var(--ok);
		color: #fff;
	}
	.phases {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-4);
		display: grid;
		gap: var(--space-2);
	}
	.phases > li {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		scroll-margin-top: 5rem;
	}
	.phases > li.current {
		border-color: var(--accent);
	}
	summary {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: baseline;
		padding: var(--space-3) var(--space-4);
		cursor: pointer;
	}
	.title {
		font-weight: 650;
	}
	.dates {
		font-size: var(--text-sm);
	}
	.body {
		padding: 0 var(--space-4) var(--space-4);
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(16rem, 1fr));
		gap: var(--space-4);
	}
	h3 {
		font-size: var(--text-base);
		margin: var(--space-3) 0 var(--space-1);
	}
	.chip {
		font-size: 0.75rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		padding: 0 var(--space-2);
	}
	.chip.current {
		background: var(--accent);
		border-color: var(--accent);
		color: #fff;
	}
	.chip.reached {
		border-color: var(--ok);
		color: var(--ok);
	}
	.bar {
		height: 0.5rem;
		border-radius: 999px;
		background: var(--accent-soft);
		overflow: hidden;
		max-width: 20rem;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--ok);
	}
	.certs,
	.exit {
		margin: 0;
		padding-left: var(--space-4);
		display: grid;
		gap: var(--space-1);
	}
	.certs li {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: baseline;
	}
	.reach {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		margin-top: var(--space-3);
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
