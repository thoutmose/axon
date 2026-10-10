<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	type Props = HTMLInputAttributes & {
		label: string;
		id: string;
		hint?: string;
		error?: string;
		value?: string | number;
	};
	let { label, id, hint, error, value = $bindable(''), ...rest }: Props = $props();
	const describedBy = $derived(
		[hint && `${id}-hint`, error && `${id}-error`].filter(Boolean).join(' ') || undefined
	);
</script>

<div class="field">
	<label for={id}>{label}</label>
	{#if hint}<p class="hint" id="{id}-hint">{hint}</p>{/if}
	<input
		{id}
		bind:value
		aria-invalid={error ? true : undefined}
		aria-describedby={describedBy}
		{...rest}
	/>
	{#if error}<p class="error" id="{id}-error">{error}</p>{/if}
</div>

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
	.hint {
		margin: 0;
		color: var(--muted);
		font-size: var(--text-sm);
	}
	.error {
		margin: 0;
		color: var(--bad);
		font-size: var(--text-sm);
	}
	input {
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		width: 100%;
	}
	input[aria-invalid='true'] {
		border-color: var(--bad);
	}
</style>
