<script lang="ts">
	import { icons, type IconName } from '$lib/icons';

	// A line icon from the one set (EP-129), in the text's colour. Decorative:
	// the words beside it say what it is.
	let { name, size = 16 }: { name: IconName; size?: number } = $props();
</script>

<svg
	class="icon"
	viewBox="0 0 24 24"
	width={size}
	height={size}
	fill="none"
	stroke="currentColor"
	stroke-width="1.75"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
	focusable="false"
>
	<!-- Constant markup from $lib/icons, never user input. -->
	{@html icons[name]}
</svg>

<style>
	.icon {
		flex: none;
		vertical-align: -0.15em;
	}
</style>
