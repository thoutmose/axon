<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { startAuthentication } from '@simplewebauthn/browser';
	import { api, message } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import SignInProviders from '$lib/SignInProviders.svelte';

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

<svelte:head><title>{t('Sign in · Axon')}</title></svelte:head>

<h1>{t('Sign in')}</h1>
<p class="muted">{t('Use the passkey on this device or your phone.')}</p>
<p><Button onclick={withPasskey} {busy}>{t('Sign in with a passkey')}</Button></p>
<SignInProviders {next} />
{#if error}<p class="error" role="alert">{error}</p>{/if}

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

<style>
	.error {
		color: var(--bad);
	}
</style>
