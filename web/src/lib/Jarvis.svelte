<script lang="ts">
	import { appLocale } from './dates';
	import { onDestroy, tick } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import { t } from '$lib/i18n.svelte';
	import JarvisBrain from '$lib/JarvisBrain.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Icon from '$lib/ui/Icon.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Jarvis (EP-153): a floating button opens a side panel where the master
	// agent works through Axon's other agents. The conversation is a run that
	// continues; what the agents want to change waits here for approval.
	// EP-172: a header with its controls, suggestions to start, named and
	// timed messages, proposals by agent with Approve all, a wider view, and
	// the conversation kept for the browser session.
	type Listing = paths['/v1/agents']['get']['responses'][200]['content']['application/json'];
	type Run = Schemas['AgentRun'];
	type Proposal = Schemas['Proposal'];
	type Detail = { run: Run; reply: string | null; proposals: Proposal[]; can_continue?: boolean };
	type Turn = { who: 'you' | 'jarvis'; text: string; proposals: Proposal[]; at: string };

	const key = 'axon-jarvis-open';
	const wideKey = 'axon-jarvis-wide';
	const talkKey = 'axon-jarvis-talk';
	const read = (k: string, store: () => Storage) => {
		try {
			return store().getItem(k);
		} catch {
			return null; // private browsing: nothing remembered
		}
	};
	const write = (k: string, v: string, store: () => Storage) => {
		try {
			store().setItem(k, v);
		} catch {
			// not remembered
		}
	};
	let open = $state(read(key, () => localStorage) === '1');
	let wide = $state(read(wideKey, () => localStorage) === '1');
	let listing = $state<Listing | null>(null);
	// The conversation, kept for this browser session.
	const saved: { turns: Turn[]; last: Detail | null } | null = (() => {
		try {
			return JSON.parse(read(talkKey, () => sessionStorage) ?? 'null');
		} catch {
			return null; // a conversation from an older version: start afresh
		}
	})();
	let turns = $state<Turn[]>(saved?.turns ?? []);
	let last = $state<Detail | null>(saved?.last ?? null);
	let text = $state('');
	let thinking = $state(false);
	let since = $state(0);
	let log = $state<HTMLElement>();
	let field = $state<HTMLTextAreaElement>();
	let timer: ReturnType<typeof setTimeout> | undefined;
	let clock: ReturnType<typeof setInterval> | undefined;
	onDestroy(() => {
		clearTimeout(timer);
		clearInterval(clock);
	});
	$effect(() => write(talkKey, JSON.stringify({ turns, last }), () => sessionStorage));
	// Open from a previous page, it still needs what the agents can do.
	$effect(() => {
		if (open) load();
	});

	const reasons: Record<string, string> = $derived({
		demo: t('Agents do not run in the shared demo.'),
		not_configured: t('No AI provider is set up yet.'),
		no_consent: t('Agents are off until you turn them on.'),
		budget_exhausted: t("This month's agent budget is used up.")
	});
	const suggestions = $derived([
		t('What should I do today?'),
		t('Plan my week around my roadmap'),
		t('What should I review today?'),
		t('How are my habits going?'),
		t('I’m reading The Martian, add it to my books'),
		t('Suggest a resource for my current phase')
	]);
	const agentTitle = (name: string) => listing?.agents.find((a) => a.name === name)?.title ?? name;
	const helpers = $derived(
		(listing?.agents ?? []).filter((a) => a.name !== 'jarvis').map((a) => a.title)
	);
	const available = $derived(!listing || listing.status.available);
	const pending = (turn: Turn) => turn.proposals.filter((p) => p.status === 'pending');
	const time = (iso: string) =>
		new Date(iso).toLocaleTimeString(appLocale(), { hour: '2-digit', minute: '2-digit' });

	async function load() {
		if (!listing) listing = await api<Listing>('/v1/agents').catch(() => null);
	}
	async function toggle() {
		open = !open;
		write(key, open ? '1' : '0', () => localStorage);
		if (open) {
			await load();
			await tick();
			field?.focus();
			scroll();
		}
	}
	function widen() {
		wide = !wide;
		write(wideKey, wide ? '1' : '0', () => localStorage);
	}
	function restart() {
		turns = [];
		last = null;
		field?.focus();
	}

	async function scroll() {
		await tick();
		log?.scrollTo({ top: log.scrollHeight, behavior: 'smooth' });
	}
	function grow() {
		if (!field) return;
		field.style.height = 'auto';
		field.style.height = Math.min(field.scrollHeight, 200) + 'px';
	}

	async function poll(id: string): Promise<Detail> {
		const d = await api<Detail>(`/v1/agent-runs/${id}`);
		if (d.run.status !== 'running') return d;
		await new Promise((r) => (timer = setTimeout(r, 1000)));
		return poll(id);
	}

	async function send(e?: SubmitEvent, said = text.trim()) {
		e?.preventDefault();
		if (!said || thinking) return;
		text = '';
		await tick();
		grow();
		turns = [...turns, { who: 'you', text: said, proposals: [], at: new Date().toISOString() }];
		thinking = true;
		since = 0;
		clock = setInterval(() => since++, 1000);
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
			turns = [
				...turns,
				{ who: 'jarvis', text: answer, proposals: last.proposals, at: new Date().toISOString() }
			];
		} catch (err) {
			turns = [
				...turns,
				{ who: 'jarvis', text: message(err), proposals: [], at: new Date().toISOString() }
			];
		} finally {
			thinking = false;
			clearInterval(clock);
			scroll();
		}
	}

	async function decide(turn: Turn, p: Proposal, action: 'approve' | 'reject', quiet = false) {
		try {
			await api(`/v1/proposals/${p.id}/${action}`, { method: 'POST' });
			if (!quiet)
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
	async function approveAll(turn: Turn) {
		const all = pending(turn);
		for (const p of all)
			await decide(turns.find((x) => x.at === turn.at) ?? turn, p, 'approve', true);
		toast(t('{n} changes approved and applied.', { n: all.length }));
	}

	function onkey(e: KeyboardEvent) {
		if (e.key === 'Escape' && open) toggle();
		// Ctrl+J (⌘J) opens and closes Jarvis from anywhere.
		if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'j') {
			e.preventDefault();
			toggle();
		}
	}
	function enter(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			send();
		}
	}
</script>

<svelte:window onkeydown={onkey} />

{#if !open}
	<button
		type="button"
		class="fab"
		aria-expanded="false"
		aria-controls="jarvis-panel"
		aria-label={t('Open Jarvis')}
		title={t('Jarvis (Ctrl+J)')}
		onclick={toggle}><Icon name="bolt" size={24} /></button
	>
{:else}
	<aside id="jarvis-panel" class="panel" class:wide aria-label="Jarvis">
		<div class="sky" aria-hidden="true"></div>
		<header>
			<div class="portrait"><JarvisBrain {thinking} /></div>
			<div class="who">
				<h2>Jarvis</h2>
				<p class="tag">
					<span class="dot" class:on={available} aria-hidden="true"></span>
					{#if !available}{t('Unavailable')}{:else if thinking}{t('Asking the agents…')}{:else}{t(
							'Ready'
						)}{/if}
					{#if listing?.status.provider}<span class="dim">
							· {t('on {provider}', { provider: listing.status.provider })}</span
						>{/if}
				</p>
			</div>
			<div class="tools">
				{#if turns.length}
					<button type="button" class="tool" onclick={restart} title={t('New conversation')}
						>＋<span class="visually-hidden">{t('New conversation')}</span></button
					>
				{/if}
				<button
					type="button"
					class="tool wide-toggle"
					onclick={widen}
					aria-pressed={wide}
					title={wide ? t('Narrower') : t('Wider')}
					>{wide ? '⇥' : '⇤'}<span class="visually-hidden">{wide ? t('Narrower') : t('Wider')}</span
					></button
				>
				<button
					type="button"
					class="tool"
					aria-expanded="true"
					aria-controls="jarvis-panel"
					aria-label={t('Close Jarvis')}
					title={t('Close (Esc)')}
					onclick={toggle}>✕</button
				>
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
				<div class="welcome">
					<div class="hero"><JarvisBrain /></div>
					<p class="lead">{t('What can I do for you?')}</p>
					<p class="hint">
						{t(
							'Say it in your own words. I ask the right agents, and every change waits for your approval here.'
						)}
					</p>
					<ul class="chips" aria-label={t('Suggestions')}>
						{#each suggestions as s (s)}
							<li>
								<button type="button" disabled={!available} onclick={() => send(undefined, s)}
									>{s}</button
								>
							</li>
						{/each}
					</ul>
					{#if helpers.length}
						<p class="dim helpers">
							{t('I work with: {agents}.', { agents: helpers.join(', ') })}
						</p>
					{/if}
				</div>
			{/if}
			{#each turns as turn, i (i)}
				<div class="turn {turn.who}">
					<p class="meta">
						<strong>{turn.who === 'you' ? t('You') : 'Jarvis'}</strong>
						{#if turn.at}<time datetime={turn.at}>{time(turn.at)}</time>{/if}
					</p>
					<p class="text">{turn.text}</p>
					{#if turn.proposals.length}
						<div class="proposals">
							{#if pending(turn).length > 1}
								<div class="row all">
									<span class="dim"
										>{t('{n} changes wait for you', { n: pending(turn).length })}</span
									>
									<Button onclick={() => approveAll(turn)}>{t('Approve all')}</Button>
								</div>
							{/if}
							{#each turn.proposals as p (p.id)}
								<div class="proposal {p.status}">
									<p class="agent">{agentTitle(p.agent)}</p>
									<p>{p.summary}</p>
									{#if p.status === 'pending'}
										<div class="row">
											<Button onclick={() => decide(turn, p, 'approve')}>{t('Approve')}</Button>
											<Button variant="ghost" onclick={() => decide(turn, p, 'reject')}
												>{t('Reject')}</Button
											>
										</div>
									{:else}
										<p class="dim">
											{p.status === 'approved' ? '✓ ' + t('Approved') : t('Rejected')}
										</p>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/each}
			{#if thinking}
				<p class="turn jarvis thinking">
					<span class="dots" aria-hidden="true"><i></i><i></i><i></i></span>
					{t('Jarvis is asking the agents…')}
					<span class="dim">{since} s</span>
				</p>
			{/if}
		</div>

		<form class="ask" onsubmit={send}>
			<label for="jarvis-text" class="visually-hidden">{t('Ask Jarvis')}</label>
			<textarea
				id="jarvis-text"
				bind:this={field}
				bind:value={text}
				rows="1"
				maxlength="4000"
				placeholder={t('Ask Jarvis…')}
				onkeydown={enter}
				oninput={grow}
			></textarea>
			<Button type="submit" busy={thinking} disabled={!text.trim() || !available}
				>{t('Send')}</Button
			>
			<p class="keys dim">{t('Enter to send · Shift+Enter for a new line · Esc to close')}</p>
		</form>
	</aside>
{/if}

<style>
	.fab {
		position: fixed;
		right: var(--space-4);
		bottom: calc(var(--space-4) + env(safe-area-inset-bottom));
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
		transition: transform 0.15s;
	}
	.fab:hover,
	.fab:focus-visible {
		color: #ff7ad9;
		border-color: rgb(255 62 200 / 0.8);
		transform: scale(1.06);
	}
	.panel {
		position: fixed;
		top: 0;
		right: 0;
		bottom: 0;
		z-index: 55;
		width: min(30rem, 100vw);
		display: flex;
		flex-direction: column;
		color: #e6f7ff;
		background: #070a1a;
		border-left: 1px solid rgb(62 242 255 / 0.35);
		box-shadow: -12px 0 40px rgb(0 0 0 / 0.45);
		overflow: hidden;
		animation: slide 0.2s ease-out;
	}
	.panel.wide {
		width: min(52rem, 100vw);
	}
	@keyframes slide {
		from {
			transform: translateX(2rem);
			opacity: 0;
		}
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
		.sky::after,
		.panel,
		.dots i {
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
		padding: var(--space-3) var(--space-3) var(--space-3) var(--space-4);
		border-bottom: 1px solid rgb(62 242 255 / 0.18);
		background: rgb(7 10 26 / 0.55);
		backdrop-filter: blur(8px);
	}
	.portrait {
		width: 5rem;
		height: 3.5rem;
		flex: none;
	}
	.who {
		flex: 1;
		min-width: 0;
	}
	h2 {
		margin: 0;
		font-size: 1.25rem;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: #3ef2ff;
		text-shadow: 0 0 10px rgb(62 242 255 / 0.7);
	}
	.tag {
		margin: 2px 0 0;
		font-size: var(--text-sm);
		display: flex;
		align-items: center;
		gap: 0.4rem;
		flex-wrap: wrap;
	}
	.dot {
		width: 0.55rem;
		height: 0.55rem;
		border-radius: 50%;
		background: #ff5c8a;
		box-shadow: 0 0 8px #ff5c8a;
	}
	.dot.on {
		background: #3effa8;
		box-shadow: 0 0 8px #3effa8;
	}
	.tools {
		display: flex;
		gap: var(--space-1);
		align-self: flex-start;
	}
	.tool {
		width: 2.25rem;
		height: 2.25rem;
		display: grid;
		place-items: center;
		border-radius: 50%;
		border: 1px solid rgb(62 242 255 / 0.35);
		background: rgb(255 255 255 / 0.04);
		color: #cfefff;
		cursor: pointer;
		font-size: 1rem;
	}
	.tool:hover,
	.tool:focus-visible {
		color: #ff7ad9;
		border-color: rgb(255 62 200 / 0.8);
	}
	@media (max-width: 40rem) {
		.wide-toggle {
			display: none;
		}
	}
	.dim {
		color: #9fb4c8;
		font-size: var(--text-sm);
	}
	.notice {
		margin: var(--space-3) var(--space-4) 0;
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
		padding: var(--space-4);
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		scrollbar-width: thin;
		scrollbar-color: rgb(62 242 255 / 0.4) transparent;
	}
	.welcome {
		margin: auto 0;
		padding: var(--space-4);
		border-radius: var(--radius-lg);
		background: rgb(7 10 26 / 0.55);
		border: 1px solid rgb(62 242 255 / 0.25);
		backdrop-filter: blur(6px);
	}
	.hero {
		width: min(16rem, 100%);
		aspect-ratio: 324 / 226;
		margin: 0 auto var(--space-3);
	}
	.lead {
		margin: 0;
		font-size: 1.2rem;
		font-weight: 650;
	}
	.hint {
		color: #b8c9d8;
		font-size: var(--text-sm);
		margin: var(--space-1) 0 var(--space-3);
	}
	.chips {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.chips button {
		font: inherit;
		font-size: var(--text-sm);
		text-align: left;
		padding: 0.45rem 0.8rem;
		border-radius: 999px;
		border: 1px solid rgb(62 242 255 / 0.45);
		background: rgb(62 242 255 / 0.08);
		color: #e6f7ff;
		cursor: pointer;
	}
	.chips button:hover:not(:disabled),
	.chips button:focus-visible {
		border-color: rgb(255 62 200 / 0.8);
		background: rgb(255 62 200 / 0.15);
	}
	.chips button:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}
	.helpers {
		margin: var(--space-3) 0 0;
	}
	.turn {
		max-width: 88%;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius-lg);
		backdrop-filter: blur(6px);
	}
	.wide .turn {
		max-width: 75%;
	}
	.turn p {
		margin: 0;
	}
	.meta {
		display: flex;
		gap: var(--space-2);
		align-items: baseline;
		font-size: var(--text-xs, 0.75rem);
		color: #9fb4c8;
		margin-bottom: 2px !important;
	}
	.meta strong {
		color: #cfefff;
		letter-spacing: 0.04em;
	}
	.text {
		white-space: pre-wrap;
		line-height: 1.5;
	}
	.you {
		align-self: flex-end;
		background: rgb(255 62 200 / 0.18);
		border: 1px solid rgb(255 62 200 / 0.45);
		border-bottom-right-radius: 4px;
	}
	.jarvis {
		align-self: flex-start;
		background: rgb(62 242 255 / 0.1);
		border: 1px solid rgb(62 242 255 / 0.4);
		border-bottom-left-radius: 4px;
	}
	.thinking {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		font-size: var(--text-sm);
	}
	.dots {
		display: inline-flex;
		gap: 3px;
	}
	.dots i {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: #3ef2ff;
		animation: blink 1.2s infinite ease-in-out;
	}
	.dots i:nth-child(2) {
		animation-delay: 0.2s;
	}
	.dots i:nth-child(3) {
		animation-delay: 0.4s;
	}
	@keyframes blink {
		0%,
		80%,
		100% {
			opacity: 0.25;
		}
		40% {
			opacity: 1;
		}
	}
	.proposals {
		margin-top: var(--space-2);
		display: grid;
		gap: var(--space-2);
	}
	.proposal {
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		background: rgb(7 10 26 / 0.65);
		border: 1px dashed rgb(62 242 255 / 0.55);
		font-size: var(--text-sm);
	}
	.proposal.approved {
		border-style: solid;
		border-color: rgb(62 255 168 / 0.6);
	}
	.proposal.rejected {
		opacity: 0.6;
	}
	.agent {
		font-size: var(--text-xs, 0.75rem);
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #7af6ff;
		margin-bottom: 2px !important;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		margin-top: var(--space-2);
		align-items: center;
	}
	.row.all {
		justify-content: space-between;
		margin-top: 0;
	}
	.ask {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: flex-end;
		padding: var(--space-3) var(--space-4) calc(var(--space-3) + env(safe-area-inset-bottom));
		background: rgb(7 10 26 / 0.9);
		border-top: 1px solid rgb(62 242 255 / 0.25);
	}
	.ask textarea {
		flex: 1;
		min-height: 2.75rem;
		max-height: 200px;
		resize: none;
		padding: 0.65rem 0.8rem;
		font: inherit;
		background: rgb(255 255 255 / 0.06);
		color: #e6f7ff;
		border: 1px solid rgb(62 242 255 / 0.4);
		border-radius: var(--radius);
	}
	.ask textarea:focus {
		outline: 2px solid rgb(62 242 255 / 0.7);
		outline-offset: 1px;
	}
	.ask textarea::placeholder {
		color: #8aa0b4;
	}
	.keys {
		flex-basis: 100%;
		margin: 0;
		font-size: var(--text-xs, 0.75rem);
	}
	@media (max-width: 40rem) {
		.keys {
			display: none;
		}
	}
</style>
