<script lang="ts">
	import CaptureDialog from '$lib/CaptureDialog.svelte';
	import CommandPalette from '$lib/CommandPalette.svelte';
	import { capturing } from '$lib/capture.svelte';

	let { children } = $props();
	let palette = $state(false);

	// "n" captures a task from any signed-in page, unless you are typing.
	function onkeydown(e: KeyboardEvent) {
		// Ctrl/Cmd+K opens the command center from anywhere, even while typing.
		if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
			e.preventDefault();
			palette = !palette;
			return;
		}
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
<CommandPalette bind:open={palette} />
