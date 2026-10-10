<script lang="ts">
	import { page } from '$app/state';

	let { children } = $props();
	const tabs = [
		{ href: '/account', label: 'Overview' },
		{ href: '/account/security', label: 'Sign-in & security' },
		{ href: '/account/tokens', label: 'Access tokens' },
		{ href: '/account/privacy', label: 'Privacy & data' },
		{ href: '/account/activity', label: 'Activity' }
	];
</script>

<h1>Account</h1>
<nav aria-label="Account sections" class="tabs">
	{#each tabs as t (t.href)}
		<a href={t.href} aria-current={page.url.pathname === t.href ? 'page' : undefined}>{t.label}</a>
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
