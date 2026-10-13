<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import List from '$lib/ui/List.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';

	const labels: Record<string, string> = $derived({
		'account.created': t('Account created'),
		'account.updated': t('Name changed'),
		'signin.passkey': t('Signed in with a passkey'),
		'signin.recovery_code': t('Signed in with a recovery code'),
		signout: t('Signed out'),
		'session.revoked': t('Device signed out'),
		'sessions.revoked_all': t('Signed out everywhere'),
		'passkey.added': t('Passkey added'),
		'passkey.removed': t('Passkey removed'),
		'recovery_codes.issued': t('New recovery codes issued'),
		'totp.enabled': t('Authenticator app turned on'),
		'totp.disabled': t('Authenticator app turned off'),
		'consent.changed': t('Privacy choice changed'),
		'data.exported': t('Data downloaded'),
		'token.created': t('Access token created'),
		'token.revoked': t('Access token revoked')
	});

	let events = $state<Schemas['AuditEvent'][] | null>(null);
	let error = $state('');

	async function load() {
		error = '';
		try {
			events = (await api<{ items: Schemas['AuditEvent'][] }>('/v1/me/audit')).items;
		} catch (err) {
			error = message(err);
		}
	}

	onMount(load);
</script>

<svelte:head><title>{t('Activity · Axon')}</title></svelte:head>

<Section
	id="activity"
	title={t('Recent activity')}
	description={t(
		'Security-relevant events on your account. If something here was not you, sign out everywhere and replace your passkeys.'
	)}
>
	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !events}
		<Status kind="loading" message={t('Loading…')} />
	{:else}
		<List items={events} key={(e) => e.id} empty={t('No activity yet.')}>
			{#snippet row(e)}
				<span>{labels[e.action] ?? e.action}</span>
				<span class="muted"
					>{new Date(e.created_at).toLocaleString(appLocale())} · {e.ip ||
						t('unknown address')}</span
				>
			{/snippet}
		</List>
	{/if}
</Section>
