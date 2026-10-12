<script lang="ts">
	import { page } from '$app/state';
	import { t as tr } from '$lib/i18n.svelte';

	let { children } = $props();
	const tabs = [
		{ href: '/account', label: 'Overview' },
		{ href: '/account/security', label: 'Sign-in & security' },
		{ href: '/account/tokens', label: 'Access tokens' },
		{ href: '/account/privacy', label: 'Privacy & data' },
		{ href: '/account/import', label: 'Import' },
		{ href: '/account/trash', label: 'Trash' },
		{ href: '/account/preferences', label: 'Preferences' },
		{ href: '/account/activity', label: 'Activity' }
	];
</script>

<h1>{tr('Account')}</h1>
<nav aria-label={tr('Account sections')} class="tabs">
	{#each tabs as t (t.href)}
		<a href={t.href} aria-current={page.url.pathname === t.href ? 'page' : undefined}
			>{tr(t.label)}</a
		>
	{/each}
</nav>
{@render children()}

<style>
	.tabs {
		display: flex;
		gap: var(--space-2);
		overflow-x: auto;
		margin: var(--space-4) 0 var(--space-8);
		padding-bottom: var(--space-1);
	}
	.tabs a {
		white-space: nowrap;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		color: var(--muted);
		text-decoration: none;
	}
	.tabs a[aria-current='page'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
	}
</style>
