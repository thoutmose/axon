<script lang="ts">
	import { appLocale } from '$lib/dates';
	import { api, message } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import {
		ago,
		describeSchedule,
		type WatchSource,
		type WatchStats,
		type WatchTopic
	} from '$lib/watch';
	import FollowSource from './FollowSource.svelte';
	import ImportSources from './ImportSources.svelte';
	import SharesSection from './SharesSection.svelte';
	import StatsPanel from './StatsPanel.svelte';
	import TopicsSection from './TopicsSection.svelte';
	import { presets } from './presets';

	// Sources and health (EP-110): what you follow, how each is doing, and
	// how to add more (by address, CSV, OPML, or from FreshRSS).
	let sources = $state<WatchSource[]>([]);
	let topics = $state<WatchTopic[]>([]);
	let stats = $state<WatchStats | null>(null);
	let error = $state('');

	async function load() {
		try {
			[{ items: sources }, { items: topics }, stats] = await Promise.all([
				api<{ items: WatchSource[] }>('/v1/watch/sources'),
				api<{ items: WatchTopic[] }>('/v1/watch/topics'),
				api<WatchStats>('/v1/watch/stats')
			]);
		} catch (err) {
			error = message(err);
		}
	}
	load();

	// --- sources ---
	async function patch(s: WatchSource, body: Record<string, unknown>) {
		try {
			await api(`/v1/watch/sources/${s.id}`, { method: 'PATCH', body });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	async function fetchNow(s: WatchSource) {
		try {
			const r = await api<{ stats: { new: number }; error: string }>(
				`/v1/watch/sources/${s.id}/fetch`,
				{ method: 'POST' }
			);
			if (r.error) toast(r.error, 'error');
			else toast(t('{n} new from {name}.', { n: r.stats.new, name: s.title }));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	async function unfollow(s: WatchSource) {
		const withItems = confirm(
			t('Stop following {name}. Also delete its news you did not save? (Cancel keeps them.)', {
				name: s.title
			})
		);
		try {
			await api(`/v1/watch/sources/${s.id}?items=${withItems ? 'delete' : 'keep'}`, {
				method: 'DELETE'
			});
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	const health: Record<string, string> = {
		ok: '● OK',
		failing: '▲ Failing',
		waiting: '◌ Waiting',
		off: '○ Off'
	};
</script>

<svelte:head><title>{t('Sources and health')} · Axon</title></svelte:head>

<p><a href="/watch">← {t('Tech watch')}</a></p>
<h1>{t('Sources and health')}</h1>
<p class="muted">
	{t('Podcasts you download with Audiobookshelf can follow here too:')}
	<a href="/account/preferences#audiobookshelf">{t('Podcasts from Audiobookshelf')}</a>
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if stats}
	<StatsPanel {stats} />
{/if}

<FollowSource {topics} onchange={load} />

<ImportSources onchange={load} />

<TopicsSection {topics} onchange={load} />

<Section id="sources" title={t('Sources')}>
	<p class="row">
		<a href="/api/v1/watch/sources.opml" download>{t('Export as OPML')}</a>
	</p>
	<div class="table">
		<table>
			<thead>
				<tr>
					<th>{t('Source')}</th>
					<th>{t('Health')}</th>
					<th>{t('Last fetch')}</th>
					<th>{t('Schedule')}</th>
					<th>{t('News')}</th>
					<th><span class="sr-only">{t('Actions')}</span></th>
				</tr>
			</thead>
			<tbody>
				{#each sources as s (s.id)}
					<tr>
						<td>
							<strong>{s.title}</strong>
							<div class="muted small">{s.url}</div>
							{#if s.topic_ids.length}<div class="small">
									{s.topic_ids.map((id) => topics.find((x) => x.id === id)?.name).join(', ')}
								</div>{/if}
						</td>
						<td>
							<span class="health {s.health}">{t(health[s.health])}</span>
							{#if s.last_error}<div class="small bad">{s.last_error}</div>{/if}
						</td>
						<td class="small">
							{s.last_fetched_at ? ago(s.last_fetched_at) : '—'}
							<div class="muted">
								{t('next')}
								{new Date(s.next_fetch_at).toLocaleString(appLocale())}
							</div>
						</td>
						<td>
							<select
								aria-label={t('Schedule of {name}', { name: s.title })}
								value={presets.some(([v]) => v === s.schedule) ? s.schedule : 'custom'}
								onchange={(e) => {
									const v = e.currentTarget.value;
									if (v !== 'custom') patch(s, { schedule: v });
								}}
							>
								{#each presets as [v, l] (v)}<option value={v}>{t(l)}</option>{/each}
								<option value="custom" disabled>{describeSchedule(s.schedule)}</option>
							</select>
							<label class="check small"
								><input
									type="checkbox"
									checked={s.enabled}
									onchange={() => patch(s, { enabled: !s.enabled })}
								/>{t('On')}</label
							>
							<label class="check small"
								><input
									type="checkbox"
									checked={s.full_text}
									onchange={() => patch(s, { full_text: !s.full_text })}
								/>{t('Full text')}</label
							>
							<label class="check small"
								><input
									type="checkbox"
									checked={s.news}
									onchange={() => patch(s, { news: !s.news })}
								/>{t('News')}</label
							>
						</td>
						<td class="small">
							<a href="/watch?source={s.id}">{s.items}</a>
							<div class="muted">{s.items_week} {t('this week')} · {s.merged} {t('merged')}</div>
						</td>
						<td>
							<Button variant="ghost" onclick={() => fetchNow(s)}>{t('Fetch now')}</Button>
							<Button variant="ghost" onclick={() => unfollow(s)}>{t('Unfollow')}</Button>
						</td>
					</tr>
				{:else}
					<tr><td colspan="6" class="muted">{t('No source yet: follow one above.')}</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
</Section>

<SharesSection />

<style>
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
	}
	.check {
		display: flex;
		gap: var(--space-1);
		align-items: baseline;
	}
	.table {
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	th,
	td {
		text-align: left;
		vertical-align: top;
		padding: var(--space-2);
		border-bottom: 1px solid var(--border);
	}
	td div {
		overflow-wrap: anywhere;
	}
	.health.ok {
		color: var(--ok);
	}
	.health.failing {
		color: var(--bad);
	}
	.health.waiting {
		color: var(--warn);
	}
	.small {
		font-size: var(--text-sm);
	}
	.bad {
		color: var(--bad);
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
		white-space: nowrap;
	}
</style>
