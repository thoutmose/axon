<script lang="ts">
	import { api, message } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import type { WatchSource } from '$lib/watch';

	// YouTube and Podcasts as places of their own (EP-144): what you follow
	// there, a field to follow one more, and channels and shows that go with
	// the roadmap, each one click away. Follows are refused to the demo.
	let { kind, onfollowed }: { kind: 'video' | 'podcast'; onfollowed: () => void } = $props();

	type Pick = { name: string; url: string; about: string };
	const youtube: Pick[] = [
		{
			name: 'Andrej Karpathy',
			url: 'https://www.youtube.com/@AndrejKarpathy',
			about: 'Neural networks from scratch, GPT'
		},
		{
			name: '3Blue1Brown',
			url: 'https://www.youtube.com/@3blue1brown',
			about: 'Linear algebra, calculus, neural networks'
		},
		{
			name: 'MIT OpenCourseWare',
			url: 'https://www.youtube.com/@mitocw',
			about: '6.006, 6.042J, 18.06: the roadmap’s lectures'
		},
		{
			name: 'CMU Database Group',
			url: 'https://www.youtube.com/@CMUDatabaseGroup',
			about: '15-445 and database internals'
		},
		{
			name: 'StatQuest with Josh Starmer',
			url: 'https://www.youtube.com/@statquest',
			about: 'Statistics and machine learning'
		},
		{ name: 'ByteByteGo', url: 'https://www.youtube.com/@ByteByteGo', about: 'System design' },
		{ name: 'Confluent', url: 'https://www.youtube.com/@Confluent', about: 'Kafka and streaming' },
		{
			name: 'Databricks',
			url: 'https://www.youtube.com/@Databricks',
			about: 'Spark and the lakehouse'
		},
		{ name: 'CNCF', url: 'https://www.youtube.com/@cncf', about: 'Kubernetes and cloud native' },
		{ name: 'Jeremy’s IT Lab', url: 'https://www.youtube.com/@JeremysITLab', about: 'CCNA, free' },
		{
			name: 'LiveOverflow',
			url: 'https://www.youtube.com/@LiveOverflow',
			about: 'Security and exploitation'
		},
		{
			name: 'Computerphile',
			url: 'https://www.youtube.com/@Computerphile',
			about: 'Computer science, explained'
		}
	];
	const podcasts: Pick[] = [
		{
			name: 'Data Engineering Podcast',
			url: 'https://serve.podhome.fm/rss/1c0357c0-6aba-5766-a2d5-2090d8dab6bc',
			about: 'Data engineering and DataOps'
		},
		{
			name: 'Software Engineering Radio',
			url: 'https://seradio.libsyn.com/rss',
			about: 'Software engineering, IEEE'
		},
		{
			name: 'Kubernetes Podcast from Google',
			url: 'https://kubernetespodcast.com/feeds/audio.xml',
			about: 'Kubernetes'
		},
		{ name: 'Talk Python To Me', url: 'https://talkpython.fm/episodes/rss', about: 'Python' },
		{
			name: 'The Changelog',
			url: 'https://changelog.com/podcast/feed',
			about: 'Open source and software'
		},
		{
			name: 'Practical AI',
			url: 'https://feeds.transistor.fm/practical-ai-machine-learning-data-science-llm',
			about: 'Machine learning in practice'
		},
		{
			name: 'Darknet Diaries',
			url: 'https://feeds.megaphone.fm/darknetdiaries',
			about: 'Security stories'
		},
		{
			name: 'The Stack Overflow Podcast',
			url: 'https://feeds.simplecast.com/XA_851k3',
			about: 'Software development'
		},
		{
			name: 'Lex Fridman Podcast',
			url: 'https://lexfridman.com/feed/podcast/',
			about: 'AI and science, long interviews'
		}
	];
	const picks = $derived(kind === 'video' ? youtube : podcasts);

	let sources = $state<WatchSource[]>([]);
	let address = $state('');
	let busy = $state('');
	async function load() {
		try {
			sources = (await api<{ items: WatchSource[] }>('/v1/watch/sources')).items;
		} catch {
			sources = [];
		}
	}
	load();

	const isYouTube = (s: WatchSource) => /youtube\.com/.test(s.url);
	// Podcasts are the feeds that are not YouTube channels and carry episodes:
	// Audiobookshelf's, and the shows followed from here or by their feed.
	const mine = $derived(
		sources.filter((s) =>
			kind === 'video'
				? isYouTube(s)
				: s.origin === 'audiobookshelf' || podcasts.some((p) => p.url === s.url)
		)
	);
	const followed = (p: Pick) =>
		sources.some((s) => s.url === p.url || s.title.toLowerCase() === p.name.toLowerCase());

	async function follow(url: string) {
		busy = url;
		try {
			const r = await api<{ source: WatchSource; stats: { new: number } }>('/v1/watch/sources', {
				method: 'POST',
				body: { url, topic_ids: [], schedule: '0 0 * * *', full_text: false }
			});
			toast(t('Following {name}: {n} new.', { name: r.source.title, n: r.stats.new }));
			address = '';
			await load();
			onfollowed();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = '';
		}
	}
</script>

<section class="follow" aria-labelledby="follow-{kind}">
	<h2 id="follow-{kind}">
		{kind === 'video' ? t('YouTube channels you follow') : t('Podcasts you follow')}
		<span class="muted">{mine.length}</span>
	</h2>
	{#if mine.length}
		<p class="small">
			{#each mine as s, i (s.id)}{i ? ' · ' : ''}<a href="/watch?source={s.id}&status=">{s.title}</a
				>{/each}
		</p>
	{/if}
	<form
		class="row"
		onsubmit={(e) => {
			e.preventDefault();
			follow(address);
		}}
	>
		<label class="visually-hidden" for="follow-address-{kind}"
			>{kind === 'video' ? t('A YouTube channel') : t('A podcast feed or its site')}</label
		>
		<input
			id="follow-address-{kind}"
			bind:value={address}
			required
			placeholder={kind === 'video'
				? 'https://www.youtube.com/@channel'
				: 'https://example.com/podcast/feed.xml'}
		/>
		<Button type="submit" busy={busy === address && !!address}>{t('Follow')}</Button>
	</form>
	<p class="muted small">
		{kind === 'video'
			? t(
					'Axon reads each new video through its captions (or Whisper, when the server runs one), labels it, and adds it to the podcast and the digest.'
				)
			: t('Episodes come with their notes; with an Audiobookshelf server, their words too.')}
		{#if kind === 'podcast'}<a href="/account/preferences#audiobookshelf"
				>{t('Connect Audiobookshelf')}</a
			>{/if}
		· <a href="/watch/sources">{t('Sources and health')}</a>
	</p>

	<details open={mine.length === 0}>
		<summary>{t('For the roadmap')}</summary>
		<ul class="picks">
			{#each picks as p (p.url)}
				<li>
					<span class="grow"
						><strong>{p.name}</strong> <span class="muted small">{p.about}</span></span
					>
					{#if followed(p)}
						<span class="small ok">{t('Following')}</span>
					{:else}
						<Button variant="secondary" busy={busy === p.url} onclick={() => follow(p.url)}
							>{t('Follow')}</Button
						>
					{/if}
				</li>
			{/each}
		</ul>
	</details>
</section>

<style>
	.follow {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin: var(--space-3) 0;
	}
	h2 {
		margin: 0 0 var(--space-2);
		font-size: var(--text-lg);
	}
	summary {
		cursor: pointer;
		font-weight: 600;
		margin: var(--space-3) 0 var(--space-2);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.row input {
		flex: 1 1 18rem;
	}
	.picks {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 18rem), 1fr));
		gap: var(--space-2);
	}
	.picks li {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-2) var(--space-3);
	}
	.grow {
		flex: 1;
	}
	.ok {
		color: var(--ok);
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
