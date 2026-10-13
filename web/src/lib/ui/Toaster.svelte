<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { dismiss, toasts } from './toast.svelte';
</script>

<div class="toaster" role="status" aria-live="polite">
	{#each toasts as item (item.id)}
		<div class="toast {item.kind}">
			<span>{item.text}</span>
			{#if item.action}
				<button
					class="action"
					onclick={() => {
						dismiss(item.id);
						item.action?.run();
					}}>{item.action.label}</button
				>
			{/if}
			<button onclick={() => dismiss(item.id)} aria-label={t('Dismiss')}>×</button>
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
	.action {
		font-weight: 650;
		text-decoration: underline;
		margin-left: auto;
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
