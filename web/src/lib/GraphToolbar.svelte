<script lang="ts">
	import { t } from '$lib/i18n.svelte';

	// The knowledge graph's tools (EP-122): find a node, zoom, fit, and
	// full screen. The graph itself also pans and zooms with the pointer.
	let {
		nodes,
		onfind,
		onzoom,
		onfit,
		fullscreen
	}: {
		nodes: { id: string; label: string }[];
		onfind: (id: string) => void;
		onzoom: (factor: number) => void;
		onfit: () => void;
		fullscreen?: HTMLElement;
	} = $props();
	const uid = $props.id();
	let query = $state('');

	function find(e: SubmitEvent) {
		e.preventDefault();
		const q = query.trim().toLowerCase();
		const hit =
			nodes.find((n) => n.label.toLowerCase() === q) ??
			nodes.find((n) => n.label.toLowerCase().includes(q));
		if (hit) onfind(hit.id);
	}
	function toggleFullscreen() {
		if (document.fullscreenElement) document.exitFullscreen();
		else fullscreen?.requestFullscreen();
	}
</script>

<div class="tools">
	<form onsubmit={find} role="search">
		<label for="{uid}-find" class="visually-hidden">{t('Find in the graph')}</label>
		<input
			id="{uid}-find"
			list="{uid}-names"
			bind:value={query}
			placeholder={t('Find in the graph…')}
			autocomplete="off"
		/>
		<datalist id="{uid}-names">
			{#each nodes as n (n.id)}<option value={n.label}></option>{/each}
		</datalist>
	</form>
	<div class="zoom" role="group" aria-label={t('Zoom')}>
		<button type="button" onclick={() => onzoom(1.4)} aria-label={t('Zoom in')}>+</button>
		<button type="button" onclick={() => onzoom(1 / 1.4)} aria-label={t('Zoom out')}>−</button>
		<button type="button" onclick={onfit}>{t('Fit')}</button>
		{#if fullscreen}
			<button type="button" onclick={toggleFullscreen}>{t('Full screen')}</button>
		{/if}
	</div>
	<p class="hint small muted">{t('Scroll or pinch to zoom, drag to move.')}</p>
</div>

<style>
	.tools {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
		margin: var(--space-2) 0;
	}
	input {
		font: inherit;
		min-height: 2.25rem;
		padding: var(--space-1) var(--space-3);
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--fg);
		min-width: 14rem;
	}
	.zoom {
		display: flex;
		gap: var(--space-1);
	}
	.zoom button {
		font: inherit;
		min-width: 2.25rem;
		min-height: 2.25rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--fg);
		cursor: pointer;
		padding: 0 var(--space-3);
	}
	.zoom button:hover {
		border-color: var(--accent);
	}
	.hint {
		margin: 0;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
