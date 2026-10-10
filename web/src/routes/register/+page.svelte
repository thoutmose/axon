<script lang="ts">
	import { startRegistration } from '@simplewebauthn/browser';
	import { api, message } from '$lib/api';

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
			const result = await api<{ recovery_codes: string[] }>('/v1/auth/register/finish', {
				method: 'POST',
				body: response
			});
			codes = result.recovery_codes;
		} catch (err) {
			error = message(err);
		} finally {
			busy = false;
		}
	}

	function download() {
		const blob = new Blob([`Axon recovery codes\n\n${codes.join('\n')}\n`], { type: 'text/plain' });
		const a = document.createElement('a');
		a.href = URL.createObjectURL(blob);
		a.download = 'axon-recovery-codes.txt';
		a.click();
		URL.revokeObjectURL(a.href);
	}
</script>

<svelte:head>
	<title>Create your account · Axon</title>
</svelte:head>

{#if codes.length}
	<h1>Save your recovery codes</h1>
	<p>
		Each code signs you in once if you lose your passkey. Store them somewhere safe, such as a
		password manager. They will not be shown again.
	</p>
	<ul class="codes">
		{#each codes as c (c)}<li>{c}</li>{/each}
	</ul>
	<p>
		<button class="secondary" onclick={download}>Download codes</button>
		<a href="/account">I saved them, continue</a>
	</p>
{:else}
	<h1>Create your account</h1>
	<p class="muted">
		Axon uses a passkey instead of a password: your device's fingerprint, face, or PIN.
	</p>
	<form onsubmit={register}>
		<label for="name">Your name</label>
		<input id="name" bind:value={name} maxlength="100" autocomplete="name" required />
		<p><button disabled={busy}>Create account with a passkey</button></p>
	</form>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
{/if}
