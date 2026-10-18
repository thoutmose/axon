<script lang="ts">
	import { api, message, type Schemas } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import ChannelSettings from '$lib/ChannelSettings.svelte';
	import { emojiIcon } from '$lib/icons';
	import Icon from '$lib/ui/Icon.svelte';
	import DigestTemplate from '$lib/DigestTemplate.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// The digest as it would go out now (EP-107): the same content the email
	// and the chat apps receive, with its picture and its line to carry.
	type Digest = Schemas['Digest'];
	type Kind = Digest['kind'];

	let kind = $state<Kind>('daily');
	let digest = $state<Digest | null>(null);
	let error = $state('');
	let customising = $state(false);

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
	// Where it goes (EP-133): email, this person's devices, and chat apps,
	// set up here rather than only in Preferences.
	let status = $state<{ email: boolean; devices: unknown[] } | null>(null);
	let email = $state('');
	api<{ email: boolean; devices: unknown[] }>('/v1/me/reminders')
		.then((r) => (status = r))
		.catch(() => {});
	api<{ reminders: { email: string } }>('/v1/me/settings')
		.then((r) => (email = r.reminders.email))
		.catch(() => {});
	async function saveEmail(e: SubmitEvent) {
		e.preventDefault();
		try {
			const doc = await api<{ reminders: Record<string, unknown> } & Record<string, unknown>>(
				'/v1/me/settings'
			);
			delete doc.ai;
			doc.reminders = { ...doc.reminders, email: email.trim(), enabled: true };
			await api('/v1/me/settings', { method: 'PUT', body: doc });
			toast(
				email.trim() ? t('The digest goes to {email}.', { email: email.trim() }) : t('No email.')
			);
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	const tabs: [Kind, string][] = [
		['daily', 'Today'],
		['weekly', 'This week'],
		['monthly', 'This month'],
		['news', 'The news']
	];
</script>

<svelte:head><title>{t('Digest')} · Axon</title></svelte:head>

{#snippet mark(emoji: string, size: number)}
	{@const icon = emojiIcon(emoji)}
	<span class="mark" aria-hidden="true"
		>{#if icon}<Icon name={icon} {size} />{:else}{emoji}{/if}</span
	>
{/snippet}

<DigestTemplate bind:open={customising} onsaved={() => load(kind)} />

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
			{#if digest.picture}
				<img
					src="/api/v1/me/digest/picture?kind={kind}&d={digest.date}"
					alt=""
					width="1200"
					height="360"
				/>
			{/if}
			<div class="body">
				<p class="eyebrow">{digest.title} · {digest.label}</p>
				<h1>{digest.greeting}</h1>
				{#if digest.summary}<p class="muted">{digest.summary}</p>{/if}

				{#if digest.stats?.length}
					<ul class="stats">
						{#each digest.stats as s (s.label)}
							<li>
								{@render mark(s.emoji, 18)}<strong>{s.value}</strong><span class="muted small"
									>{s.label}</span
								>
							</li>
						{/each}
					</ul>
				{/if}

				{#each digest.sections ?? [] as sec (sec.key)}
					<section>
						<h2>{@render mark(sec.emoji, 20)} {sec.title}</h2>
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

				{#if digest.quote.text}
					<blockquote>
						“{digest.quote.text}”
						<footer>— {digest.quote.author}</footer>
					</blockquote>
				{/if}
			</div>
		</article>
		<div class="row">
			<Button variant="secondary" onclick={send}>{t('Send it to me now')}</Button>
			<Button variant="secondary" onclick={() => (customising = true)}>{t('Customise')}</Button>
			<Button variant="ghost" href="/account/preferences#reminders">{t('Times')}</Button>
		</div>
		<p class="muted small">
			{t(
				'In a chat app with a bot, answer “done 2” when you finish the second item, “add call the bank tomorrow” to capture, or “help”.'
			)}
		</p>
	{/if}

	<section class="card where" aria-labelledby="where-title">
		<h2 id="where-title">{t('Where it goes')}</h2>
		<form class="email" onsubmit={saveEmail}>
			<label for="digest-email">{t('By email')}</label>
			{#if status?.email}
				<div class="row">
					<input
						id="digest-email"
						type="email"
						maxlength="254"
						autocomplete="email"
						placeholder={t('you@example.com')}
						bind:value={email}
					/>
					<Button type="submit" variant="secondary">{t('Save')}</Button>
				</div>
			{:else}
				<p class="muted small" id="digest-email">
					{t('Email needs the operator’s mail server (AXON_SMTP_URL).')}
				</p>
			{/if}
		</form>
		<p class="small">
			{t('On {n} device(s) as notifications.', { n: status?.devices.length ?? 0 })}
			<a href="/account/preferences#reminders">{t('Turn them on')}</a>
		</p>
		<ChannelSettings />
	</section>
</div>

<style>
	.mark {
		color: var(--accent);
		display: inline-flex;
		vertical-align: -0.1em;
	}
	.where {
		margin-top: var(--space-6);
		padding: var(--space-4);
		display: grid;
		gap: var(--space-2);
	}
	.where h2 {
		margin: 0;
	}
	.email .row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.email input {
		flex: 1 1 14rem;
	}
	.page {
		max-width: 42rem;
		margin: 0 auto;
	}
	.tabs {
		display: flex;
		flex-wrap: wrap;
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
