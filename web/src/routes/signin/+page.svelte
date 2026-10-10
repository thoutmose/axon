<script lang="ts">
	import { goto } from '$app/navigation';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { api, message } from '$lib/api';

	let busy = $state(false);
	let error = $state('');
	let code = $state('');
	let totp = $state('');

	async function withPasskey() {
		busy = true;
		error = '';
		try {
			const options = await api<{ publicKey: never }>('/v1/auth/login/begin', { method: 'POST' });
			const response = await startAuthentication({ optionsJSON: options.publicKey });
			await api('/v1/auth/login/finish', { method: 'POST', body: response });
			await goto('/account');
		} catch (err) {
			error = message(err);
		} finally {
			busy = false;
		}
	}

	async function withRecoveryCode(event: SubmitEvent) {
		event.preventDefault();
		busy = true;
		error = '';
		try {
			await api('/v1/auth/recovery', { method: 'POST', body: { code, totp } });
			await goto('/account');
		} catch (err) {
			error = message(err);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>Sign in · Axon</title>
</svelte:head>

<h1>Sign in</h1>
<p><button onclick={withPasskey} disabled={busy}>Sign in with a passkey</button></p>

{#if error}<p class="error" role="alert">{error}</p>{/if}

<details>
	<summary>Lost your passkey? Use a recovery code</summary>
	<form onsubmit={withRecoveryCode}>
		<label for="code">Recovery code</label>
		<input id="code" bind:value={code} autocomplete="one-time-code" required />
		<label for="totp">Authenticator code (only if you turned it on)</label>
		<input id="totp" bind:value={totp} inputmode="numeric" autocomplete="one-time-code" />
		<p><button class="secondary" disabled={busy}>Sign in with recovery code</button></p>
	</form>
</details>
