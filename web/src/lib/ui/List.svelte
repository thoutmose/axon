<script lang="ts" generics="T">
	import type { Snippet } from 'svelte';

	let {
		items,
		key,
		row,
		empty
	}: { items: T[]; key: (item: T) => string; row: Snippet<[T]>; empty: string } = $props();
</script>

{#if items.length === 0}
	<p class="muted">{empty}</p>
{:else}
	<ul>
		{#each items as item (key(item))}
			<li>{@render row(item)}</li>
		{/each}
	</ul>
{/if}

<style>
	ul {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-4);
	}
	li {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--border);
	}
	li:last-child {
		border-bottom: none;
	}
</style>
