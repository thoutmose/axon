<script lang="ts">
	import '../app.css';
	import '$lib/spaces.css';
	import { spaceFor } from '$lib/spaces';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import Toaster from '$lib/ui/Toaster.svelte';
	import { connection, watchConnection } from '$lib/connection.svelte';
	import { applyTheme, storedTheme, type Theme } from '$lib/theme';
	import { setLocale, storedLocale, t } from '$lib/i18n.svelte';
	import { applyAppearance, storedAppearance } from '$lib/preferences';
	import { api, type AuthStatus } from '$lib/api';
	import { avatar, avatarSrc } from '$lib/avatar.svelte';

	let { children } = $props();
	// Pages with a full-bleed hero lay themselves out edge to edge.
	const wide = $derived(page.url.pathname === '/' || page.url.pathname === '/today');
	const current = (href: string) =>
		page.url.pathname === href || page.url.pathname.startsWith(href + '/');
	let theme = $state<Theme>('system');

	// The area's scenery, and a scroll offset for its parallax.
	$effect(() => {
		const space = spaceFor(page.url.pathname);
		if (space) document.documentElement.dataset.space = space;
		else delete document.documentElement.dataset.space;
	});
	onMount(() => {
		let frame = 0;
		const onScroll = () => {
			cancelAnimationFrame(frame);
			frame = requestAnimationFrame(() =>
				document.documentElement.style.setProperty('--scroll', String(Math.round(window.scrollY)))
			);
		};
		window.addEventListener('scroll', onScroll, { passive: true });
		return () => window.removeEventListener('scroll', onScroll);
	});

	onMount(() => {
		const saved = storedAppearance();
		if (saved) applyAppearance({ ...saved, theme: storedTheme() });
		theme = storedTheme();
		applyTheme(theme);
		setLocale(storedLocale());
		api<AuthStatus>('/v1/auth/status')
			.then((s) => (avatar.signedIn = s.signed_in))
			.catch(() => {});
		return watchConnection();
	});

	function cycleTheme() {
		theme = theme === 'system' ? 'dark' : theme === 'dark' ? 'light' : 'system';
		applyTheme(theme);
	}
</script>

<div class="space-art" aria-hidden="true"></div>
<a class="skip" href="#main">{t('Skip to content')}</a>
<header>
	<div class="bar">
		<a href="/" class="brand"
			><svg viewBox="0 0 24 24" width="22" height="22" aria-hidden="true"
				><defs
					><linearGradient id="axon-mark" x1="0" y1="0" x2="1" y2="1"
						><stop offset="0" stop-color="#5aa9ff" /><stop
							offset="1"
							stop-color="#b16cff"
						/></linearGradient
					></defs
				><circle cx="12" cy="12" r="10" fill="url(#axon-mark)" /><circle
					cx="8"
					cy="10"
					r="2"
					fill="#fff"
				/><circle cx="16" cy="9" r="1.6" fill="#fff" /><circle
					cx="13"
					cy="16"
					r="1.8"
					fill="#fff"
				/><path
					d="M8 10 16 9M8 10l5 6M16 9l-3 7"
					stroke="#fff"
					stroke-width="1.2"
					fill="none"
					opacity=".8"
				/></svg
			>Axon</a
		>
		<nav aria-label={t('Main')}>
			<a href="/today" aria-current={current('/today') ? 'page' : undefined}>{t('Today')}</a>
			<a href="/tasks" aria-current={current('/tasks') ? 'page' : undefined}>{t('Tasks')}</a>
			<a href="/notes" aria-current={current('/notes') ? 'page' : undefined}>{t('Notes')}</a>
			<a href="/goals" aria-current={current('/goals') ? 'page' : undefined}>{t('Goals')}</a>
			<a href="/projects" aria-current={current('/projects') ? 'page' : undefined}
				>{t('Projects')}</a
			>
			<a href="/areas" aria-current={current('/areas') ? 'page' : undefined}>{t('Areas')}</a>
			<a href="/resources" aria-current={current('/resources') ? 'page' : undefined}
				>{t('Resources')}</a
			>
			<a href="/archive" aria-current={current('/archive') ? 'page' : undefined}>{t('Archive')}</a>
			<a
				href="/roadmaps"
				aria-current={current('/roadmaps') || current('/cards') ? 'page' : undefined}
				>{t('Learn')}</a
			>
			<a href="/habits" aria-current={current('/habits') ? 'page' : undefined}>{t('Habits')}</a>
			<a href="/focus" aria-current={current('/focus') ? 'page' : undefined}>{t('Focus')}</a>
			<a href="/calendar" aria-current={current('/calendar') ? 'page' : undefined}
				>{t('Calendar')}</a
			>
			<a href="/health" aria-current={current('/health') ? 'page' : undefined}>{t('Health')}</a>
			<a href="/media" aria-current={current('/media') ? 'page' : undefined}>{t('Media')}</a>
			<a href="/budget" aria-current={current('/budget') ? 'page' : undefined}>{t('Budget')}</a>
			<a href="/agents" aria-current={current('/agents') ? 'page' : undefined}>{t('Agents')}</a>
			<a href="/account" class="account" aria-current={current('/account') ? 'page' : undefined}
				>{#if avatar.signedIn}<img
						src={avatarSrc(avatar.version)}
						alt=""
						width="22"
						height="22"
					/>{/if}{t('Account')}</a
			>
			<button
				class="theme"
				onclick={cycleTheme}
				aria-label={t('Theme: {theme}. Change theme', { theme: t(theme) })}
			>
				<svg
					viewBox="0 0 24 24"
					width="18"
					height="18"
					aria-hidden="true"
					fill="none"
					stroke="currentColor"
					stroke-width="1.8"
					stroke-linecap="round"
				>
					{#if theme === 'dark'}
						<path d="M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5Z" />
					{:else if theme === 'light'}
						<circle cx="12" cy="12" r="4" /><path
							d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"
						/>
					{:else}
						<circle cx="12" cy="12" r="8.5" /><path
							d="M12 3.5v17a8.5 8.5 0 0 0 0-17Z"
							fill="currentColor"
						/>
					{/if}
				</svg>
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

<main id="main" class:wide>
	{@render children()}
</main>
<Toaster />

<style>
	header {
		position: sticky;
		top: 0;
		z-index: 10;
		background: var(--surface-glass);
		-webkit-backdrop-filter: var(--glass-blur);
		backdrop-filter: var(--glass-blur);
		border-bottom: 1px solid var(--border);
	}
	.bar {
		padding: var(--space-2) var(--gutter);
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-6);
		min-height: 3rem;
	}
	.brand {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		font-weight: 650;
		font-size: var(--text-lg);
		color: var(--fg);
		text-decoration: none;
		letter-spacing: -0.02em;
		flex-shrink: 0;
	}
	nav {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		min-width: 0;
		overflow-x: auto; /* narrow screens scroll the menu, never the page */
		scrollbar-width: none;
	}
	nav a {
		white-space: nowrap;
		color: var(--fg);
		opacity: 0.8;
		text-decoration: none;
		font-size: var(--text-sm);
		padding: var(--space-1) var(--space-3);
		border-radius: 999px;
		transition:
			opacity var(--motion-fast) var(--ease),
			background var(--motion-fast) var(--ease);
	}
	nav a:hover {
		opacity: 1;
	}
	nav a[aria-current='page'] {
		opacity: 1;
		font-weight: 600;
		background: var(--accent-soft);
	}
	.theme {
		display: inline-grid;
		place-items: center;
		background: none;
		border: none;
		border-radius: 999px;
		color: var(--fg);
		min-width: 2.25rem;
		min-height: 2.25rem;
		cursor: pointer;
		font-size: var(--text-base);
		flex-shrink: 0;
	}
	.theme:hover {
		background: var(--accent-soft);
	}
	main {
		padding: var(--space-8) var(--gutter) var(--space-16);
	}
	main.wide {
		padding: 0 0 var(--space-16);
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
		background: var(--surface-solid);
		padding: var(--space-2) var(--space-3);
	}
	.account {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
	}
	.account img {
		border-radius: 50%;
	}
</style>
