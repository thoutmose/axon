<script lang="ts">
	import '../app.css';
	import '$lib/spaces.css';
	import { spaceFor, spaceInfo } from '$lib/spaces';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import Toaster from '$lib/ui/Toaster.svelte';
	import { connection, watchConnection } from '$lib/connection.svelte';
	import { applyTheme, storedTheme, type Theme } from '$lib/theme';
	import { setLocale, storedLocale, t } from '$lib/i18n.svelte';
	import { applyAppearance, storedAppearance } from '$lib/preferences';
	import { api, type AuthStatus } from '$lib/api';
	import { avatar, avatarSrc } from '$lib/avatar.svelte';
	import { flush, loadOutbox, outbox } from '$lib/outbox.svelte';
	import { watchInstall } from '$lib/install.svelte';
	import { loadMenu, menu, storedHidden } from '$lib/menu.svelte';
	import { setZone, startSky } from '$lib/skyState.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let { children } = $props();
	// Pages with a full-bleed hero lay themselves out edge to edge.
	const wide = $derived(page.url.pathname === '/' || page.url.pathname === '/today');
	const current = (href: string) =>
		page.url.pathname === href || page.url.pathname.startsWith(href + '/');
	let theme = $state<Theme>('system');

	// Nineteen pages in five groups (EP-54): what you do, what you think,
	// how you grow, the rest of life, and reflecting. Today stays one click
	// away; the pages a person hid are left out, and so is an empty group (EP-94).
	type NavLink = { href: string; label: string; also?: string[] };
	const allGroups: { id: string; label: string; links: NavLink[] }[] = $derived([
		{
			id: 'do',
			label: t('Do'),
			links: [
				{ href: '/tasks', label: t('Tasks') },
				{ href: '/projects', label: t('Projects') },
				{ href: '/calendar', label: t('Calendar') },
				{ href: '/focus', label: t('Focus') }
			]
		},
		{
			id: 'think',
			label: t('Think'),
			links: [
				{ href: '/notes', label: t('Notes') },
				{ href: '/areas', label: t('Areas') },
				{ href: '/resources', label: t('Resources') },
				{ href: '/archive', label: t('Archive') }
			]
		},
		{
			id: 'grow',
			label: t('Grow'),
			links: [
				{ href: '/learn', label: t('Learn'), also: ['/roadmaps', '/cards'] },
				{ href: '/goals', label: t('Goals') },
				{ href: '/habits', label: t('Habits') }
			]
		},
		{
			id: 'life',
			label: t('Life'),
			links: [
				{ href: '/health', label: t('Health') },
				{ href: '/media', label: t('Media') },
				{ href: '/budget', label: t('Budget') },
				{ href: '/household', label: t('Household') }
			]
		},
		{
			id: 'reflect',
			label: t('Reflect'),
			links: [
				{ href: '/journal', label: t('Journal') },
				{ href: '/review', label: t('Weekly review') },
				{ href: '/insights', label: t('Insights') },
				{ href: '/agents', label: t('Agents') }
			]
		}
	]);
	const groups = $derived(
		allGroups
			.map((g) => ({ ...g, links: g.links.filter((l) => !menu.hidden.includes(l.href.slice(1))) }))
			.filter((g) => g.links.length > 0)
	);
	const inSection = (l: NavLink) => current(l.href) || (l.also ?? []).some(current);
	let openGroup = $state<string | null>(null);
	// Narrow screens get one Menu button and every group in a sheet.
	let sheet = $state(false);
	// A menu closes when a page opens, on Escape, and on a click elsewhere.
	$effect(() => {
		void page.url.pathname;
		openGroup = null;
		sheet = false;
	});
	function closeMenus(e: Event) {
		if (
			e instanceof KeyboardEvent ? e.key === 'Escape' : !(e.target as Element).closest?.('.group')
		)
			openGroup = null;
	}

	// The area's scenery, and a scroll offset for its parallax.
	const space = $derived(spaceFor(page.url.pathname));
	$effect(() => {
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
		menu.hidden = storedHidden();
		// The sky follows the sun in the account's time zone; so does the "sun" theme (EP-96).
		const stopSky = startSky(() => theme === 'sun' && applyTheme('sun'));
		api<AuthStatus>('/v1/auth/status')
			.then((s) => {
				avatar.signedIn = s.signed_in;
				if (!s.signed_in) return;
				loadMenu().catch(() => {});
				api<{ timezone: string }>('/v1/me')
					.then((m) => {
						setZone(m.timezone);
						if (theme === 'sun') applyTheme('sun');
					})
					.catch(() => {});
			})
			.catch(() => {});
		// Captures kept offline go out as soon as Axon answers again.
		loadOutbox();
		const send = () =>
			flush().then((n) => {
				if (n)
					toast(
						`${n} ${n === 1 ? 'capture' : 'captures'} kept offline ${n === 1 ? 'was' : 'were'} sent.`
					);
			});
		send();
		addEventListener('online', send);
		const stop = watchConnection();
		const stopInstall = watchInstall();
		return () => {
			removeEventListener('online', send);
			stopSky();
			stop();
			stopInstall();
		};
	});

	function cycleTheme() {
		theme = (
			{ system: 'sun', sun: 'dark', dark: 'light', light: 'system' } as const satisfies Record<
				Theme,
				Theme
			>
		)[theme];
		applyTheme(theme);
	}
</script>

<svelte:window onclick={closeMenus} onkeydown={closeMenus} />
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
			<button
				type="button"
				class="menu-toggle"
				aria-expanded={sheet}
				aria-controls="nav-sheet"
				onclick={() => (sheet = !sheet)}>{t('Menu')}</button
			>
			<div class="sheet glass" id="nav-sheet" hidden={!sheet}>
				{#each groups as g (g.id)}
					<section aria-labelledby="sheet-{g.id}">
						<h2 id="sheet-{g.id}">{t(g.label)}</h2>
						<ul>
							{#each g.links as l (l.href)}
								<li>
									<a href={l.href} aria-current={inSection(l) ? 'page' : undefined}>{t(l.label)}</a>
								</li>
							{/each}
						</ul>
					</section>
				{/each}
			</div>
			<ul class="groups">
				{#each groups as g (g.id)}
					{@const here = g.links.some((l) => inSection(l))}
					<li class="group">
						<button
							type="button"
							class:here
							aria-expanded={openGroup === g.id}
							aria-controls="nav-{g.id}"
							onclick={() => (openGroup = openGroup === g.id ? null : g.id)}
							>{t(g.label)}{#if here}<span class="visually-hidden">
									({t('current section')})</span
								>{/if}<span class="caret" aria-hidden="true">▾</span></button
						>
						<ul class="menu glass" id="nav-{g.id}" hidden={openGroup !== g.id}>
							{#each g.links as l (l.href)}
								<li>
									<a href={l.href} aria-current={inSection(l) ? 'page' : undefined}>{t(l.label)}</a>
								</li>
							{/each}
						</ul>
					</li>
				{/each}
			</ul>
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
				aria-label={t('Theme: {theme}. Change theme', {
					theme: theme === 'sun' ? t('follows the sun') : t(theme)
				})}
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
					{#if theme === 'sun'}
						<path
							d="M3 17h18M6 17a6 6 0 0 1 12 0M12 5v2M4.9 9.9l1.4 1.4M19.1 9.9l-1.4 1.4M7 21h10"
						/>
					{:else if theme === 'dark'}
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
	<p class="banner" role="alert">
		{t(
			'You are offline. Captures (press n) are kept on this device and sent when you reconnect; other changes are not saved.'
		) + (outbox.items.length ? ' ' + t('{n} waiting.', { n: outbox.items.length }) : '')}
	</p>
{:else if connection.apiDown}
	<p class="banner" role="alert">
		{t("Axon's server is not responding. Your data is safe; try again shortly.")}
	</p>
{/if}

<main id="main" class:wide>
	{@render children()}
	{#if space}
		<!-- What the page's picture shows, and what it nods to (EP-98). -->
		<p class="space-credit">
			<strong>{t(spaceInfo[space].title)}.</strong>
			{t(spaceInfo[space].about)}
		</p>
	{/if}
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
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: var(--space-1);
		min-width: 0;
	}
	.groups {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.group {
		position: relative;
	}
	.group > button {
		font: inherit;
		font-size: var(--text-sm);
		color: var(--fg);
		opacity: 0.8;
		background: none;
		border: none;
		border-radius: 999px;
		padding: var(--space-1) var(--space-3);
		cursor: pointer;
		white-space: nowrap;
	}
	.group > button:hover,
	.group > button[aria-expanded='true'] {
		opacity: 1;
		background: var(--accent-soft);
	}
	.group > button.here {
		opacity: 1;
		font-weight: 600;
	}
	.caret {
		margin-left: var(--space-1);
		font-size: 0.75em;
	}
	.menu {
		position: absolute;
		top: calc(100% + var(--space-1));
		left: 0;
		z-index: 30;
		min-width: 11rem;
		list-style: none;
		margin: 0;
		padding: var(--space-1);
		display: grid;
		gap: 2px;
		background: var(--surface-solid);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow, 0 8px 24px rgb(0 0 0 / 0.18));
	}
	.menu[hidden] {
		display: none;
	}
	.menu-toggle,
	.sheet {
		display: none;
	}
	@media (max-width: 760px) {
		.bar {
			gap: var(--space-2);
		}
		nav {
			flex-wrap: nowrap;
		}
		.groups {
			display: none;
		}
		.menu-toggle {
			display: inline-block;
			font: inherit;
			font-size: var(--text-sm);
			color: var(--fg);
			background: none;
			border: 1px solid var(--border);
			border-radius: 999px;
			padding: var(--space-1) var(--space-3);
			cursor: pointer;
		}
		.sheet:not([hidden]) {
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: var(--space-3);
			position: absolute;
			top: 100%;
			left: 0;
			right: 0;
			padding: var(--space-4) var(--gutter);
			background: var(--surface-solid);
			border-bottom: 1px solid var(--border);
			box-shadow: var(--shadow, 0 8px 24px rgb(0 0 0 / 0.18));
			max-height: calc(100vh - 4rem);
			overflow-y: auto;
		}
		.sheet h2 {
			font-size: var(--text-sm);
			text-transform: uppercase;
			letter-spacing: 0.06em;
			color: var(--muted);
			margin: 0 0 var(--space-1);
		}
		.sheet ul {
			list-style: none;
			margin: 0;
			padding: 0;
		}
		.sheet a {
			display: block;
			padding: var(--space-2) var(--space-3);
		}
	}
	.menu a {
		display: block;
		border-radius: var(--radius-sm);
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
	.space-credit {
		max-width: var(--content-width, 60rem);
		margin: var(--space-12) auto var(--space-6);
		padding: 0 var(--gutter);
		font-size: var(--text-sm);
		color: var(--muted);
	}
	:global(:root[data-spaces='off']) .space-credit {
		display: none;
	}
</style>
