<script lang="ts">
	import { dismiss, toasts } from './toast.svelte';
</script>

<div class="toaster" role="status" aria-live="polite">
	{#each toasts as t (t.id)}
		<div class="toast {t.kind}">
			<span>{t.text}</span>
			<button onclick={() => dismiss(t.id)} aria-label="Dismiss">×</button>
		</div>
	{/each}
</div>

<style>
	.toaster {
		position: fixed;
		bottom: var(--space-4);
		left: 50%;
		transform: translateX(-50%);
		display: grid;
		gap: var(--space-2);
		width: min(28rem, calc(100vw - 2 * var(--space-4)));
		z-index: 10;
	}
	.toast {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-4);
		border-radius: var(--radius);
		background: var(--fg);
		color: var(--bg);
		box-shadow: var(--shadow);
	}
	.toast.error {
		background: var(--bad);
		color: #fff;
	}
	button {
		background: none;
		border: none;
		color: inherit;
		font-size: var(--text-lg);
		cursor: pointer;
		min-width: 2rem;
		min-height: 2rem;
	}
</style>
