<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { startRegistration } from '@simplewebauthn/browser';
	import { api, message, type Schemas } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import List from '$lib/ui/List.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import RecoveryCodes from '$lib/RecoveryCodes.svelte';
	import SignInProviders from '$lib/SignInProviders.svelte';
	import { lowOnRecoveryCodes, recovery, requirePasskeys } from '$lib/recovery.svelte';
	import { page } from '$app/state';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let me = $state<Schemas['Me'] | null>(null);
	let passkeys = $state<Schemas['Passkey'][]>([]);
	let sessions = $state<Schemas['Session'][]>([]);
	type Identity = {
		id: string;
		provider: string;
		email: string;
		name: string;
		last_used_at: string | null;
	};
	let identities = $state<Identity[]>([]);
	const params = page.url.searchParams;
	const banner = params.get('welcome')
		? t(
				'Welcome to Axon. Add a passkey below: it is the safest way to sign in, and works if the other service is unavailable.'
			)
		: params.get('linked')
			? t('Linked: you can now sign in with it.')
			: params.get('error') === 'taken'
				? t('That account is already linked to another Axon account.')
				: '';
	let error = $state('');
	let busy = $state(false);
	let passkeyName = $state('');
	let codes = $state<string[]>([]);
	let totpSetup = $state<{ secret: string; otpauth_url: string } | null>(null);
	let totpCode = $state('');

	const when = (iso: string | null) =>
		iso ? new Date(iso).toLocaleString(appLocale()) : t('never');

	async function load() {
		error = '';
		try {
			[me, { items: passkeys }, { items: sessions }, { items: identities }] = await Promise.all([
				api<Schemas['Me']>('/v1/me'),
				api<{ items: Schemas['Passkey'][] }>('/v1/me/passkeys'),
				api<{ items: Schemas['Session'][] }>('/v1/me/sessions'),
				api<{ items: Identity[] }>('/v1/me/identities')
			]);
			recovery.remaining = me.recovery_codes_remaining;
		} catch (err) {
			error = message(err);
		}
	}

	async function run(action: () => Promise<unknown>, done?: string) {
		busy = true;
		try {
			await action();
			if (done) toast(done);
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	const addPasskey = (e: SubmitEvent) => {
		e.preventDefault();
		return run(async () => {
			requirePasskeys();
			const options = await api<{ publicKey: never }>('/v1/me/passkeys/begin', {
				method: 'POST',
				body: { name: passkeyName }
			});
			const response = await startRegistration({ optionsJSON: options.publicKey });
			await api('/v1/me/passkeys/finish', { method: 'POST', body: response });
			passkeyName = '';
		}, t('Passkey added.'));
	};

	const removePasskey = (p: Schemas['Passkey']) =>
		run(
			() => api(`/v1/me/passkeys/${p.id}`, { method: 'DELETE' }),
			t('Removed “{name}”.', { name: p.name })
		);

	const newCodes = () =>
		run(async () => {
			codes = (await api<{ recovery_codes: string[] }>('/v1/me/recovery-codes', { method: 'POST' }))
				.recovery_codes;
		}, t('New recovery codes issued. The old ones no longer work.'));

	const totpBegin = () =>
		run(async () => {
			totpSetup = await api('/v1/me/totp/begin', { method: 'POST' });
		});

	const totpSubmit = (e: SubmitEvent, on: boolean) => {
		e.preventDefault();
		return run(
			async () => {
				await api(on ? '/v1/me/totp/confirm' : '/v1/me/totp/disable', {
					method: 'POST',
					body: { code: totpCode }
				});
				totpSetup = null;
				totpCode = '';
			},
			on ? t('Authenticator codes are on.') : t('Authenticator codes are off.')
		);
	};

	const unlink = (i: Identity) =>
		run(() => api(`/v1/me/identities/${i.id}`, { method: 'DELETE' }), t('Unlinked.'));

	const revoke = (s: Schemas['Session']) =>
		run(async () => {
			await api(`/v1/me/sessions/${s.id}`, { method: 'DELETE' });
			if (s.current) await goto('/signin');
		}, t('Device signed out.'));

	const signOutEverywhere = () =>
		run(async () => {
			await api('/v1/me/logout-all', { method: 'POST' });
			await goto('/signin');
		});

	onMount(load);
</script>

<svelte:head><title>{t('Sign-in & security · Axon')}</title></svelte:head>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !me}
	<Status kind="loading" message={t('Loading…')} />
{:else}
	{#if banner}<p class="banner" role="status">{banner}</p>{/if}
	<Section
		id="passkeys"
		title={t('Passkeys')}
		description={t(
			'Sign in with your fingerprint, face, or device PIN. Add one on each device you use.'
		)}
	>
		<List items={passkeys} key={(p) => p.id} empty={t('No passkeys.')}>
			{#snippet row(p)}
				<span
					>{p.name}
					<span class="muted">{t('· last used {value}', { value: when(p.last_used_at) })}</span
					></span
				>
				<Button
					variant="danger"
					onclick={() => removePasskey(p)}
					disabled={busy || passkeys.length < 2}
					aria-label={t('Remove passkey {name}', { name: p.name })}>{t('Remove')}</Button
				>
			{/snippet}
		</List>
		<form onsubmit={addPasskey}>
			<TextField
				id="pk-name"
				label={t('Add a passkey')}
				placeholder={t('e.g. Phone')}
				bind:value={passkeyName}
				maxlength={100}
				required
			/>
			<Button type="submit" {busy}>{t('Add passkey')}</Button>
		</form>
	</Section>

	<Section
		id="linked"
		title={t('Linked accounts')}
		description={t(
			'Sign in with another account you link here. Axon never links one by its email address.'
		)}
	>
		<List items={identities} key={(i) => i.id} empty={t('No linked account.')}>
			{#snippet row(i)}
				<span
					>{i.provider}
					<span class="muted">· {i.email || i.name} · {t('last used')} {when(i.last_used_at)}</span
					></span
				>
				<Button variant="danger" onclick={() => unlink(i)} disabled={busy}>{t('Unlink')}</Button>
			{/snippet}
		</List>
		<SignInProviders intent="link" next="/account/security" />
	</Section>

	<Section
		id="recovery"
		title={t('Recovery codes')}
		description={t('Each code signs you in once if you lose your passkeys.')}
	>
		<p>
			{t('{recovery_codes_remaining} of 10 unused.', {
				recovery_codes_remaining: me.recovery_codes_remaining
			})}
		</p>
		{#if lowOnRecoveryCodes(me.recovery_codes_remaining)}
			<p role="status">{t('Running low: issue new codes before you run out.')}</p>
		{/if}
		{#if codes.length}<RecoveryCodes {codes} />{/if}
		<Button variant="secondary" onclick={newCodes} {busy}>{t('Issue new codes')}</Button>
	</Section>

	{#if data.status.totp_available}
		<Section
			id="totp"
			title={t('Authenticator app')}
			description={t(
				'When on, signing in with a recovery code also needs a code from your authenticator app.'
			)}
		>
			{#if me.totp_enabled}
				<form onsubmit={(e) => totpSubmit(e, false)}>
					<TextField
						id="totp-off"
						label={t('Current code, to turn it off')}
						inputmode="numeric"
						autocomplete="one-time-code"
						bind:value={totpCode}
						required
					/>
					<Button type="submit" variant="danger" {busy}>{t('Turn off')}</Button>
				</form>
			{:else if totpSetup}
				<p>{t('Add this key to your authenticator app, then enter the code it shows.')}</p>
				<p>
					<code>{totpSetup.secret}</code> · <a href={totpSetup.otpauth_url}>{t('open in app')}</a>
				</p>
				<form onsubmit={(e) => totpSubmit(e, true)}>
					<TextField
						id="totp-on"
						label={t('Code from the app')}
						inputmode="numeric"
						autocomplete="one-time-code"
						bind:value={totpCode}
						required
					/>
					<Button type="submit" {busy}>{t('Turn on')}</Button>
				</form>
			{:else}
				<Button variant="secondary" onclick={totpBegin} {busy}>{t('Set up')}</Button>
			{/if}
		</Section>
	{/if}

	<Section id="devices" title={t('Signed-in devices')}>
		<List items={sessions} key={(s) => s.id} empty={t('No other devices.')}>
			{#snippet row(s)}
				<span
					>{s.user_agent || t('Unknown device')}
					<span class="muted"
						>{s.current
							? t('· this device')
							: t('· active {when}', { when: when(s.last_seen_at) })}</span
					></span
				>
				<Button variant="secondary" onclick={() => revoke(s)} {busy}>{t('Sign out')}</Button>
			{/snippet}
		</List>
		<Button variant="danger" onclick={signOutEverywhere} {busy}>{t('Sign out everywhere')}</Button>
	</Section>
{/if}

<style>
	.banner {
		padding: var(--space-3);
		border-radius: var(--radius);
		background: var(--accent-soft);
	}
</style>
