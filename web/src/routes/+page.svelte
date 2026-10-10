<script lang="ts">
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Axon</title>
</svelte:head>

<main>
	<h1>Axon</h1>
	<p class="tagline">Your self-hosted life, connected.</p>

	<p class="status" role="status">
		{#if data.api.state === 'up'}
			<span class="dot up" aria-hidden="true"></span>
			API is running · version {data.api.version}
		{:else}
			<span class="dot down" aria-hidden="true"></span>
			API is not available. {data.api.reason}
		{/if}
	</p>
</main>

<style>
	:global(:root) {
		--bg: #fafaf9;
		--fg: #1c1917;
		--muted: #57534e;
		--ok: #15803d;
		--bad: #b91c1c;
	}
	@media (prefers-color-scheme: dark) {
		:global(:root) {
			--bg: #0c0a09;
			--fg: #f5f5f4;
			--muted: #a8a29e;
			--ok: #4ade80;
			--bad: #f87171;
		}
	}
	:global(body) {
		margin: 0;
		background: var(--bg);
		color: var(--fg);
		font-family: system-ui, sans-serif;
	}
	main {
		max-width: 40rem;
		margin: 0 auto;
		padding: 4rem 1rem;
	}
	h1 {
		font-size: clamp(2.5rem, 8vw, 4rem);
		margin: 0;
	}
	.tagline {
		color: var(--muted);
		font-size: 1.25rem;
	}
	.status {
		margin-top: 3rem;
		display: flex;
		align-items: center;
		gap: 0.5rem;
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
