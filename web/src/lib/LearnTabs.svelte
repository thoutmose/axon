<script lang="ts">
	import { page } from '$app/state';
	import { t } from './i18n.svelte';

	const tabs = [
		{ href: '/roadmaps', label: 'Roadmaps' },
		{ href: '/cards', label: 'Flashcards' },
		{ href: '/roadmaps/assessment', label: 'Skills assessment' }
	];
	const current = (href: string) =>
		href === '/roadmaps' ? page.url.pathname === '/roadmaps' : page.url.pathname.startsWith(href);
</script>

<nav class="learn-tabs" aria-label={t('Learn')}>
	{#each tabs as tab (tab.href)}
		<a href={tab.href} aria-current={current(tab.href) ? 'page' : undefined}>{t(tab.label)}</a>
	{/each}
</nav>

<style>
	.learn-tabs {
		display: flex;
		gap: var(--space-1);
		margin: 0 0 var(--space-4);
		flex-wrap: wrap;
	}
	a {
		padding: var(--space-1) var(--space-3);
		border-radius: 999px;
		color: var(--muted);
		text-decoration: none;
		border: 1px solid transparent;
	}
	a[aria-current='page'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
		border-color: var(--border);
	}
</style>
