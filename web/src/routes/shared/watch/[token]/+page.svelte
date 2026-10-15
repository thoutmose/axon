<script lang="ts">
	import { page } from '$app/state';
	import { api, message } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { exportFormats } from '$lib/watch';

	// A shared selection of news (EP-112), for anyone with the link: what
	// the sources published, and where it comes from.
	type Shared = {
		name: string;
		description: string;
		count: number;
		generated_at: string;
		expires_at: string;
		attribution: string;
		items: {
			title: string;
			url: string;
			source_title: string;
			author: string;
			published_at: string;
			summary: string;
			kind: 'article' | 'video' | 'podcast';
			labels: string[];
			reading_minutes: number;
		}[];
	};
	const token = page.params.token ?? '';
	let data = $state<Shared | null>(null);
	let error = $state('');
	api<Shared>(`/v1/shared/watch/${encodeURIComponent(token)}`)
		.then((d) => (data = d))
		.catch((err) => (error = message(err)));
	const day = (iso: string) =>
		new Date(iso).toLocaleDateString(undefined, {
			day: 'numeric',
			month: 'short',
			year: 'numeric'
		});
	const kinds = { article: 'Article', video: 'Video', podcast: 'Podcast' };
	const meta = (it: Shared['items'][number]) =>
		[
			it.source_title,
			it.author,
			day(it.published_at),
			t(kinds[it.kind]),
			it.kind === 'article' && it.reading_minutes ? t('{n} min', { n: it.reading_minutes }) : ''
		]
			.filter(Boolean)
			.join(' · ');
</script>

<svelte:head>
	<title>{data?.name ?? t('Shared news')} · Axon</title>
	<!-- The address holds the link's secret: never send it to the sites it links to. -->
	<meta name="referrer" content="no-referrer" />
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

{#if error}
	<Status kind="error" message={error} />
{:else if !data}
	<Status kind="loading" message={t('Loading the shared news…')} />
{:else}
	<header class="head">
		<p class="badge">
			<a href="https://github.com/thoutmose/axon" rel="noreferrer noopener"
				>{t('Shared from Axon Project')}</a
			>
		</p>
		<h1>{data.name}</h1>
		<p class="muted">
			{data.description} · {t('{n} news', { n: data.count })} · {t('link valid until {date}', {
				date: day(data.expires_at)
			})}
		</p>
		<ul class="formats" aria-label={t('Download')}>
			{#each exportFormats as [format, label] (format)}
				<li>
					<a href="/api/v1/shared/watch/{encodeURIComponent(token)}?format={format}" download
						>{t(label)}</a
					>
				</li>
			{/each}
		</ul>
	</header>

	{#if data.items.length}
		<ol class="items">
			{#each data.items as it (it.url)}
				<li>
					<a class="title" href={it.url} rel="noreferrer noopener nofollow" target="_blank"
						>{it.title}</a
					>
					<p class="small muted">{meta(it)}</p>
					{#if it.summary}<p>{it.summary}</p>{/if}
					{#if it.labels.length}
						<p class="labels">
							{#each it.labels as l (l)}<span class="chip">{l}</span>{/each}
						</p>
					{/if}
				</li>
			{/each}
		</ol>
		{#if data.count > data.items.length}
			<p class="muted small">
				{t('The newest {shown} of {n}; the downloads hold them all.', {
					shown: data.items.length,
					n: data.count
				})}
			</p>
		{/if}
	{:else}
		<Status kind="empty" message={t('Nothing in this selection yet.')} />
	{/if}
	<footer class="small muted">{data.attribution} · {day(data.generated_at)}</footer>
{/if}

<style>
	.small {
		font-size: var(--text-sm);
	}
	.head h1 {
		margin: var(--space-2) 0;
	}
	.badge a {
		display: inline-block;
		font-size: var(--text-sm);
		border: 1px solid var(--border);
		border-radius: 999px;
		padding: 0 var(--space-3);
		color: var(--fg);
		text-decoration: none;
		background: var(--accent-soft);
	}
	.formats {
		list-style: none;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.formats a {
		display: inline-block;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-1) var(--space-3);
		text-decoration: none;
		color: var(--fg);
		font-size: var(--text-sm);
	}
	.items {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-3);
		max-width: 48rem;
	}
	.items li {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-3) var(--space-4);
	}
	.items p {
		margin: var(--space-1) 0 0;
	}
	.title {
		font-weight: 600;
		font-size: var(--text-lg);
	}
	.labels {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
	}
	.chip {
		font-size: var(--text-sm);
		border: 1px solid var(--border);
		border-radius: 999px;
		padding: 0 var(--space-2);
	}
	footer {
		margin: var(--space-6) 0;
	}
</style>
