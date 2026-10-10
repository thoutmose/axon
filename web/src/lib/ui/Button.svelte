<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type Props = HTMLButtonAttributes & {
		variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
		busy?: boolean;
		href?: string;
		children: Snippet;
	};
	let {
		variant = 'primary',
		busy = false,
		href,
		disabled,
		type = 'button',
		children,
		...rest
	}: Props = $props();
</script>

{#if href}
	<a class="btn {variant}" {href}>{@render children()}</a>
{:else}
	<button class="btn {variant}" {type} disabled={disabled || busy} aria-busy={busy} {...rest}>
		{@render children()}
	</button>
{/if}

<style>
	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-4);
		border-radius: var(--radius);
		border: 1px solid transparent;
		font: inherit;
		font-weight: 600;
		text-decoration: none;
		cursor: pointer;
		transition:
			background var(--motion-fast) var(--ease),
			border-color var(--motion-fast) var(--ease);
	}
	.primary {
		background: var(--accent);
		color: var(--accent-fg);
	}
	.secondary {
		background: var(--surface);
		color: var(--fg);
		border-color: var(--border);
	}
	.secondary:hover {
		border-color: var(--accent);
	}
	.danger {
		background: transparent;
		color: var(--bad);
		border-color: var(--bad);
	}
	.danger:hover {
		background: var(--bad-soft);
	}
	.ghost {
		background: transparent;
		color: var(--accent);
		padding-inline: var(--space-2);
	}
	.btn:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}
	.btn[aria-busy='true'] {
		cursor: progress;
	}
</style>
