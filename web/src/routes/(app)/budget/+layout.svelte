<script lang="ts">
	import { page } from '$app/state';
	import '$lib/finance.css';

	let { children } = $props();
	const tabs = [
		{ href: '/budget', label: 'Overview' },
		{ href: '/budget/accounts', label: 'Accounts' },
		{ href: '/budget/pots', label: 'Savings pots' },
		{ href: '/budget/recurring', label: 'Recurring' },
		{ href: '/budget/rules', label: 'Rules' },
		{ href: '/budget/import', label: 'Import' },
		{ href: '/budget/reports', label: 'Reports' }
	];
</script>

<div class="finance">
	<nav aria-label="Finance" class="subnav">
		{#each tabs as t (t.href)}
			<a href={t.href} aria-current={page.url.pathname === t.href ? 'page' : undefined}>{t.label}</a
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
