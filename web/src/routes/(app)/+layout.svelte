<script lang="ts">
	import CaptureDialog from '$lib/CaptureDialog.svelte';
	import CommandPalette from '$lib/CommandPalette.svelte';
	import { searching } from '$lib/search.svelte';
	import { capturing } from '$lib/capture.svelte';

	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { setLocale, storedLocale, t } from '$lib/i18n.svelte';
	import { applyAppearance, type Appearance } from '$lib/preferences';
	import { goto } from '$app/navigation';
	import ShortcutsHelp from '$lib/ShortcutsHelp.svelte';
	import Jarvis from '$lib/Jarvis.svelte';
	import { jarvis } from '$lib/jarvis.svelte';
	import { chord, typing } from '$lib/shortcuts';
	import RecoveryBanner from '$lib/RecoveryBanner.svelte';
	import { recovery } from '$lib/recovery.svelte';

	let { children, data } = $props();

	// The person's saved settings win over this browser's cached ones.
	onMount(() => {
		api<{
			language: 'en' | 'fr';
			appearance: Appearance;
			keyboard: { shortcuts: boolean };
			jarvis?: { enabled: boolean };
		}>('/v1/me/settings')
			.then((s) => {
				const before = storedLocale();
				setLocale(s.language);
				// First visit in another language on this device: what the page
				// already fetched came in the old one, so fetch it again, once.
				if (s.language !== before && storedLocale() === s.language) location.reload();
				applyAppearance(s.appearance);
				shortcuts = s.keyboard.shortcuts;
				jarvis.enabled = s.jarvis?.enabled ?? false;
			})
			.catch(() => {});
		// Few recovery codes left: the banner asks for new ones.
		if (!data.status.demo_session) {
			api<{ recovery_codes_remaining: number }>('/v1/me')
				.then((me) => (recovery.remaining = me.recovery_codes_remaining))
				.catch(() => {});
		}
	});
	let help = $state(false);
	let shortcuts = $state(true);
	let pendingG = false;
	let gTimer: ReturnType<typeof setTimeout> | undefined;

	// "n" captures a task from any signed-in page, unless you are typing.
	function onkeydown(e: KeyboardEvent) {
		// Ctrl/Cmd+K opens the command center from anywhere, even while typing.
		if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
			e.preventDefault();
			searching.open = !searching.open;
			return;
		}
		if (!shortcuts || typing(e.target) || e.metaKey || e.ctrlKey || e.altKey) return;
		if (capturing.open || searching.open) return; // they have inputs; the help dialog does not
		if (pendingG) {
			pendingG = false;
			clearTimeout(gTimer);
			const href = chord(e.key);
			if (href) {
				e.preventDefault();
				help = false;
				goto(href);
			}
			return;
		}
		switch (e.key) {
			case 'n':
				e.preventDefault();
				capturing.open = true;
				break;
			case '/':
				e.preventDefault();
				searching.open = true;
				break;
			case '?':
				e.preventDefault();
				help = true;
				break;
			case 'g':
				pendingG = true;
				gTimer = setTimeout(() => (pendingG = false), 1500);
				break;
		}
	}
</script>

<svelte:window {onkeydown} />
{#if data.status.demo_session}
	<p class="demo-banner" role="note">
		{t(
			'You are in the shared demo: sample data that resets every night. Passkeys, tokens, AI providers, and repositories are off, and anything you change is visible to other visitors.'
		)}
	</p>
{/if}
<RecoveryBanner />
{@render children()}
<CaptureDialog bind:open={capturing.open} oncaptured={() => capturing.count++} />
{#if jarvis.enabled}<Jarvis />{/if}
<CommandPalette bind:open={searching.open} />
<ShortcutsHelp bind:open={help} />

<style>
	.demo-banner {
		margin: 0 0 var(--space-4);
		padding: var(--space-2) var(--space-4);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--accent-soft);
		font-size: var(--text-sm);
	}
</style>
