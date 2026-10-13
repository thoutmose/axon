<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { noteType } from '$lib/noteTypes';

	// A note type's illustration with what it is for (EP-98).
	let { kind, compact = false }: { kind: string; compact?: boolean } = $props();
	const nt = $derived(noteType(kind));
</script>

<figure class="note-type" class:compact>
	<svg viewBox="0 0 48 48" width={compact ? 28 : 44} height={compact ? 28 : 44} aria-hidden="true">
		<path
			d={nt.d}
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
	</svg>
	<figcaption>
		<strong>{t(nt.title)}</strong>
		{#if !compact}<span>{t(nt.about)}</span>{/if}
	</figcaption>
</figure>

<style>
	.note-type {
		display: flex;
		gap: var(--space-3);
		align-items: center;
		margin: var(--space-2) 0;
		color: var(--accent);
	}
	figcaption {
		display: grid;
		color: var(--muted);
		font-size: var(--text-sm);
	}
	strong {
		color: var(--fg);
		font-weight: 600;
	}
	.compact figcaption {
		display: inline;
	}
</style>
