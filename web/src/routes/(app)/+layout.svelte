<script lang="ts">
	import CaptureDialog from '$lib/CaptureDialog.svelte';
	import { capturing } from '$lib/capture.svelte';

	let { children } = $props();

	// "n" captures a task from any signed-in page, unless you are typing.
	function onkeydown(e: KeyboardEvent) {
		const el = e.target as HTMLElement;
		const typing = el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName);
		if (e.key === 'n' && !typing && !e.metaKey && !e.ctrlKey && !e.altKey && !capturing.open) {
			e.preventDefault();
			capturing.open = true;
		}
	}
</script>

<svelte:window {onkeydown} />
{@render children()}
<CaptureDialog bind:open={capturing.open} oncaptured={() => capturing.count++} />
