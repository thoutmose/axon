<script lang="ts">
	import { api, message, type Schemas } from './api';
	import { t } from './i18n.svelte';
	import Button from './ui/Button.svelte';
	import { toast } from './ui/toast.svelte';

	// Chat apps that receive the digests and reminders (EP-70); bots also
	// read the answers once linked to the person's chat (EP-71).
	type Channel = Schemas['Channel'];
	type Kind = Channel['kind'];

	let channels = $state<Channel[]>([]);
	let adding = $state(false);
	let kind = $state<Kind>('slack');
	let name = $state('');
	let mode = $state<'webhook' | 'bot'>('webhook');
	let config = $state<Record<string, string>>({});
	let digests = $state<Record<string, boolean>>({ daily: true, weekly: true, monthly: true });
	let reminders = $state(true);
	let linkCode = $state<{ id: string; code: string } | null>(null);
	let busy = $state(false);

	const kinds: { id: Kind; label: string; bot: boolean; webhook: boolean }[] = [
		{ id: 'slack', label: 'Slack', bot: true, webhook: true },
		{ id: 'discord', label: 'Discord', bot: true, webhook: true },
		{ id: 'teams', label: 'Microsoft Teams', bot: false, webhook: true },
		{ id: 'telegram', label: 'Telegram', bot: true, webhook: false },
		{ id: 'ntfy', label: 'ntfy', bot: false, webhook: true },
		{ id: 'webhook', label: 'Webhook', bot: false, webhook: true }
	];
	const help: Record<string, string> = {
		'slack-webhook':
			'In Slack: Apps → Incoming Webhooks → Add to a channel, then paste the address (https://hooks.slack.com/services/…).',
		'slack-bot':
			'Create a Slack app with Socket Mode on, the chat:write and im:history scopes, and the message.im event. Paste its bot token (xoxb-) and an app-level token (xapp-) with connections:write. You can then answer the digest in the app’s messages.',
		'discord-webhook':
			'In Discord: a channel’s settings → Integrations → Webhooks → New webhook → Copy webhook URL.',
		'discord-bot':
			'Create an application in the Discord Developer Portal, add a bot, copy its token, and invite it to a server you share. It writes to you in direct messages, and you can answer there.',
		'teams-webhook':
			'In Teams: the channel’s ••• → Workflows → “Post to a channel when a webhook request is received”, then paste the address it gives.',
		'telegram-bot':
			'Talk to @BotFather: /newbot, then paste the token it gives. You can answer the digest in the chat.',
		'ntfy-webhook':
			'Install the ntfy app and subscribe to a topic of your own, hard to guess. Use your own server if you have one.',
		'webhook-webhook':
			'Axon posts JSON to this address. With a secret, each body is signed in X-Axon-Signature (sha256=HMAC).'
	};
	let current = $derived(kinds.find((k) => k.id === kind)!);
	$effect(() => {
		if (!current.webhook) mode = 'bot';
		else if (!current.bot) mode = 'webhook';
	});

	async function load() {
		try {
			channels = (await api<{ items: Channel[] }>('/v1/me/channels')).items;
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	load();

	function fields(): [string, string, string][] {
		if (mode === 'bot') {
			return kind === 'slack'
				? [
						['bot_token', 'Bot token', 'xoxb-…'],
						['app_token', 'App-level token', 'xapp-…']
					]
				: [['bot_token', 'Bot token', kind === 'telegram' ? '123456789:AA…' : '']];
		}
		switch (kind) {
			case 'ntfy':
				return [
					['server', 'Server', 'https://ntfy.sh'],
					['topic', 'Topic', 'my-axon-7f3k'],
					['token', 'Access token (optional)', '']
				];
			case 'webhook':
				return [
					['webhook_url', 'Address', 'https://…'],
					['secret', 'Secret (optional)', '']
				];
			default:
				return [['webhook_url', 'Webhook address', 'https://…']];
		}
	}

	async function add() {
		busy = true;
		try {
			const res = await api<{ channel: Channel; link_code?: string }>('/v1/me/channels', {
				method: 'POST',
				body: {
					kind,
					name: name || current.label,
					config: Object.fromEntries(Object.entries(config).filter(([, v]) => v)),
					digests: Object.keys(digests).filter((k) => digests[k]),
					reminders
				}
			});
			if (res.link_code) linkCode = { id: res.channel.id, code: res.link_code };
			adding = false;
			config = {};
			name = '';
			await load();
			toast(t('Channel connected.'));
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function update(c: Channel, change: Partial<Channel>) {
		try {
			await api(`/v1/me/channels/${c.id}`, {
				method: 'PUT',
				body: {
					name: c.name,
					digests: change.digests ?? c.digests,
					reminders: change.reminders ?? c.reminders,
					enabled: change.enabled ?? c.enabled
				}
			});
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function test(c: Channel) {
		try {
			await api(`/v1/me/channels/${c.id}/test`, { method: 'POST' });
			toast(t('Sent to {name}.', { name: c.name }));
		} catch (err) {
			toast(message(err), 'error');
		}
		await load();
	}

	async function relink(c: Channel) {
		try {
			const res = await api<{ link_code: string }>(`/v1/me/channels/${c.id}/link`, {
				method: 'POST'
			});
			linkCode = { id: c.id, code: res.link_code };
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove(c: Channel) {
		if (!confirm(t('Disconnect {name}?', { name: c.name }))) return;
		try {
			await api(`/v1/me/channels/${c.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function toggleDigest(c: Channel, d: 'daily' | 'weekly' | 'monthly') {
		const next = c.digests.includes(d) ? c.digests.filter((x) => x !== d) : [...c.digests, d];
		update(c, { digests: next });
	}

	const linkHow = (c: Channel | undefined, code: string) =>
		c?.kind === 'telegram'
			? t('Open your bot in Telegram and send: /start {code}', { code })
			: t('Send this to the bot in a direct message: link {code}', { code });
	const labels: Record<string, string> = { daily: 'Daily', weekly: 'Weekly', monthly: 'Monthly' };
</script>

<h3 class="sub">{t('Channels')}</h3>
{#if channels.length}
	<ul class="channels">
		{#each channels as c (c.id)}
			<li>
				<div class="head">
					<strong>{c.name}</strong>
					<span class="muted small">{kinds.find((k) => k.id === c.kind)?.label} · {c.target}</span>
					{#if c.mode === 'bot'}
						<span class="pill" class:ok={c.linked}
							>{c.linked
								? t('answers {name}', { name: c.peer_name || t('you') })
								: t('not linked yet')}</span
						>
					{/if}
				</div>
				<div class="opts">
					<label class="check"
						><input
							type="checkbox"
							checked={c.enabled}
							onchange={() => update(c, { enabled: !c.enabled })}
						/>{t('On')}</label
					>
					{#each ['daily', 'weekly', 'monthly'] as const as d (d)}
						<label class="check"
							><input
								type="checkbox"
								checked={c.digests.includes(d)}
								onchange={() => toggleDigest(c, d)}
							/>{t(labels[d])}</label
						>
					{/each}
					<label class="check"
						><input
							type="checkbox"
							checked={c.reminders}
							onchange={() => update(c, { reminders: !c.reminders })}
						/>{t('Reminders')}</label
					>
				</div>
				{#if c.last_error}<p class="bad small">{c.last_error}</p>{/if}
				{#if linkCode?.id === c.id}
					<p class="code" role="status">{linkHow(c, linkCode.code)}</p>
				{/if}
				<div class="row">
					<Button variant="ghost" onclick={() => test(c)}>{t('Send a test')}</Button>
					{#if c.mode === 'bot'}
						<Button variant="ghost" onclick={() => relink(c)}
							>{c.linked ? t('Link another chat') : t('New code')}</Button
						>
					{/if}
					<Button variant="ghost" onclick={() => remove(c)}>{t('Disconnect')}</Button>
				</div>
			</li>
		{/each}
	</ul>
{:else}
	<p class="muted small">
		{t(
			'Receive the digests in Slack, Discord, Teams, Telegram, ntfy, or any web address. With a bot, answer “done 2” when you finish something.'
		)}
	</p>
{/if}

{#if adding}
	<div class="add">
		<div class="grid">
			<div class="field">
				<label for="ch-kind">{t('App')}</label>
				<select id="ch-kind" bind:value={kind}>
					{#each kinds as k (k.id)}<option value={k.id}>{k.label}</option>{/each}
				</select>
			</div>
			{#if current.bot && current.webhook}
				<div class="field">
					<label for="ch-mode">{t('How')}</label>
					<select id="ch-mode" bind:value={mode}>
						<option value="webhook">{t('Webhook: Axon posts to a channel')}</option>
						<option value="bot">{t('Bot: also reads your answers')}</option>
					</select>
				</div>
			{/if}
			<div class="field">
				<label for="ch-name">{t('Name')}</label>
				<input id="ch-name" bind:value={name} maxlength="100" placeholder={current.label} />
			</div>
		</div>
		<p class="muted small">{t(help[`${kind}-${mode}`] ?? '')}</p>
		<div class="grid">
			{#each fields() as [key, label, placeholder] (key)}
				<div class="field">
					<label for="ch-{key}">{t(label)}</label>
					<input
						id="ch-{key}"
						type={key.includes('token') || key === 'secret' ? 'password' : 'text'}
						autocomplete="off"
						bind:value={config[key]}
						{placeholder}
					/>
				</div>
			{/each}
		</div>
		<div class="opts">
			{#each ['daily', 'weekly', 'monthly'] as const as d (d)}
				<label class="check"
					><input type="checkbox" bind:checked={digests[d]} />{t(labels[d])}</label
				>
			{/each}
			<label class="check"
				><input type="checkbox" bind:checked={reminders} />{t(
					'Task reminders and the evening nudge'
				)}</label
			>
		</div>
		<div class="row">
			<Button variant="secondary" {busy} onclick={add}>{t('Connect')}</Button>
			<Button variant="ghost" onclick={() => (adding = false)}>{t('Cancel')}</Button>
		</div>
	</div>
{:else}
	<Button variant="secondary" onclick={() => (adding = true)}>{t('Add a channel')}</Button>
{/if}

<style>
	.channels {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-3);
	}
	.channels li,
	.add {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-3);
	}
	.add {
		margin-top: var(--space-3);
	}
	.head {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: baseline;
	}
	.pill {
		font-size: var(--text-sm);
		padding: 0 var(--space-2);
		border-radius: 999px;
		background: var(--accent-soft);
	}
	.pill.ok {
		color: var(--ok);
	}
	.opts,
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1) var(--space-3);
		margin-top: var(--space-2);
	}
	.check {
		display: flex;
		gap: var(--space-1);
		align-items: baseline;
		font-weight: 400;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 14rem), 1fr));
		gap: var(--space-3) var(--space-4);
	}
	.field {
		display: grid;
		gap: var(--space-1);
	}
	.field label {
		font-weight: 600;
	}
	.field input,
	.field select {
		width: 100%;
		min-width: 0;
		box-sizing: border-box;
	}
	.code {
		font-family: var(--font-mono);
		background: var(--accent-soft);
		padding: var(--space-2);
		border-radius: var(--radius);
	}
	.small {
		font-size: var(--text-sm);
	}
	.bad {
		color: var(--bad);
	}
	.sub {
		font-size: var(--text-base);
		margin: var(--space-3) 0 var(--space-1);
	}
</style>
