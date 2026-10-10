<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		id,
		children
	}: { title: string; description?: string; id: string; children: Snippet } = $props();
</script>

<section class="section" aria-labelledby={id}>
	<h2 {id}>{title}</h2>
	{#if description}<p class="muted">{description}</p>{/if}
	{@render children()}
</section>

<style>
	.section {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-6);
		margin-bottom: var(--space-6);
		box-shadow: var(--shadow);
	}
	@media (max-width: 30rem) {
		.section {
			padding: var(--space-4);
		}
	}
</style>
