<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';
	import { page } from '$app/state';
	import { startAuthentication, startRegistration } from '@simplewebauthn/browser';
	import { api, message } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import SignInProviders from '$lib/SignInProviders.svelte';
	import { appLocale } from '$lib/dates';
	import { requirePasskeys } from '$lib/recovery.svelte';

	let busy = $state(false);
	// A provider sign-in that did not work comes back with ?error=.
	const reasons: Record<string, string> = {
		not_linked:
			'That account is not linked to an Axon account here. Sign in with your passkey, then link it in Account → Sign-in & security.',
		state: 'That sign-in was started in another browser or tab, or took too long. Try again.',
		expired: 'That sign-in took too long. Try again.',
		cancelled: 'Sign-in was cancelled.',
		provider: 'The other service did not confirm who you are. Try again.'
	};
	let error = $state(
		reasons[page.url.searchParams.get('error') ?? '']
			? t(reasons[page.url.searchParams.get('error')!])
			: ''
	);
	let code = $state('');
	let totp = $state('');
	// A passkey that did not work: offer to try again or to use a recovery code.
	let passkeyFailed = $state(false);
	// Signed in with a recovery code: offer a passkey on this device, once.
	let offer = $state(false);
	let offerHeading = $state<HTMLElement>();

	// Only same-site paths, so a crafted link cannot send you elsewhere.
	const next = $derived.by(() => {
		const n = page.url.searchParams.get('next') ?? '/';
		return n.startsWith('/') && !n.startsWith('//') ? n : '/';
	});

	async function attempt(action: () => Promise<unknown>, afterwards = () => goto(next)) {
		busy = true;
		error = '';
		try {
			await action();
			await afterwards();
		} catch (err) {
			error = message(err);
		} finally {
			busy = false;
		}
	}

	const withPasskey = async () => {
		passkeyFailed = false;
		await attempt(async () => {
			try {
				requirePasskeys();
				await passkeyCeremony();
			} catch (err) {
				passkeyFailed = true;
				throw err;
			}
		});
	};

	const passkeyCeremony = async () => {
		const options = await api<{ publicKey: never }>('/v1/auth/login/begin', { method: 'POST' });
		const response = await startAuthentication({ optionsJSON: options.publicKey });
		await api('/v1/auth/login/finish', { method: 'POST', body: response });
	};

	const withRecoveryCode = (e: SubmitEvent) => {
		e.preventDefault();
		return attempt(
			() => api('/v1/auth/recovery', { method: 'POST', body: { code, totp } }),
			async () => {
				offer = true;
				await tick();
				offerHeading?.focus();
			}
		);
	};

	const addHerePasskey = () =>
		attempt(async () => {
			requirePasskeys();
			const name = t('Passkey from {date}', { date: new Date().toLocaleDateString(appLocale()) });
			const options = await api<{ publicKey: never }>('/v1/me/passkeys/begin', {
				method: 'POST',
				body: { name }
			});
			const response = await startRegistration({ optionsJSON: options.publicKey });
			await api('/v1/me/passkeys/finish', { method: 'POST', body: response });
		});

	const useRecoveryCode = () => document.getElementById('code')?.focus();
</script>

<svelte:head><title>{t('Sign in · Axon')}</title></svelte:head>

{#if offer}
	<h1 tabindex="-1" bind:this={offerHeading}>{t('You are signed in')}</h1>
	<p>
		{t('You used a recovery code. Add a passkey on this device so you do not need one next time.')}
	</p>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	<p class="actions">
		<Button onclick={addHerePasskey} {busy}>{t('Add a passkey on this device')}</Button>
		<Button variant="secondary" onclick={() => goto(next)} disabled={busy}>{t('Not now')}</Button>
	</p>
{:else}
	<h1>{t('Sign in')}</h1>
	<p class="muted">{t('Use the passkey on this device or your phone.')}</p>
	<p><Button onclick={withPasskey} {busy}>{t('Sign in with a passkey')}</Button></p>
	<SignInProviders {next} />
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if passkeyFailed}
		<p class="actions">
			{t('Try again, or sign in with a recovery code.')}
			<Button variant="secondary" onclick={withPasskey} {busy}>{t('Try again')}</Button>
			<Button variant="secondary" onclick={useRecoveryCode}>{t('Use a recovery code')}</Button>
		</p>
	{/if}

	<Section
		id="recovery"
		title={t('Lost your passkey?')}
		description={t('Use one of the recovery codes you saved when you created your account.')}
	>
		<form onsubmit={withRecoveryCode}>
			<TextField
				id="code"
				label={t('Recovery code')}
				placeholder={t('xxxx-xxxx-xxxx-xxxx')}
				autocomplete="off"
				bind:value={code}
				required
			/>
			<TextField
				id="totp"
				label={t('Authenticator code')}
				hint={t('Only if you turned on the authenticator app.')}
				inputmode="numeric"
				autocomplete="one-time-code"
				bind:value={totp}
			/>
			<Button type="submit" variant="secondary" {busy}>{t('Sign in with recovery code')}</Button>
		</form>
	</Section>
{/if}

<style>
	.error {
		color: var(--bad);
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
	}
</style>
