<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type AuthStatus } from '$lib/api';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let status = $state<AuthStatus | null>(null);

	onMount(async () => {
		status = await api<AuthStatus>('/v1/auth/status').catch(() => null);
	});
</script>

<svelte:head>
	<title>Axon</title>
</svelte:head>

<h1>Axon</h1>
<p class="muted">Your self-hosted life, connected.</p>

{#if status?.signed_in}
	<p>You are signed in. <a href="/account">Manage your account</a>.</p>
{:else if status}
	<p>
		<a href="/signin">Sign in</a>{#if status.registration_open}
			or <a href="/register">create your account</a>{/if}.
	</p>
{/if}

<p class="status" role="status">
	{#if data.api.state === 'up'}
		<span class="dot up" aria-hidden="true"></span>
		API is running · version {data.api.version}
	{:else}
		<span class="dot down" aria-hidden="true"></span>
		API is not available. {data.api.reason}
	{/if}
</p>

<style>
	.status {
		margin-top: 3rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: var(--muted);
	}
	.dot {
		width: 0.625rem;
		height: 0.625rem;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.up {
		background: var(--ok);
	}
	.down {
		background: var(--bad);
	}
</style>
