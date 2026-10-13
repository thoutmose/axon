<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { onDestroy } from 'svelte';
	import { api, message, type Schemas } from './api';
	import type { paths } from './api-types';
	import Button from './ui/Button.svelte';
	import { toast } from './ui/toast.svelte';

	// The agents whose skills fit a page, asked from that page (EP-53). What
	// the page is about goes along with the request, so "this note" or "my
	// inbox" needs no explaining; every change still waits for approval.

	type Listing = paths['/v1/agents']['get']['responses'][200]['content']['application/json'];
	type Run = Schemas['AgentRun'];
	type Proposal = Schemas['Proposal'];
	type RunDetail = {
		run: Run;
		reply: string | null;
		proposals: Proposal[];
		can_continue?: boolean;
	};

	let {
		agents,
		suggestions = {},
		context = '',
		onapplied
	}: {
		/** Agent names, most relevant first. */
		agents: string[];
		/** Ready-made requests per agent. */
		suggestions?: Record<string, string[]>;
		/** What the page shows, sent with the request (ids included, so tools can use them). */
		context?: string;
		/** Called after a proposal is approved, so the page can reload. */
		onapplied?: () => void;
	} = $props();

	let open = $state(false);
	let listing = $state<Listing | null>(null);
	let chosen = $state('');
	let instruction = $state('');
	let current = $state<RunDetail | null>(null);
	let answer = $state('');
	let timer: ReturnType<typeof setTimeout> | undefined;
	onDestroy(() => clearTimeout(timer));

	const reasons: Record<string, string> = $derived({
		demo: t('Agents do not run in the shared demo.'),
		not_configured: t('No AI provider is set up yet.'),
		no_consent: t('Agents are off until you turn them on.'),
		budget_exhausted: t("This month's agent budget is used up.")
	});
	const available = $derived(
		(listing?.agents ?? [])
			.filter((a) => agents.includes(a.name))
			.sort((a, b) => agents.indexOf(a.name) - agents.indexOf(b.name))
	);
	const agent = $derived(available.find((a) => a.name === chosen) ?? available[0]);

	async function toggle() {
		open = !open;
		if (open && !listing) {
			try {
				listing = await api<Listing>('/v1/agents');
				chosen = agents.find((n) => listing!.agents.some((a) => a.name === n)) ?? '';
			} catch (err) {
				toast(message(err), 'error');
				open = false;
			}
		}
	}

	/** Opens the dock on an agent and sends a request composed by a form on the page. */
	export async function start(name: string, text: string) {
		if (!open) await toggle();
		chosen = name;
		await ask(undefined, text);
	}

	async function poll(id: string) {
		current = await api<RunDetail>(`/v1/agent-runs/${id}`);
		if (current.run.status === 'running') timer = setTimeout(() => poll(id), 1000);
	}

	async function ask(e?: SubmitEvent, text = instruction) {
		e?.preventDefault();
		if (!agent || !text.trim()) return;
		try {
			const body = context ? `${text}\n\n(Context from the page: ${context})` : text;
			const run = await api<Run>(`/v1/agents/${agent.name}/runs`, {
				method: 'POST',
				body: { instruction: body.slice(0, 4000) }
			});
			instruction = text;
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
			if (action === 'approve') onapplied?.();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<section class="dock" aria-label={t('Agents for this page')}>
	<button type="button" class="toggle" aria-expanded={open} onclick={toggle}>
		{'✨ ' + (open ? t('Close') : agents.length > 1 ? t('Ask an agent') : t('Ask the agent'))}
	</button>
	{#if open}
		<div class="panel glass">
			{#if !listing}
				<p class="muted small">{t('Loading…')}</p>
			{:else if !available.length}
				<p class="muted small">{t('No agent for this page.')}</p>
			{:else}
				{#if available.length > 1}
					<div class="choose" role="radiogroup" aria-label={t('Agent')}>
						{#each available as a (a.name)}
							<label class:on={agent?.name === a.name}>
								<input type="radio" name="dock-agent" value={a.name} bind:group={chosen} />
								{a.title}
							</label>
						{/each}
					</div>
				{/if}
				{#if agent}
					<p class="muted small">{agent.description}</p>
					{#if !listing.status.available && listing.status.reason}
						<p class="notice small">
							{reasons[listing.status.reason]}
							<a href="/agents">{t('Set up agents')}</a>
						</p>
					{:else}
						{#if (suggestions[agent.name] ?? []).length}
							<ul class="chips" aria-label={t('Suggestions')}>
								{#each suggestions[agent.name] as s (s)}
									<li>
										<button
											type="button"
											onclick={() => ask(undefined, s)}
											disabled={current?.run.status === 'running'}>{s}</button
										>
									</li>
								{/each}
							</ul>
						{/if}
						<form onsubmit={ask}>
							<label for="dock-ask" class="visually-hidden"
								>{t('Ask {title}', { title: agent.title })}</label
							>
							<textarea
								id="dock-ask"
								rows="2"
								maxlength="3000"
								placeholder={t('Ask {title}…', { title: agent.title })}
								bind:value={instruction}
							></textarea>
							<Button type="submit" variant="secondary" disabled={current?.run.status === 'running'}
								>{t('Ask {title}', { title: agent.title })}</Button
							>
						</form>
					{/if}
				{/if}
				{#if current}
					<div class="run" aria-live="polite">
						{#if current.run.status === 'running'}
							<p class="muted small">{t('Working…')}</p>
						{:else if current.reply}
							<p class="reply">{current.reply}</p>
						{/if}
						{#each current.proposals.filter((p) => p.status === 'pending') as p (p.id)}
							<div class="proposal">
								<div>
									<strong>{p.summary}</strong>
									{#if p.reason}<span class="muted small">{p.reason}</span>{/if}
								</div>
								<div class="row">
									<Button
										variant="secondary"
										onclick={() => decide(p, 'reject')}
										aria-label={t('Reject: {summary}', { summary: p.summary })}
										>{t('Reject')}</Button
									>
									<Button
										onclick={() => decide(p, 'approve')}
										aria-label={t('Approve: {summary}', { summary: p.summary })}
										>{t('Approve')}</Button
									>
								</div>
							</div>
						{/each}
						{#if current.can_continue}
							<form onsubmit={reply}>
								<label for="dock-answer" class="visually-hidden">{t('Answer the agent')}</label>
								<textarea id="dock-answer" rows="2" maxlength="4000" required bind:value={answer}
								></textarea>
								<Button type="submit" variant="secondary">{t('Send')}</Button>
							</form>
						{/if}
						<p class="muted small"><a href="/agents">{t('All agents, history, and budget')}</a></p>
					</div>
				{/if}
			{/if}
		</div>
	{/if}
</section>

<style>
	.dock {
		margin: var(--space-4) 0;
	}
	.toggle {
		font: inherit;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--fg);
		padding: var(--space-1) var(--space-3);
		cursor: pointer;
	}
	.toggle:hover {
		background: var(--accent-soft);
	}
	.panel {
		margin-top: var(--space-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--surface);
		padding: var(--space-4);
		display: grid;
		gap: var(--space-2);
	}
	.small {
		font-size: var(--text-sm);
		margin: 0;
	}
	.choose {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.choose label {
		border: 1px solid var(--border);
		border-radius: 999px;
		padding: 0 var(--space-3);
		cursor: pointer;
	}
	.choose label.on {
		border-color: var(--accent);
		background: var(--accent-soft);
	}
	.choose input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}
	.choose label:focus-within {
		outline: 2px solid var(--accent);
	}
	.chips {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		gap: var(--space-1);
		flex-wrap: wrap;
	}
	.chips button {
		font: inherit;
		font-size: var(--text-sm);
		border: 1px dashed var(--border);
		border-radius: var(--radius-sm);
		background: none;
		color: var(--fg);
		padding: var(--space-1) var(--space-2);
		cursor: pointer;
		text-align: left;
	}
	form {
		display: grid;
		gap: var(--space-2);
		justify-items: start;
	}
	textarea {
		width: 100%;
		font: inherit;
	}
	.reply {
		white-space: pre-wrap;
	}
	.proposal {
		display: flex;
		justify-content: space-between;
		gap: var(--space-2);
		flex-wrap: wrap;
		border-top: 1px solid var(--border);
		padding-top: var(--space-2);
	}
	.proposal > div:first-child {
		display: grid;
	}
	.row {
		display: flex;
		gap: var(--space-2);
	}
	.notice {
		padding: var(--space-2);
		border-radius: var(--radius-sm);
		background: var(--accent-soft);
	}
</style>
