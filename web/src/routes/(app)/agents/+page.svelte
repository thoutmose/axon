<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { onDestroy, tick } from 'svelte';
	import { page } from '$app/state';
	import { api, message, type Schemas } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import ProviderSettings from '$lib/ProviderSettings.svelte';
	import RepoConnections from '$lib/RepoConnections.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Listing = paths['/v1/agents']['get']['responses'][200]['content']['application/json'];
	type Run = Schemas['AgentRun'];
	type Proposal = Schemas['Proposal'];
	type RunDetail = {
		run: Run;
		reply: string | null;
		proposals: Proposal[];
		can_continue?: boolean;
	};
	type Turn = { who: 'you' | 'agent'; text: string };
	type Memory = { id: string; agent: string; content: string; created_at: string };

	let listing = $state<Listing | null>(null);
	let pending = $state<Proposal[]>([]);
	let runs = $state<Run[]>([]);
	let memories = $state<Memory[]>([]);
	let error = $state('');
	let budget = $state('');
	let instructions = $state<Record<string, string>>({});
	let current = $state<RunDetail | null>(null);
	// The conversation lives only in this tab; the server keeps it for an hour.
	let thread = $state<Turn[]>([]);
	let answer = $state('');
	let timer: ReturnType<typeof setTimeout> | undefined;

	const reasons: Record<string, string> = $derived({
		demo: t(
			'Agents do not run in the shared demo, so nobody spends AI credit. The proposals below were prepared so you can try approving and rejecting.'
		),
		not_configured: t(
			'No AI provider is set up yet. Choose one under “AI provider” below; everything else in Axon works without agents.'
		),
		no_consent: t(
			'Agents are off. When on, an agent sends the tasks, goals, and notes it reads to Anthropic to draft suggestions. Nothing changes without your approval.'
		),
		budget_exhausted: t(
			"This month's agent budget is used up. Raise it below or wait until next month."
		)
	});
	const outcomes: Record<string, string> = $derived({
		running: t('Working…'),
		completed: t('Finished'),
		budget_exhausted: t('Stopped: the next step could have gone over your monthly budget'),
		step_limit: t('Stopped after 8 steps'),
		refused: t('The model declined this request'),
		failed: t('Failed')
	});

	const dollars = (micros: number) => `$${(micros / 1_000_000).toFixed(2)}`;

	async function load() {
		error = '';
		try {
			const [l, p, r, m] = await Promise.all([
				api<Listing>('/v1/agents'),
				api<{ items: Proposal[] }>('/v1/proposals?status=pending'),
				api<{ items: Run[] }>('/v1/agent-runs'),
				api<{ items: Memory[] }>('/v1/agent-memories')
			]);
			listing = l;
			pending = p.items;
			runs = r.items;
			memories = m.items;
			budget = (l.status.monthly_budget_cents / 100).toFixed(2);
		} catch (err) {
			error = message(err);
		}
	}
	// Other pages hand over a request: /agents?agent=teacher&ask=…
	const handover = page.url.searchParams.get('agent');
	if (handover) instructions[handover] = page.url.searchParams.get('ask') ?? '';
	load().then(async () => {
		if (!handover) return;
		await tick();
		const field = document.getElementById(`ask-${handover}`);
		field?.scrollIntoView({ block: 'center' });
		field?.focus();
	});
	onDestroy(() => clearTimeout(timer));

	async function poll(id: string) {
		current = await api<RunDetail>(`/v1/agent-runs/${id}`);
		if (current.run.status === 'running') {
			timer = setTimeout(() => poll(id), 1000);
		} else {
			if (current.reply) thread = [...thread, { who: 'agent', text: current.reply }];
			await load();
		}
	}

	async function start(name: string, e: SubmitEvent) {
		e.preventDefault();
		try {
			const instruction = instructions[name] ?? '';
			const run = await api<Run>(`/v1/agents/${name}/runs`, {
				method: 'POST',
				body: { instruction }
			});
			thread = [{ who: 'you', text: instruction }];
			current = { run, reply: null, proposals: [] };
			await poll(run.id);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function reply(e: SubmitEvent) {
		e.preventDefault();
		if (!current) return;
		try {
			const run = await api<Run>(`/v1/agent-runs/${current.run.id}/continue`, {
				method: 'POST',
				body: { instruction: answer }
			});
			thread = [...thread, { who: 'you', text: answer }];
			answer = '';
			current = { run, reply: null, proposals: [] };
			await poll(run.id);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function decide(p: Proposal, action: 'approve' | 'reject') {
		try {
			await api(`/v1/proposals/${p.id}/${action}`, { method: 'POST' });
			toast(action === 'approve' ? t('Approved and applied.') : t('Rejected. Nothing changed.'));
			if (current) current = await api<RunDetail>(`/v1/agent-runs/${current.run.id}`);
			await load();
		} catch (err) {
			toast(message(err), 'error');
			await load();
		}
	}

	async function saveBudget(e: SubmitEvent) {
		e.preventDefault();
		try {
			await api('/v1/me/agent-settings', {
				method: 'PUT',
				body: { monthly_budget_cents: Math.round(Number(budget) * 100) }
			});
			toast(t('Budget saved.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function forget(m: Memory) {
		try {
			await api(`/v1/agent-memories/${m.id}`, { method: 'DELETE' });
			toast(t('Forgotten.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>{t('Agents · Axon')}</title></svelte:head>

<h1>{t('Agents')}</h1>
<p class="muted hint">
	{t(
		'Agents read what you allow and draft changes. Every change waits for your approval, and Axon works fully with agents off.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !listing}
	<Status kind="loading" message={t('Loading agents…')} />
{:else}
	{@const st = listing.status}
	{#if !st.available && st.reason}
		<div class="notice" role="status">
			<p>{reasons[st.reason]}</p>
			{#if st.reason === 'no_consent'}
				<Button href="/account/privacy" variant="secondary">{t('Review AI consent')}</Button>
			{/if}
		</div>
	{/if}

	{#if pending.length}
		<section class="card approvals" aria-labelledby="queue-title">
			<h2 id="queue-title">
				{t('Waiting for your approval ({length})', { length: pending.length })}
			</h2>
			<ul class="rows">
				{#each pending as p (p.id)}
					<li>
						<div class="grow">
							<strong>{p.summary}</strong>
							{#if p.reason}<span class="muted small">{p.reason}</span>{/if}
							<span class="muted small">{t('Proposed by {agent}', { agent: p.agent })}</span>
						</div>
						<div class="row">
							<Button
								variant="secondary"
								onclick={() => decide(p, 'reject')}
								aria-label={t('Reject: {summary}', { summary: p.summary })}>{t('Reject')}</Button
							>
							<Button
								onclick={() => decide(p, 'approve')}
								aria-label={t('Approve: {summary}', { summary: p.summary })}>{t('Approve')}</Button
							>
						</div>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	{#if current}
		<section class="card" aria-labelledby="current-title" aria-live="polite">
			<h2 id="current-title">
				{t('Conversation with {agent} · {value}', {
					agent: current.run.agent,
					value: outcomes[current.run.status] ?? current.run.status
				})}
			</h2>
			<ol class="thread">
				{#each thread as turn, i (i)}
					<li class={turn.who}>
						<span class="visually-hidden">{turn.who === 'you' ? t('You') : t('Agent')}:</span>
						<p class="reply">{turn.text}</p>
					</li>
				{/each}
			</ol>
			{#if current.run.status !== 'running'}
				<p class="muted small">
					{t('{proposals} proposal(s) · {length} tool call(s) · cost {value}', {
						proposals: current.run.proposals,
						length: current.run.tools_called.length,
						value: dollars(current.run.cost_micros)
					})}
				</p>
			{/if}
			{#if current.can_continue}
				<form class="answer" onsubmit={reply}>
					<label for="answer">{t('Answer the agent')}</label>
					<textarea id="answer" rows="2" maxlength="4000" required bind:value={answer}></textarea>
					<Button type="submit" variant="secondary">{t('Send')}</Button>
				</form>
			{/if}
		</section>
	{/if}

	{#each listing.agents as a (a.name)}
		<section class="card" aria-labelledby="agent-{a.name}">
			<h2 id="agent-{a.name}">{a.title} <span class="muted small">v{a.version}</span></h2>
			<p class="muted">{a.description}</p>
			{#if a.consents?.length}
				<p class="small">
					{t('Also needs your consent: {value}.', { value: a.consents.join(', ') })}
					<a href="/account/privacy">{t('Review in Privacy')}</a>{t(
						', where each says what is shared.'
					)}
				</p>
			{/if}
			<details>
				<summary>{t('What it can do')}</summary>
				<ul class="tools">
					{#each a.tools as toolCall (toolCall.name)}
						<li>
							<span class="chip" class:write={toolCall.write}
								>{toolCall.write ? t('needs approval') : 'reads'}</span
							>
							{toolCall.description}
						</li>
					{/each}
				</ul>
			</details>
			<form onsubmit={(e) => start(a.name, e)}>
				<label for="ask-{a.name}">{t('What should it do?')}</label>
				<textarea
					id="ask-{a.name}"
					rows="2"
					maxlength="4000"
					required
					bind:value={instructions[a.name]}
					placeholder={(
						{
							goals: t('Suggest milestones for my running goal'),
							habits: t('Help me restart my reading habit'),
							'side-projects': t('What should I work on next in my repository?'),
							'code-review': t('Review the open pull request')
						} as Record<string, string>
					)[a.name] ?? t('Clarify my inbox into next actions')}
				></textarea>
				<Button
					type="submit"
					variant="secondary"
					disabled={!st.available || current?.run.status === 'running'}
					>{t('Ask {title}', { title: a.title })}</Button
				>
			</form>
		</section>
	{/each}

	<ProviderSettings onchange={load} />

	<RepoConnections />

	<section class="card" aria-labelledby="budget-title">
		<h2 id="budget-title">{t('Budget')}</h2>
		<p>
			{t('Spent')} <strong>{dollars(st.spent_micros)}</strong>
			{t('of ${value} this month. A run stops before any step that could go over the budget.', {
				value: (st.monthly_budget_cents / 100).toFixed(2)
			})}
		</p>
		<form class="row" onsubmit={saveBudget}>
			<label for="budget">{t('Monthly budget (US$)')}</label>
			<input id="budget" type="number" min="0" max="1000" step="0.5" bind:value={budget} />
			<Button type="submit" variant="secondary">{t('Save')}</Button>
		</form>
	</section>

	<section class="card" aria-labelledby="memory-title">
		<h2 id="memory-title">{t('What agents remember')}</h2>
		{#if memories.length === 0}
			<p class="muted">{t('Nothing. Agents can only remember what you approve.')}</p>
		{:else}
			<ul class="rows">
				{#each memories as m (m.id)}
					<li>
						<span class="grow">{m.content} <span class="muted small">({m.agent})</span></span>
						<Button
							variant="ghost"
							onclick={() => forget(m)}
							aria-label={t('Forget: {content}', { content: m.content })}>{t('Forget')}</Button
						>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section class="card" aria-labelledby="runs-title">
		<h2 id="runs-title">{t('Run history')}</h2>
		{#if runs.length === 0}
			<p class="muted">{t('No runs yet.')}</p>
		{:else}
			<div class="table-wrap">
				<table>
					<caption class="visually-hidden">{t('Recent agent runs (no content is stored)')}</caption>
					<thead>
						<tr
							><th>{t('When')}</th><th>{t('Agent')}</th><th>{t('Outcome')}</th><th>{t('Tools')}</th
							><th>{t('Cost')}</th></tr
						>
					</thead>
					<tbody>
						{#each runs as r (r.id)}
							<tr>
								<td>{new Date(r.started_at).toLocaleString(appLocale())}</td>
								<td>{r.agent} v{r.skill_version}</td>
								<td>{outcomes[r.status] ?? r.status}</td>
								<td>{r.tools_called.join(', ') || '—'}</td>
								<td>{dollars(r.cost_micros)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
{/if}

<style>
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.notice {
		border: 1px solid var(--border);
		background: var(--surface);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin: var(--space-4) 0 var(--space-6);
	}
	.notice p {
		margin-top: 0;
	}
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
	.card h2 {
		font-size: var(--text-lg);
	}
	.approvals {
		border-color: var(--accent);
	}
	.rows,
	.tools {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.rows li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
	}
	.rows li:last-child {
		border-bottom: none;
	}
	.tools li {
		padding: var(--space-1) 0;
	}
	.grow {
		flex: 1;
		min-width: 12rem;
		display: grid;
		gap: var(--space-1);
		overflow-wrap: anywhere;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		flex-wrap: wrap;
	}
	.chip {
		font-size: var(--text-sm);
		border-radius: 999px;
		padding: 0 var(--space-2);
		background: var(--accent-soft);
	}
	.chip.write {
		background: var(--bad-soft);
		color: var(--bad);
	}
	form {
		display: grid;
		gap: var(--space-2);
		margin-top: var(--space-3);
	}
	form.row {
		display: flex;
	}
	label {
		font-weight: 600;
	}
	textarea,
	input {
		font: inherit;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		min-height: 2.75rem;
	}
	input {
		width: 8rem;
	}
	.reply {
		white-space: pre-wrap;
		margin: 0;
	}
	.thread {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-3);
		display: grid;
		gap: var(--space-2);
	}
	.thread li {
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		max-width: 46rem;
	}
	.thread li.you {
		background: var(--accent-soft);
		justify-self: end;
	}
	.thread li.agent {
		border: 1px solid var(--border);
	}
	.answer {
		display: grid;
		gap: var(--space-2);
		justify-items: start;
	}
	.answer textarea {
		width: 100%;
	}
	summary {
		cursor: pointer;
		color: var(--accent);
	}
	.table-wrap {
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--text-sm);
	}
	th,
	td {
		text-align: left;
		padding: var(--space-2);
		border-bottom: 1px solid var(--border);
		vertical-align: top;
	}
</style>
