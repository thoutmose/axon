<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { coverHue } from '$lib/media';
	import type { Schemas } from '$lib/api';

	// A film's, series', or book's cover, or one drawn from its title when the
	// catalogues have none (EP-17, EP-135).
	let { item, kind }: { item: Schemas['MediaItem']; kind: string } = $props();
</script>

{#if item.cover_id}
	<img class="cover" src="/api/v1/media/covers/{item.cover_id}" alt="" loading="lazy" />
{:else}
	<span class="cover generated" style:--hue={coverHue(item.title)} aria-hidden="true">
		<span class="g-kind">{t(kind)}</span>
		<span class="g-title">{item.title}</span>
		{#if item.creators}<span class="g-by">{item.creators}</span>{/if}
	</span>
{/if}

<style>
	.cover {
		width: 100%;
		aspect-ratio: 2 / 3;
		object-fit: cover;
		display: block;
		border-radius: var(--radius);
	}
	.generated {
		display: grid;
		grid-template-rows: auto 1fr auto;
		gap: var(--space-2);
		padding: var(--space-3);
		text-align: left;
		color: #fff;
		background: linear-gradient(
			160deg,
			hsl(var(--hue) 45% 30%),
			hsl(calc(var(--hue) + 40) 50% 16%)
		);
	}
	.g-kind {
		font-size: 0.75rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		opacity: 0.85;
	}
	.g-title {
		font-weight: 700;
		font-size: var(--text-lg);
		line-height: 1.2;
		overflow-wrap: anywhere;
		display: -webkit-box;
		-webkit-line-clamp: 5;
		line-clamp: 5;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.g-by {
		font-size: var(--text-sm);
		opacity: 0.9;
	}
</style>
