<script lang="ts">
	import { onMount } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import List from '$lib/ui/List.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let tokens = $state<Schemas['Token'][] | null>(null);
	let error = $state('');
	let busy = $state(false);
	let name = $state('');
	let scope = $state<'read' | 'write'>('read');
	let days = $state(90);
	let secret = $state('');

	async function load() {
		error = '';
		try {
			tokens = (await api<{ items: Schemas['Token'][] }>('/v1/me/tokens')).items;
		} catch (err) {
			error = message(err);
		}
	}

	async function create(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			const res = await api<{ token: string }>('/v1/me/tokens', {
				method: 'POST',
				body: { name, scope, expires_in_days: days }
			});
			secret = res.token;
			name = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function revoke(t: Schemas['Token']) {
		busy = true;
		try {
			await api(`/v1/me/tokens/${t.id}`, { method: 'DELETE' });
			toast(`Revoked “${t.name}”.`);
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	onMount(load);
</script>

<svelte:head><title>Access tokens · Axon</title></svelte:head>

<Section
	id="tokens"
	title="Access tokens"
	description="For scripts and sync tools. A token can read (or change) your data, but never manage your account, export it, or delete it."
>
	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !tokens}
		<Status kind="loading" message="Loading…" />
	{:else}
		<List items={tokens} key={(t) => t.id} empty="No tokens yet.">
			{#snippet row(t)}
				<span
					>{t.name}
					<span class="muted"
						>· {t.scope === 'read' ? 'read only' : 'read and write'} · expires {new Date(
							t.expires_at
						).toLocaleDateString()}</span
					></span
				>
				<Button variant="secondary" onclick={() => revoke(t)} {busy}>Revoke</Button>
			{/snippet}
		</List>
	{/if}
	{#if secret}
		<p role="status">Copy this token now; it will not be shown again:</p>
		<p><code>{secret}</code></p>
	{/if}
	<form onsubmit={create}>
		<TextField
			id="token-name"
			label="Name"
			placeholder="e.g. Notes sync"
			bind:value={name}
			maxlength={100}
			required
		/>
		<div class="field">
			<label for="token-scope">Access</label>
			<select id="token-scope" bind:value={scope}>
				<option value="read">Read only</option>
				<option value="write">Read and write</option>
			</select>
		</div>
		<TextField
			id="token-days"
			label="Expires after (days)"
			type="number"
			min={1}
			max={365}
			bind:value={days}
			required
		/>
		<Button type="submit" {busy}>Create token</Button>
	</form>
</Section>

<style>
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
		max-width: 28rem;
	}
	label {
		font-weight: 600;
	}
	select {
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
</style>
