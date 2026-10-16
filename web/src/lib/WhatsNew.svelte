<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import type { IconName } from '$lib/icons';
	import Button from '$lib/ui/Button.svelte';
	import Icon from '$lib/ui/Icon.svelte';

	// What is new in Axon (EP-127): each recent feature one click away, until
	// the person closes the card; a later release shows it again.
	const key = 'axon-whats-new';
	const release = '2026-10-16';
	let shown = $state(false);
	try {
		shown = localStorage.getItem(key) !== release;
	} catch {
		shown = true;
	}
	function close() {
		shown = false;
		try {
			localStorage.setItem(key, release);
		} catch {
			// Private browsing: closed for this page only.
		}
	}
	const news: { href: string; icon: IconName; title: string; about: string }[] = [
		{
			href: '/health#life-web-title',
			icon: 'web',
			title: 'Life web',
			about: 'Food, water, sleep, activity, caffeine, and mood on one web, with a level to reach.'
		},
		{
			href: '/watch/podcast',
			icon: 'mic',
			title: 'Podcast with natural voices',
			about: 'Your tech watch read by one host, or two in conversation.'
		},
		{
			href: '/media',
			icon: 'media',
			title: 'Films, series, and books',
			about: 'Every cover, a page for each, a note from its template, and pages turned by a habit.'
		},
		{
			href: '/budget/reports',
			icon: 'pie',
			title: 'Budget charts',
			about: 'Where the money went and where it came from, month by month.'
		},
		{
			href: '/watch?kind=video',
			icon: 'play',
			title: 'YouTube videos',
			about:
				'Channels you follow, with their subtitles read and their links to the video and the channel.'
		},
		{
			href: '/watch?kind=podcast',
			icon: 'mic',
			title: 'Podcasts',
			about: 'Episodes with their notes, from feeds or your Audiobookshelf server.'
		},
		{
			href: '/collection',
			icon: 'trophy',
			title: 'Collection',
			about: 'Trophies, cards, and medals earned by your streaks and practice.'
		},
		{
			href: '/coffee',
			icon: 'coffee',
			title: 'Coffee and tea',
			about: 'Your caffeine against the daily limit, and the last coffee that keeps your sleep.'
		},
		{
			href: '/habits#protocols-title',
			icon: 'sun',
			title: 'Huberman protocols',
			about: 'Morning light, cold, sleep: habits with the studies behind them.'
		},
		{
			href: '/account#invites-title',
			icon: 'mail',
			title: 'Invite someone',
			about: 'A link to join Axon, or your household.'
		}
	];
</script>

{#if shown}
	<section class="card news" aria-labelledby="news-title">
		<div class="head">
			<h2 id="news-title">{t('New in Axon')}</h2>
			<Button variant="ghost" onclick={close}>{t('Close')}</Button>
		</div>
		<ul>
			{#each news as n (n.href)}
				<li>
					<a href={n.href}
						><span class="icon"><Icon name={n.icon} size={20} /></span>
						<strong>{t(n.title)}</strong>
						<span class="muted small">{t(n.about)}</span></a
					>
				</li>
			{/each}
		</ul>
	</section>
{/if}

<style>
	.news {
		margin-bottom: var(--space-4);
	}
	.head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-2);
	}
	ul {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(15rem, 1fr));
		gap: var(--space-2);
	}
	a {
		display: grid;
		grid-template-columns: auto 1fr;
		column-gap: var(--space-2);
		padding: var(--space-3);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--surface);
		color: var(--fg);
		text-decoration: none;
		height: 100%;
	}
	a:hover {
		border-color: var(--accent);
	}
	.icon {
		grid-row: span 2;
		color: var(--accent);
		padding-top: 0.1rem;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
