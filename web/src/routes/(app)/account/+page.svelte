<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, apiResponse, message, type Schemas } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let me = $state<Schemas['Me'] | null>(null);
	let etag = '';
	let name = $state('');
	let timezone = $state('UTC');
	const zones =
		typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : ['UTC'];
	const deviceZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
	let error = $state('');
	let busy = $state(false);

	async function load() {
		error = '';
		try {
			const res = await apiResponse<Schemas['Me']>('/v1/me');
			me = res.data;
			etag = res.etag;
			name = me.display_name;
			timezone = me.timezone;
		} catch (err) {
			error = message(err);
		}
	}

	async function save(body: Record<string, string>, done: string) {
		busy = true;
		try {
			await api('/v1/me', { method: 'PATCH', body, headers: { 'If-Match': etag } });
			toast(done);
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	const rename = (e: SubmitEvent) => {
		e.preventDefault();
		return save({ display_name: name }, 'Name updated.');
	};
	const setZone = (zone: string) => save({ timezone: zone }, `Time zone set to ${zone}.`);

	async function signOut() {
		busy = true;
		try {
			await api('/v1/auth/logout', { method: 'POST' });
			await goto('/');
		} catch (err) {
			toast(message(err), 'error');
			busy = false;
		}
	}

	onMount(load);
</script>

<svelte:head><title>Account · Axon</title></svelte:head>

{#if me}
	<Section id="profile" title="Name">
		<form onsubmit={rename}>
			<TextField
				id="display-name"
				label="Your name"
				hint="Shown in Axon and in your passkey manager."
				bind:value={name}
				maxlength={100}
				required
			/>
			<Button type="submit" variant="secondary" {busy} disabled={name === me.display_name}
				>Save</Button
			>
		</form>
	</Section>
	<Section
		id="zone"
		title="Time zone"
		description="Decides what “today” means for your tasks and check-ins."
	>
		<div class="zone">
			<label for="timezone" class="visually-hidden">Time zone</label>
			<select
				id="timezone"
				bind:value={timezone}
				onchange={() => setZone(timezone)}
				disabled={busy}
			>
				{#each zones as z (z)}<option value={z}>{z}</option>{/each}
			</select>
			{#if deviceZone && deviceZone !== me.timezone}
				<Button variant="ghost" onclick={() => setZone(deviceZone)} {busy}
					>Use this device's zone ({deviceZone})</Button
				>
			{/if}
		</div>
	</Section>
	<Section id="summary" title="At a glance">
		<ul class="facts">
			<li><strong>{me.passkeys}</strong> passkey{me.passkeys === 1 ? '' : 's'}</li>
			<li><strong>{me.recovery_codes_remaining}</strong> of 10 recovery codes unused</li>
			<li>Authenticator app <strong>{me.totp_enabled ? 'on' : 'off'}</strong></li>
			<li>Member since {new Date(me.created_at).toLocaleDateString()}</li>
		</ul>
		<Button variant="secondary" onclick={signOut} {busy}>Sign out</Button>
	</Section>
{:else if error}
	<Status kind="error" message={error} retry={load} />
{:else}
	<Status kind="loading" message="Loading your account…" />
{/if}

<style>
	.zone {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		align-items: center;
	}
	.zone select {
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		max-width: 100%;
	}
	.facts {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-2);
		margin: 0 0 var(--space-6);
	}
</style>
