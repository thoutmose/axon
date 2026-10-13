<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { page } from '$app/state';
	import '$lib/finance.css';

	let { children } = $props();
	const tabs = $derived([
		{ href: '/budget', label: t('Overview') },
		{ href: '/budget/accounts', label: t('Accounts') },
		{ href: '/budget/pots', label: t('Savings pots') },
		{ href: '/budget/recurring', label: t('Recurring') },
		{ href: '/budget/rules', label: t('Rules') },
		{ href: '/budget/import', label: t('Import') },
		{ href: '/budget/reports', label: t('Reports') }
	]);
</script>

<div class="finance">
	<nav aria-label={t('Finance')} class="subnav">
		{#each tabs as tab (tab.href)}
			<a href={tab.href} aria-current={page.url.pathname === tab.href ? 'page' : undefined}
				>{tab.label}</a
			>
		{/each}
	</nav>
	{@render children()}
</div>

<style>
	.subnav {
		display: flex;
		gap: var(--space-1);
		overflow-x: auto;
		margin-bottom: var(--space-4);
	}
	.subnav a {
		white-space: nowrap;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		color: var(--muted);
		text-decoration: none;
	}
	.subnav a[aria-current='page'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
	}
</style>
