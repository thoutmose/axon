<script lang="ts">
	import Button from './Button.svelte';

	/** Loading, error, and empty states share one calm shape. */
	let {
		kind,
		message,
		retry
	}: { kind: 'loading' | 'error' | 'empty'; message: string; retry?: () => void } = $props();
</script>

<div class="status {kind}" role={kind === 'error' ? 'alert' : 'status'}>
	{#if kind === 'loading'}<span class="dot" aria-hidden="true"></span>{/if}
	<p>{message}</p>
	{#if kind === 'error' && retry}<Button variant="secondary" onclick={retry}>Try again</Button>{/if}
</div>

<style>
	.status {
		display: grid;
		justify-items: start;
		gap: var(--space-3);
		padding: var(--space-6) 0;
		color: var(--muted);
	}
	.error {
		color: var(--bad);
	}
	p {
		margin: 0;
	}
	.dot {
		width: 0.75rem;
		height: 0.75rem;
		border-radius: 50%;
		background: var(--accent);
		animation: pulse calc(var(--motion) * 5) ease-in-out infinite;
	}
	@keyframes pulse {
		50% {
			opacity: 0.3;
		}
	}
</style>
