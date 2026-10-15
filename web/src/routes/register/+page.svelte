<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { startRegistration } from '@simplewebauthn/browser';
	import { api, message } from '$lib/api';
	import RecoveryCodes from '$lib/RecoveryCodes.svelte';
	import Button from '$lib/ui/Button.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { page } from '$app/state';

	let name = $state('');
	let busy = $state(false);
	let error = $state('');
	let codes = $state<string[]>([]);

	// An invitation link (EP-120) opens registration for one person.
	const invite = page.url.searchParams.get('invite') ?? '';
	let invitation = $state<{ inviter: string; household: boolean } | null>(null);
	let inviteError = $state('');
	if (invite)
		api<{ inviter: string; household: boolean }>(
			`/v1/auth/invite?token=${encodeURIComponent(invite)}`
		)
			.then((i) => (invitation = i))
			.catch((err) => (inviteError = message(err)));

	async function register(event: SubmitEvent) {
		event.preventDefault();
		busy = true;
		error = '';
		try {
			const options = await api<{ publicKey: never }>('/v1/auth/register/begin', {
				method: 'POST',
				// The account starts in this device's time zone (changeable in Account).
				body: {
					display_name: name,
					timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
					...(invite ? { invite } : {})
				}
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

<svelte:head><title>{t('Create your account · Axon')}</title></svelte:head>

{#if codes.length}
	<h1>{t('Save your recovery codes')}</h1>
	<p>{t('Each code signs you in once if you lose your passkey.')}</p>
	<RecoveryCodes {codes} />
	<Button href="/">{t('I saved them, continue')}</Button>
{:else}
	<h1>{t('Create your account')}</h1>
	{#if invitation}
		<p class="invited" role="note">
			{invitation.household
				? t('{name} invites you to their Axon, and to their household.', {
						name: invitation.inviter
					})
				: t('{name} invites you to their Axon.', { name: invitation.inviter })}
		</p>
	{:else if inviteError}
		<p class="error" role="alert">{inviteError}</p>
	{/if}
	<p class="muted">
		{t("Axon uses a passkey instead of a password: your device's fingerprint, face, or PIN.")}
	</p>
	<form onsubmit={register}>
		<TextField
			id="name"
			label={t('Your name')}
			autocomplete="name"
			maxlength={100}
			bind:value={name}
			required
		/>
		<Button type="submit" {busy}>{t('Create account with a passkey')}</Button>
	</form>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
{/if}

<style>
	.invited {
		padding: var(--space-3) var(--space-4);
		border-radius: var(--radius-lg);
		background: var(--accent-soft);
	}
	.error {
		color: var(--bad);
		margin-top: var(--space-4);
	}
</style>
