<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type AuthStatus } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let status = $state<AuthStatus | null>(null);

	onMount(async () => {
		status = await api<AuthStatus>('/v1/auth/status').catch(() => null);
	});
</script>

<svelte:head><title>Axon</title></svelte:head>

<section class="hero">
	<h1>Your self-hosted life, connected.</h1>
	<p class="lead">
		Notes, action, learning, health, projects, and reflection in one humane graph that you own.
	</p>
	{#if status?.signed_in}
		<p>Welcome back. Your daily view arrives with the dashboard; for now, manage your account.</p>
		<Button href="/account">Open your account</Button>
	{:else if status}
		<div class="actions">
			<Button href="/signin">Sign in</Button>
			{#if status.registration_open}<Button href="/register" variant="secondary"
					>Create your account</Button
				>{/if}
		</div>
	{/if}
</section>

<p class="status" role="status">
	<span class="dot {data.api.state}" aria-hidden="true"></span>
	{#if data.api.state === 'up'}
		Server running · version {data.api.version}
	{:else}
		Server not available. {data.api.reason}
	{/if}
</p>

<style>
	.hero {
		padding: var(--space-8) 0 var(--space-12);
	}
	.lead {
		font-size: var(--text-xl);
		color: var(--muted);
		max-width: 36rem;
	}
	.actions {
		display: flex;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	.status {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		color: var(--muted);
		font-size: var(--text-sm);
	}
	.dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		background: var(--bad);
	}
	.dot.up {
		background: var(--ok);
	}
</style>
