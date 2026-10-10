<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';
	import { api, message } from '$lib/api';
	import RecoveryCodes from '$lib/RecoveryCodes.svelte';
	import Button from '$lib/ui/Button.svelte';
	import TextField from '$lib/ui/TextField.svelte';

	let name = $state('');
	let busy = $state(false);
	let error = $state('');
	let codes = $state<string[]>([]);

	async function register(event: SubmitEvent) {
		event.preventDefault();
		busy = true;
		error = '';
		try {
			const options = await api<{ publicKey: never }>('/v1/auth/register/begin', {
				method: 'POST',
				body: { display_name: name }
			});
			const response = await startRegistration({ optionsJSON: options.publicKey });
			codes = (
				await api<{ recovery_codes: string[] }>('/v1/auth/register/finish', {
					method: 'POST',
					body: response
				})
			).recovery_codes;
		} catch (err) {
			error = message(err);
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Create your account · Axon</title></svelte:head>

{#if codes.length}
	<h1>Save your recovery codes</h1>
	<p>Each code signs you in once if you lose your passkey.</p>
	<RecoveryCodes {codes} />
	<Button href="/">I saved them, continue</Button>
{:else}
	<h1>Create your account</h1>
	<p class="muted">
		Axon uses a passkey instead of a password: your device's fingerprint, face, or PIN.
	</p>
	<form onsubmit={register}>
		<TextField
			id="name"
			label="Your name"
			autocomplete="name"
			maxlength={100}
			bind:value={name}
			required
		/>
		<Button type="submit" {busy}>Create account with a passkey</Button>
	</form>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
{/if}

<style>
	.error {
		color: var(--bad);
		margin-top: var(--space-4);
	}
</style>
