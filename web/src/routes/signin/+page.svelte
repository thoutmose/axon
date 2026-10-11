<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { api, message } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import TextField from '$lib/ui/TextField.svelte';

	let busy = $state(false);
	let error = $state('');
	let code = $state('');
	let totp = $state('');

	// Only same-site paths, so a crafted link cannot send you elsewhere.
	const next = $derived.by(() => {
		const n = page.url.searchParams.get('next') ?? '/';
		return n.startsWith('/') && !n.startsWith('//') ? n : '/';
	});

	async function attempt(action: () => Promise<unknown>) {
		busy = true;
		error = '';
		try {
			await action();
			await goto(next);
		} catch (err) {
			error = message(err);
		} finally {
			busy = false;
		}
	}

	const withPasskey = () =>
		attempt(async () => {
			const options = await api<{ publicKey: never }>('/v1/auth/login/begin', { method: 'POST' });
			const response = await startAuthentication({ optionsJSON: options.publicKey });
			await api('/v1/auth/login/finish', { method: 'POST', body: response });
		});

	const withRecoveryCode = (e: SubmitEvent) => {
		e.preventDefault();
		return attempt(() => api('/v1/auth/recovery', { method: 'POST', body: { code, totp } }));
	};
</script>

<svelte:head><title>Sign in · Axon</title></svelte:head>

<h1>{t('Sign in')}</h1>
<p class="muted">{t('Use the passkey on this device or your phone.')}</p>
<p><Button onclick={withPasskey} {busy}>{t('Sign in with a passkey')}</Button></p>
{#if error}<p class="error" role="alert">{error}</p>{/if}

<Section
	id="recovery"
	title="Lost your passkey?"
	description="Use one of the recovery codes you saved when you created your account."
>
	<form onsubmit={withRecoveryCode}>
		<TextField
			id="code"
			label="Recovery code"
			placeholder="xxxx-xxxx-xxxx-xxxx"
			autocomplete="off"
			bind:value={code}
			required
		/>
		<TextField
			id="totp"
			label="Authenticator code"
			hint="Only if you turned on the authenticator app."
			inputmode="numeric"
			autocomplete="one-time-code"
			bind:value={totp}
		/>
		<Button type="submit" variant="secondary" {busy}>{t('Sign in with recovery code')}</Button>
	</form>
</Section>

<style>
	.error {
		color: var(--bad);
	}
</style>
