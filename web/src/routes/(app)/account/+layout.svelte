<script lang="ts">
	import { page } from '$app/state';
	import { t } from '$lib/i18n.svelte';

	let { children } = $props();
	const tabs = $derived([
		{ href: '/account', label: t('Overview') },
		{ href: '/account/security', label: t('Sign-in & security') },
		{ href: '/account/tokens', label: t('Access tokens') },
		{ href: '/account/privacy', label: t('Privacy & data') },
		{ href: '/account/import', label: t('Import') },
		{ href: '/account/trash', label: t('Trash') },
		{ href: '/account/preferences', label: t('Preferences') },
		{ href: '/account/activity', label: t('Activity') }
	]);
</script>

<h1>{t('Account')}</h1>
<nav aria-label={t('Account sections')} class="tabs">
	{#each tabs as tab (tab.href)}
		<a href={tab.href} aria-current={page.url.pathname === tab.href ? 'page' : undefined}
			>{t(tab.label)}</a
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
