<script lang="ts">
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
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let me = $state<Schemas['Me'] | null>(null);
	let passkeys = $state<Schemas['Passkey'][]>([]);
	let sessions = $state<Schemas['Session'][]>([]);
	let error = $state('');
	let busy = $state(false);
	let passkeyName = $state('');
	let codes = $state<string[]>([]);
	let totpSetup = $state<{ secret: string; otpauth_url: string } | null>(null);
	let totpCode = $state('');

	const when = (iso: string | null) => (iso ? new Date(iso).toLocaleString() : 'never');

	async function load() {
		error = '';
		try {
			[me, { items: passkeys }, { items: sessions }] = await Promise.all([
				api<Schemas['Me']>('/v1/me'),
				api<{ items: Schemas['Passkey'][] }>('/v1/me/passkeys'),
				api<{ items: Schemas['Session'][] }>('/v1/me/sessions')
			]);
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
			const options = await api<{ publicKey: never }>('/v1/me/passkeys/begin', {
				method: 'POST',
				body: { name: passkeyName }
			});
			const response = await startRegistration({ optionsJSON: options.publicKey });
			await api('/v1/me/passkeys/finish', { method: 'POST', body: response });
			passkeyName = '';
		}, 'Passkey added.');
	};

	const removePasskey = (p: Schemas['Passkey']) =>
		run(() => api(`/v1/me/passkeys/${p.id}`, { method: 'DELETE' }), `Removed “${p.name}”.`);

	const newCodes = () =>
		run(async () => {
			codes = (await api<{ recovery_codes: string[] }>('/v1/me/recovery-codes', { method: 'POST' }))
				.recovery_codes;
		}, 'New recovery codes issued. The old ones no longer work.');

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
			on ? 'Authenticator codes are on.' : 'Authenticator codes are off.'
		);
	};

	const revoke = (s: Schemas['Session']) =>
		run(async () => {
			await api(`/v1/me/sessions/${s.id}`, { method: 'DELETE' });
			if (s.current) await goto('/signin');
		}, 'Device signed out.');

	const signOutEverywhere = () =>
		run(async () => {
			await api('/v1/me/logout-all', { method: 'POST' });
			await goto('/signin');
		});

	onMount(load);
</script>

<svelte:head><title>Sign-in & security · Axon</title></svelte:head>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !me}
	<Status kind="loading" message="Loading…" />
{:else}
	<Section
		id="passkeys"
		title="Passkeys"
		description="Sign in with your fingerprint, face, or device PIN. Add one on each device you use."
	>
		<List items={passkeys} key={(p) => p.id} empty="No passkeys.">
			{#snippet row(p)}
				<span>{p.name} <span class="muted">· last used {when(p.last_used_at)}</span></span>
				<Button
					variant="danger"
					onclick={() => removePasskey(p)}
					disabled={busy || passkeys.length < 2}
					aria-label={`Remove passkey ${p.name}`}>Remove</Button
				>
			{/snippet}
		</List>
		<form onsubmit={addPasskey}>
			<TextField
				id="pk-name"
				label="Add a passkey"
				placeholder="e.g. Phone"
				bind:value={passkeyName}
				maxlength={100}
				required
			/>
			<Button type="submit" {busy}>Add passkey</Button>
		</form>
	</Section>

	<Section
		id="recovery"
		title="Recovery codes"
		description="Each code signs you in once if you lose your passkeys."
	>
		<p>{me.recovery_codes_remaining} of 10 unused.</p>
		{#if codes.length}<RecoveryCodes {codes} />{/if}
		<Button variant="secondary" onclick={newCodes} {busy}>Issue new codes</Button>
	</Section>

	{#if data.status.totp_available}
		<Section
			id="totp"
			title="Authenticator app"
			description="When on, signing in with a recovery code also needs a code from your authenticator app."
		>
			{#if me.totp_enabled}
				<form onsubmit={(e) => totpSubmit(e, false)}>
					<TextField
						id="totp-off"
						label="Current code, to turn it off"
						inputmode="numeric"
						autocomplete="one-time-code"
						bind:value={totpCode}
						required
					/>
					<Button type="submit" variant="danger" {busy}>Turn off</Button>
				</form>
			{:else if totpSetup}
				<p>Add this key to your authenticator app, then enter the code it shows.</p>
				<p><code>{totpSetup.secret}</code> · <a href={totpSetup.otpauth_url}>open in app</a></p>
				<form onsubmit={(e) => totpSubmit(e, true)}>
					<TextField
						id="totp-on"
						label="Code from the app"
						inputmode="numeric"
						autocomplete="one-time-code"
						bind:value={totpCode}
						required
					/>
					<Button type="submit" {busy}>Turn on</Button>
				</form>
			{:else}
				<Button variant="secondary" onclick={totpBegin} {busy}>Set up</Button>
			{/if}
		</Section>
	{/if}

	<Section id="devices" title="Signed-in devices">
		<List items={sessions} key={(s) => s.id} empty="No other devices.">
			{#snippet row(s)}
				<span
					>{s.user_agent || 'Unknown device'}
					<span class="muted">· {s.current ? 'this device' : `active ${when(s.last_seen_at)}`}</span
					></span
				>
				<Button variant="secondary" onclick={() => revoke(s)} {busy}>Sign out</Button>
			{/snippet}
		</List>
		<Button variant="danger" onclick={signOutEverywhere} {busy}>Sign out everywhere</Button>
	</Section>
{/if}
