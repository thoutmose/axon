<script lang="ts">
	import { api, message, type Schemas } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// The digest as it would go out now (EP-70): the same content the email
	// and the chat apps receive, with its picture and its line to carry.
	type Digest = Schemas['Digest'];
	type Kind = Digest['kind'];

	let kind = $state<Kind>('daily');
	let digest = $state<Digest | null>(null);
	let error = $state('');

	async function load(k: Kind) {
		error = '';
		try {
			digest = await api<Digest>(`/v1/me/digest?kind=${k}`);
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		load(kind);
	});

	async function send() {
		try {
			const r = await api<{ devices: number; email: boolean; channels: number }>(
				`/v1/me/reminders/test?kind=${kind}`,
				{ method: 'POST' }
			);
			toast(
				t('Sent to {devices} device(s), {channels} channel(s){email}.', {
					devices: r.devices,
					channels: r.channels,
					email: r.email ? t(' and by email') : ''
				})
			);
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	const tabs: [Kind, string][] = [
		['daily', 'Today'],
		['weekly', 'This week'],
		['monthly', 'This month']
	];
</script>

<svelte:head><title>{t('Digest')} · Axon</title></svelte:head>

<div class="page">
	<div class="tabs" role="tablist" aria-label={t('Digest')}>
		{#each tabs as [k, label] (k)}
			<button
				type="button"
				role="tab"
				aria-selected={kind === k}
				class:active={kind === k}
				onclick={() => (kind = k)}>{t(label)}</button
			>
		{/each}
	</div>

	{#if error}
		<Status kind="error" message={error} retry={() => load(kind)} />
	{:else if !digest}
		<Status kind="loading" message="…" />
	{:else}
		<article class="card">
			<img
				src="/api/v1/me/digest/picture?kind={kind}&d={digest.date}"
				alt=""
				width="1200"
				height="360"
			/>
			<div class="body">
				<p class="eyebrow">{digest.title} · {digest.label}</p>
				<h1>{digest.greeting}</h1>
				{#if digest.summary}<p class="muted">{digest.summary}</p>{/if}

				{#if digest.stats?.length}
					<ul class="stats">
						{#each digest.stats as s (s.label)}
							<li>
								<span aria-hidden="true">{s.emoji}</span><strong>{s.value}</strong><span
									class="muted small">{s.label}</span
								>
							</li>
						{/each}
					</ul>
				{/if}

				{#each digest.sections ?? [] as sec (sec.key)}
					<section>
						<h2><span aria-hidden="true">{sec.emoji}</span> {sec.title}</h2>
						<ul class="items">
							{#each sec.items as it, i (it.id + i)}
								<li>
									{#if it.n}<span class="n" title={t('Answer with this number')}>{it.n}</span>{/if}
									<a href={it.url}>{it.text}</a>
									{#if it.detail}<span class="detail" class:late={it.late}>{it.detail}</span>{/if}
								</li>
							{/each}
							{#if sec.more}<li class="muted small">
									{t('and {n} more', { n: sec.more })}
								</li>{/if}
						</ul>
					</section>
				{:else}
					<p class="muted">{t('Nothing to do and nothing to report: enjoy the day.')}</p>
				{/each}

				<blockquote>
					“{digest.quote.text}”
					<footer>— {digest.quote.author}</footer>
				</blockquote>
			</div>
		</article>
		<div class="row">
			<Button variant="secondary" onclick={send}>{t('Send it to me now')}</Button>
			<Button variant="ghost" href="/account/preferences#reminders"
				>{t('Times and channels')}</Button
			>
		</div>
		<p class="muted small">
			{t(
				'In a chat app with a bot, answer “done 2” when you finish the second item, “add call the bank tomorrow” to capture, or “help”.'
			)}
		</p>
	{/if}
</div>

<style>
	.page {
		max-width: 42rem;
		margin: 0 auto;
	}
	.tabs {
		display: flex;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}
	.tabs button {
		font: inherit;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		border-radius: 999px;
		padding: var(--space-1) var(--space-3);
		cursor: pointer;
	}
	.tabs button.active {
		background: var(--accent);
		color: var(--accent-fg);
		border-color: var(--accent);
	}
	.card {
		background: var(--surface-solid);
		border: 1px solid var(--border);
		border-radius: var(--radius-xl);
		overflow: hidden;
		box-shadow: var(--shadow);
	}
	.card img {
		display: block;
		width: 100%;
		height: auto;
	}
	.body {
		padding: var(--space-6);
	}
	.eyebrow {
		text-transform: uppercase;
		letter-spacing: 0.06em;
		font-size: var(--text-sm);
		color: var(--muted);
		margin: 0;
	}
	h1 {
		font-size: var(--text-2xl);
		margin: var(--space-1) 0;
	}
	h2 {
		font-size: var(--text-lg);
		margin: var(--space-6) 0 var(--space-2);
	}
	.stats {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(7rem, 1fr));
		gap: var(--space-2);
		margin: var(--space-4) 0 0;
	}
	.stats li {
		display: grid;
		justify-items: center;
		background: var(--accent-soft);
		border-radius: var(--radius-lg);
		padding: var(--space-3);
	}
	.stats strong {
		font-size: var(--text-xl);
	}
	.items {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.items li {
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: baseline;
	}
	.items a {
		color: inherit;
		text-decoration: none;
	}
	.items a:hover {
		text-decoration: underline;
	}
	.n {
		min-width: 1.6em;
		text-align: center;
		border-radius: 999px;
		background: var(--accent);
		color: var(--accent-fg);
		font-size: var(--text-sm);
		font-weight: 700;
	}
	.detail {
		color: var(--muted);
		font-size: var(--text-sm);
	}
	.detail.late {
		color: var(--bad);
	}
	blockquote {
		margin: var(--space-6) 0 0;
		padding: var(--space-3) var(--space-4);
		border-left: 4px solid var(--accent);
		background: var(--accent-soft);
		border-radius: var(--radius);
		font-style: italic;
		font-size: var(--text-lg);
	}
	blockquote footer {
		font-style: normal;
		font-size: var(--text-sm);
		color: var(--muted);
		margin-top: var(--space-1);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		margin: var(--space-4) 0 var(--space-2);
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
