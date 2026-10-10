<script lang="ts">
	import { onMount } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import List from '$lib/ui/List.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';

	const labels: Record<string, string> = {
		'account.created': 'Account created',
		'account.updated': 'Name changed',
		'signin.passkey': 'Signed in with a passkey',
		'signin.recovery_code': 'Signed in with a recovery code',
		signout: 'Signed out',
		'session.revoked': 'Device signed out',
		'sessions.revoked_all': 'Signed out everywhere',
		'passkey.added': 'Passkey added',
		'passkey.removed': 'Passkey removed',
		'recovery_codes.issued': 'New recovery codes issued',
		'totp.enabled': 'Authenticator app turned on',
		'totp.disabled': 'Authenticator app turned off',
		'consent.changed': 'Privacy choice changed',
		'data.exported': 'Data downloaded',
		'token.created': 'Access token created',
		'token.revoked': 'Access token revoked'
	};

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

<svelte:head><title>Activity · Axon</title></svelte:head>

<Section
	id="activity"
	title="Recent activity"
	description="Security-relevant events on your account. If something here was not you, sign out everywhere and replace your passkeys."
>
	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !events}
		<Status kind="loading" message="Loading…" />
	{:else}
		<List items={events} key={(e) => e.id} empty="No activity yet.">
			{#snippet row(e)}
				<span>{labels[e.action] ?? e.action}</span>
				<span class="muted"
					>{new Date(e.created_at).toLocaleString()} · {e.ip || 'unknown address'}</span
				>
			{/snippet}
		</List>
	{/if}
</Section>
