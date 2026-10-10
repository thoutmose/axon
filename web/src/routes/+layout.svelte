<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import Toaster from '$lib/ui/Toaster.svelte';
	import { connection, watchConnection } from '$lib/connection.svelte';
	import { applyTheme, storedTheme, type Theme } from '$lib/theme';

	let { children } = $props();
	let theme = $state<Theme>('system');

	onMount(() => {
		theme = storedTheme();
		applyTheme(theme);
		return watchConnection();
	});

	function cycleTheme() {
		theme = theme === 'system' ? 'dark' : theme === 'dark' ? 'light' : 'system';
		applyTheme(theme);
	}
</script>

<a class="skip" href="#main">Skip to content</a>
<header>
	<div class="bar">
		<a href="/" class="brand">Axon</a>
		<nav aria-label="Main">
			<a href="/">Home</a>
			<a href="/tasks">Tasks</a>
			<a href="/notes">Notes</a>
			<a href="/account">Account</a>
			<button class="theme" onclick={cycleTheme} aria-label="Theme: {theme}. Change theme">
				{theme === 'dark' ? '☾' : theme === 'light' ? '☀' : '◐'}
			</button>
		</nav>
	</div>
</header>

{#if !connection.online}
	<p class="banner" role="alert">You are offline. Changes will not be saved until you reconnect.</p>
{:else if connection.apiDown}
	<p class="banner" role="alert">
		Axon's server is not responding. Your data is safe; try again shortly.
	</p>
{/if}

<main id="main">
	{@render children()}
</main>
<Toaster />

<style>
	header {
		position: sticky;
		top: 0;
		z-index: 5;
		background: var(--surface-glass);
		backdrop-filter: saturate(1.4) blur(12px);
		border-bottom: 1px solid var(--border);
	}
	.bar {
		max-width: var(--content-width);
		margin: 0 auto;
		padding: var(--space-3) var(--space-4);
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-4);
	}
	.brand {
		font-weight: 750;
		font-size: var(--text-lg);
		color: var(--fg);
		text-decoration: none;
		letter-spacing: -0.02em;
	}
	nav {
		display: flex;
		align-items: center;
		gap: var(--space-4);
	}
	nav a {
		color: var(--fg);
		text-decoration: none;
	}
	nav a:hover {
		color: var(--accent);
	}
	.theme {
		background: none;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		color: var(--fg);
		min-width: 2.5rem;
		min-height: 2.5rem;
		cursor: pointer;
		font-size: var(--text-lg);
	}
	main {
		max-width: var(--content-width);
		margin: 0 auto;
		padding: var(--space-8) var(--space-4) var(--space-16);
	}
	.banner {
		margin: 0;
		padding: var(--space-3) var(--space-4);
		text-align: center;
		background: var(--bad-soft);
		color: var(--bad);
	}
	.skip {
		position: absolute;
		left: -999px;
	}
	.skip:focus {
		left: var(--space-4);
		top: var(--space-4);
		z-index: 20;
		background: var(--surface);
		padding: var(--space-2) var(--space-3);
	}
</style>
