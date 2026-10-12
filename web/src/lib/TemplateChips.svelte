<script lang="ts">
	let {
		label = 'Templates',
		items,
		onpick,
		busy = false
	}: {
		label?: string;
		items: { title: string; hint?: string }[];
		onpick: (index: number) => void;
		busy?: boolean;
	} = $props();
</script>

<div class="templates" role="group" aria-label={label}>
	<span class="muted small">{label}:</span>
	{#each items as t, i (t.title)}
		<button type="button" disabled={busy} onclick={() => onpick(i)} title={t.hint}
			>＋ {t.title}</button
		>
	{/each}
</div>

<style>
	.templates {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
		margin: 0 0 var(--space-6);
	}
	.small {
		font-size: var(--text-sm);
	}
	button {
		font: inherit;
		font-size: var(--text-sm);
		border: 1px dashed var(--border);
		background: var(--surface);
		color: var(--fg);
		border-radius: 999px;
		padding: var(--space-1) var(--space-3);
		cursor: pointer;
		min-height: 2rem;
	}
	button:hover:not(:disabled) {
		border-color: var(--accent);
	}
	button:disabled {
		opacity: 0.6;
		cursor: progress;
	}
</style>
