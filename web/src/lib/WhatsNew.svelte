<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';

	// What is new in Axon (EP-127): each recent feature one click away, until
	// the person closes the card; a later release shows it again.
	const key = 'axon-whats-new';
	const release = '2026-10';
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
	const news = [
		{
			href: '/watch?kind=video',
			icon: '▶',
			title: 'YouTube videos',
			about:
				'Channels you follow, with their subtitles read and their links to the video and the channel.'
		},
		{
			href: '/watch?kind=podcast',
			icon: '🎧',
			title: 'Podcasts',
			about: 'Episodes with their notes, from feeds or your Audiobookshelf server.'
		},
		{
			href: '/collection',
			icon: '🏆',
			title: 'Collection',
			about: 'Trophies, cards, and medals earned by your streaks and practice.'
		},
		{
			href: '/coffee',
			icon: '☕',
			title: 'Coffee and tea',
			about: 'Your caffeine against the daily limit, and the last coffee that keeps your sleep.'
		},
		{
			href: '/habits#protocols-title',
			icon: '🌅',
			title: 'Huberman protocols',
			about: 'Morning light, cold, sleep: habits with the studies behind them.'
		},
		{
			href: '/account#invites-title',
			icon: '✉',
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
						><span class="icon" aria-hidden="true">{n.icon}</span>
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
		font-size: 1.5rem;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
