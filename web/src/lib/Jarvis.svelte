<script lang="ts">
	import { onDestroy, tick } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import { t } from '$lib/i18n.svelte';
	import JarvisFace from '$lib/JarvisFace.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Icon from '$lib/ui/Icon.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Jarvis (EP-153): a floating button opens a side panel where the master
	// agent works through Axon's other agents. The conversation is a run that
	// continues; what the agents want to change waits here for approval.
	type Listing = paths['/v1/agents']['get']['responses'][200]['content']['application/json'];
	type Run = Schemas['AgentRun'];
	type Proposal = Schemas['Proposal'];
	type Detail = { run: Run; reply: string | null; proposals: Proposal[]; can_continue?: boolean };
	type Turn = { who: 'you' | 'jarvis'; text: string; proposals: Proposal[] };

	const key = 'axon-jarvis-open';
	let open = $state(false);
	try {
		open = localStorage.getItem(key) === '1';
	} catch {
		// private browsing: closed by default
	}
	let listing = $state<Listing | null>(null);
	let turns = $state<Turn[]>([]);
	let text = $state('');
	let last = $state<Detail | null>(null);
	let thinking = $state(false);
	let log = $state<HTMLElement>();
	let field = $state<HTMLTextAreaElement>();
	let timer: ReturnType<typeof setTimeout> | undefined;
	onDestroy(() => clearTimeout(timer));

	const reasons: Record<string, string> = $derived({
		demo: t('Agents do not run in the shared demo.'),
		not_configured: t('No AI provider is set up yet.'),
		no_consent: t('Agents are off until you turn them on.'),
		budget_exhausted: t("This month's agent budget is used up.")
	});

	async function toggle() {
		open = !open;
		try {
			localStorage.setItem(key, open ? '1' : '0');
		} catch {
			// not remembered
		}
		if (open) {
			if (!listing) listing = await api<Listing>('/v1/agents').catch(() => null);
			await tick();
			field?.focus();
		}
	}

	async function scroll() {
		await tick();
		log?.scrollTo({ top: log.scrollHeight });
	}

	async function poll(id: string): Promise<Detail> {
		const d = await api<Detail>(`/v1/agent-runs/${id}`);
		if (d.run.status !== 'running') return d;
		await new Promise((r) => (timer = setTimeout(r, 1000)));
		return poll(id);
	}

	async function send(e?: SubmitEvent) {
		e?.preventDefault();
		const said = text.trim();
		if (!said || thinking) return;
		text = '';
		turns = [...turns, { who: 'you', text: said, proposals: [] }];
		thinking = true;
		scroll();
		try {
			// A conversation continues the last run while it can.
			const run =
				last?.can_continue && last.run.status === 'completed'
					? await api<Run>(`/v1/agent-runs/${last.run.id}/continue`, {
							method: 'POST',
							body: { instruction: said.slice(0, 4000) }
						})
					: await api<Run>('/v1/agents/jarvis/runs', {
							method: 'POST',
							body: { instruction: said.slice(0, 4000) }
						});
			last = await poll(run.id);
			const answer =
				last.reply ??
				(last.run.status === 'budget_exhausted'
					? t("This month's agent budget is used up.")
					: t('Jarvis could not finish ({status}).', { status: last.run.status }));
			turns = [...turns, { who: 'jarvis', text: answer, proposals: last.proposals }];
		} catch (err) {
			turns = [...turns, { who: 'jarvis', text: message(err), proposals: [] }];
		} finally {
			thinking = false;
			scroll();
		}
	}

	async function decide(turn: Turn, p: Proposal, action: 'approve' | 'reject') {
		try {
			await api(`/v1/proposals/${p.id}/${action}`, { method: 'POST' });
			toast(action === 'approve' ? t('Approved and applied.') : t('Rejected. Nothing changed.'));
			const status: Proposal['status'] = action === 'approve' ? 'approved' : 'rejected';
			turns = turns.map((x) =>
				x === turn
					? { ...x, proposals: x.proposals.map((q) => (q.id === p.id ? { ...q, status } : q)) }
					: x
			);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function onkey(e: KeyboardEvent) {
		if (e.key === 'Escape' && open) toggle();
	}
	function enter(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			send();
		}
	}
</script>

<svelte:window onkeydown={onkey} />

<button
	type="button"
	class="fab"
	class:open
	aria-expanded={open}
	aria-controls="jarvis-panel"
	aria-label={open ? t('Close Jarvis') : t('Open Jarvis')}
	title="Jarvis"
	onclick={toggle}><Icon name="bolt" size={24} /></button
>

{#if open}
	<aside id="jarvis-panel" class="panel" aria-label="Jarvis">
		<div class="sky" aria-hidden="true"></div>
		<header>
			<div class="portrait"><JarvisFace {thinking} /></div>
			<div>
				<h2>Jarvis</h2>
				<p class="tag">
					{t('Your master agent: it asks Axon’s agents for you.')}
					{#if listing?.status.provider}<br /><span class="dim"
							>{t('On {provider}, with your Jarvis model.', {
								provider: listing.status.provider
							})}</span
						>{/if}
				</p>
			</div>
		</header>

		{#if listing && !listing.status.available}
			<p class="notice">
				{reasons[listing.status.reason ?? ''] ?? ''}
				<a href="/account/preferences#ai">{t('AI settings')}</a> ·
				<a href="/account/privacy">{t('Privacy')}</a>
			</p>
		{/if}

		<div class="log" bind:this={log} aria-live="polite">
			{#if turns.length === 0}
				<p class="hint">
					{t(
						'Say it in your own words: “I’m reading The Martian, add it to my books”, “Plan my week around the roadmap”, “What should I review today?”. Every change waits for your approval here.'
					)}
				</p>
			{/if}
			{#each turns as turn, i (i)}
				<div class="turn {turn.who}">
					<p>{turn.text}</p>
					{#each turn.proposals as p (p.id)}
						<div class="proposal">
							<p>{p.summary}</p>
							{#if p.status === 'pending'}
								<div class="row">
									<Button onclick={() => decide(turn, p, 'approve')}>{t('Approve')}</Button>
									<Button variant="ghost" onclick={() => decide(turn, p, 'reject')}
										>{t('Reject')}</Button
									>
								</div>
							{:else}
								<p class="dim">{p.status === 'approved' ? t('Approved') : t('Rejected')}</p>
							{/if}
						</div>
					{/each}
				</div>
			{/each}
			{#if thinking}<p class="turn jarvis dim">{t('Jarvis is asking the agents…')}</p>{/if}
		</div>

		<form class="ask" onsubmit={send}>
			<label for="jarvis-text" class="visually-hidden">{t('Ask Jarvis')}</label>
			<textarea
				id="jarvis-text"
				bind:this={field}
				bind:value={text}
				rows="2"
				maxlength="4000"
				placeholder={t('Ask Jarvis…')}
				onkeydown={enter}
			></textarea>
			<Button type="submit" busy={thinking} disabled={!text.trim()}>{t('Send')}</Button>
		</form>
	</aside>
{/if}

<style>
	.fab {
		position: fixed;
		right: var(--space-4);
		bottom: var(--space-4);
		z-index: 60;
		width: 3.5rem;
		height: 3.5rem;
		border-radius: 50%;
		border: 1px solid rgb(62 242 255 / 0.6);
		background: radial-gradient(circle at 30% 30%, #2b1d5c, #070a1a);
		color: #3ef2ff;
		box-shadow:
			0 0 18px rgb(62 242 255 / 0.45),
			0 0 40px rgb(255 62 200 / 0.25);
		cursor: pointer;
		display: grid;
		place-items: center;
	}
	.fab:hover,
	.fab.open {
		color: #ff7ad9;
		border-color: rgb(255 62 200 / 0.8);
	}
	/* Open, the button sits beside the panel, or over its header on a phone. */
	.fab.open {
		right: calc(min(27rem, 100vw) + var(--space-3));
	}
	@media (max-width: 34rem) {
		.fab.open {
			right: var(--space-3);
			top: var(--space-3);
			bottom: auto;
			width: 2.75rem;
			height: 2.75rem;
		}
		header {
			padding-right: 3.75rem;
		}
	}
	.panel {
		position: fixed;
		top: 0;
		right: 0;
		bottom: 0;
		z-index: 55;
		width: min(27rem, 100vw);
		display: flex;
		flex-direction: column;
		color: #e6f7ff;
		background: #070a1a;
		border-left: 1px solid rgb(62 242 255 / 0.35);
		box-shadow: -12px 0 40px rgb(0 0 0 / 0.45);
		overflow: hidden;
	}
	/* A city at night under a neon horizon, and the grid of the floor. */
	.sky {
		position: absolute;
		inset: 0;
		z-index: 0;
		pointer-events: none;
		background:
			linear-gradient(180deg, rgb(7 10 26 / 0) 55%, rgb(7 10 26 / 0.9) 75%),
			radial-gradient(ellipse 80% 30% at 50% 58%, rgb(255 62 200 / 0.35), transparent 70%),
			linear-gradient(180deg, #0b0f2e 0%, #1b0f3a 45%, #2a0f45 58%, #070a1a 60%);
	}
	.sky::after {
		content: '';
		position: absolute;
		left: -50%;
		right: -50%;
		bottom: 0;
		height: 42%;
		background:
			repeating-linear-gradient(90deg, rgb(62 242 255 / 0.32) 0 1px, transparent 1px 48px),
			repeating-linear-gradient(0deg, rgb(255 62 200 / 0.3) 0 1px, transparent 1px 32px);
		transform: perspective(320px) rotateX(60deg);
		transform-origin: bottom;
		animation: floor 6s linear infinite;
	}
	.sky::before {
		content: '';
		position: absolute;
		inset: 0;
		background: repeating-linear-gradient(0deg, rgb(255 255 255 / 0.03) 0 1px, transparent 1px 3px);
	}
	@keyframes floor {
		to {
			background-position:
				0 0,
				0 32px;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.sky::after {
			animation: none;
		}
	}
	header,
	.notice,
	.log,
	.ask {
		position: relative;
		z-index: 1;
	}
	header {
		display: flex;
		gap: var(--space-3);
		align-items: center;
		padding: var(--space-4) var(--space-4) var(--space-2);
	}
	.portrait {
		width: 5.5rem;
		height: 6rem;
		flex: none;
	}
	h2 {
		margin: 0;
		font-size: 1.4rem;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: #3ef2ff;
		text-shadow: 0 0 10px rgb(62 242 255 / 0.7);
	}
	.tag {
		margin: var(--space-1) 0 0;
		font-size: var(--text-sm);
	}
	.dim {
		color: #9fb4c8;
		font-size: var(--text-sm);
	}
	.notice {
		margin: 0 var(--space-4);
		padding: var(--space-2) var(--space-3);
		border: 1px solid rgb(255 62 200 / 0.5);
		border-radius: var(--radius);
		background: rgb(255 62 200 / 0.1);
		font-size: var(--text-sm);
	}
	.notice a,
	.panel a {
		color: #7af6ff;
	}
	.log {
		flex: 1;
		overflow-y: auto;
		padding: var(--space-3) var(--space-4);
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}
	.hint {
		color: #b8c9d8;
		font-size: var(--text-sm);
	}
	.turn {
		max-width: 92%;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius-lg);
		backdrop-filter: blur(6px);
	}
	.turn p {
		margin: 0;
		white-space: pre-wrap;
	}
	.you {
		align-self: flex-end;
		background: rgb(255 62 200 / 0.18);
		border: 1px solid rgb(255 62 200 / 0.45);
	}
	.jarvis {
		align-self: flex-start;
		background: rgb(62 242 255 / 0.1);
		border: 1px solid rgb(62 242 255 / 0.4);
	}
	.proposal {
		margin-top: var(--space-2);
		padding: var(--space-2);
		border-radius: var(--radius);
		background: rgb(7 10 26 / 0.6);
		border: 1px dashed rgb(62 242 255 / 0.5);
		font-size: var(--text-sm);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		margin-top: var(--space-2);
	}
	.ask {
		display: flex;
		gap: var(--space-2);
		align-items: flex-end;
		padding: var(--space-3) var(--space-4) var(--space-4);
		background: rgb(7 10 26 / 0.85);
		border-top: 1px solid rgb(62 242 255 / 0.25);
	}
	.ask textarea {
		flex: 1;
		resize: none;
		background: rgb(255 255 255 / 0.06);
		color: #e6f7ff;
		border: 1px solid rgb(62 242 255 / 0.4);
		border-radius: var(--radius);
	}
	.ask textarea::placeholder {
		color: #8aa0b4;
	}
	@media (max-width: 30rem) {
		.fab {
			bottom: calc(var(--space-4) + env(safe-area-inset-bottom));
		}
	}
</style>
