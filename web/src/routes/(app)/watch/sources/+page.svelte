<script lang="ts">
	import { api, message } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import {
		ago,
		csvHeader,
		describeSchedule,
		type WatchShare,
		type WatchSource,
		type WatchStats,
		type WatchTopic
	} from '$lib/watch';

	// Sources and health (EP-73): what you follow, how each is doing, and
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

	// --- share links (EP-75) ---
	let shares = $state<WatchShare[]>([]);
	async function loadShares() {
		try {
			({ items: shares } = await api<{ items: WatchShare[] }>('/v1/watch/shares'));
		} catch {
			// The page works without them.
		}
	}
	loadShares();
	async function revoke(s: WatchShare) {
		if (!confirm(t('Revoke “{name}”? The link stops working at once.', { name: s.name }))) return;
		try {
			await api(`/v1/watch/shares/${s.id}`, { method: 'DELETE' });
			shares = shares.filter((x) => x.id !== s.id);
			toast(t('Link revoked.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	const expired = (s: WatchShare) => new Date(s.expires_at) < new Date();

	// --- follow ---
	const presets: [string, string][] = [
		['0 0 * * *', 'Every day at midnight'],
		['0 7 * * *', 'Every morning at 07:00'],
		['0 7,19 * * *', 'Twice a day'],
		['0 */6 * * *', 'Every 6 hours'],
		['0 * * * *', 'Every hour'],
		['0 7 * * 1', 'Every Monday']
	];
	let address = $state('');
	let preview = $state<{
		url: string;
		title: string;
		recent: string[];
		entries: number;
	} | null>(null);
	let picked = $state<string[]>([]);
	let schedule = $state('0 0 * * *');
	let custom = $state('');
	let fullText = $state(false);
	let busy = $state(false);

	async function find(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		preview = null;
		try {
			preview = await api('/v1/watch/discover', { method: 'POST', body: { url: address } });
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	async function follow() {
		busy = true;
		try {
			const r = await api<{ source: WatchSource; stats: { new: number; merged: number } }>(
				'/v1/watch/sources',
				{
					method: 'POST',
					body: {
						url: preview?.url ?? address,
						topic_ids: picked,
						schedule: schedule === 'custom' ? custom : schedule,
						full_text: fullText
					}
				}
			);
			toast(
				t('Following {name}: {n} news stored, {m} already known.', {
					name: r.source.title,
					n: r.stats.new,
					m: r.stats.merged
				})
			);
			address = '';
			preview = null;
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	// --- import ---
	type ImportResult = {
		added: number;
		existing: number;
		topics_created: number;
		failed: { line: number; url: string; reason: string }[];
	};
	let imported = $state<ImportResult | null>(null);
	async function importFile(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		const format = /\.csv$/i.test(file.name) ? 'csv' : 'opml';
		try {
			const res = await fetch(`/api/v1/watch/import?format=${format}`, {
				method: 'POST',
				headers: { 'Content-Type': format === 'csv' ? 'text/csv' : 'text/x-opml' },
				body: await file.text()
			});
			const out = await res.json();
			if (!res.ok) throw new Error(out.detail ?? res.statusText);
			imported = out;
			await load();
		} catch (err) {
			toast(err instanceof Error ? err.message : String(err), 'error');
		}
		input.value = '';
	}
	let fresh = $state({ url: '', user: '', password: '' });
	async function importFresh(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			imported = await api<ImportResult>('/v1/watch/import/freshrss', {
				method: 'POST',
				body: fresh
			});
			fresh.password = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	const sampleCSV = `${csvHeader}\nhttps://medium.com/@someone,Someone on Medium,AI|Writing,0 7 * * *,yes\nhttps://example.substack.com,,Databases,,\n`;

	// --- topics ---
	let topicName = $state('');
	let topicWords = $state('');
	let editing = $state<string | null>(null);
	async function saveTopic(e: SubmitEvent) {
		e.preventDefault();
		const body = {
			name: topicName,
			keywords: topicWords
				.split(',')
				.map((w) => w.trim())
				.filter(Boolean)
		};
		try {
			await api(editing ? `/v1/watch/topics/${editing}` : '/v1/watch/topics', {
				method: editing ? 'PUT' : 'POST',
				body
			});
			topicName = topicWords = '';
			editing = null;
			toast(t('Topic saved; the last month’s news are labelled again.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	function editTopic(tp: WatchTopic) {
		editing = tp.id;
		topicName = tp.name;
		topicWords = tp.keywords.join(', ');
	}
	async function deleteTopic(tp: WatchTopic) {
		if (!confirm(t('Remove the topic {name}?', { name: tp.name }))) return;
		try {
			await api(`/v1/watch/topics/${tp.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

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
	const maxDay = $derived(Math.max(1, ...(stats?.daily.map((d) => d.count) ?? [1])));
	let hover = $state<{ day: string; count: number; x: number } | null>(null);
</script>

<svelte:head><title>{t('Sources and health')} · Axon</title></svelte:head>

<p><a href="/watch">← {t('Tech watch')}</a></p>
<h1>{t('Sources and health')}</h1>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if stats}
	<section class="tiles" aria-label={t('At a glance')}>
		<div class="tile">
			<strong>{stats.sources.total}</strong><span>{t('sources')}</span>
			<small
				>{stats.sources.ok} OK{#if stats.sources.failing}
					· <span class="bad">{stats.sources.failing} {t('failing')}</span
					>{/if}{#if stats.sources.waiting}
					· {stats.sources.waiting} {t('waiting')}{/if}</small
			>
		</div>
		<div class="tile">
			<strong>{stats.items.unread}</strong><span>{t('unread')}</span>
			<small>{stats.items.today} {t('today')} · {stats.items.week} {t('this week')}</small>
		</div>
		<div class="tile">
			<strong>{stats.items.merged}</strong><span>{t('stories seen twice, kept once')}</span>
			<small>{stats.items.outliers} {t('outliers set aside')}</small>
		</div>
		<div class="tile">
			<strong>{stats.next_fetch_at ? ago(stats.next_fetch_at).replace('-', '') : '—'}</strong>
			<span>{t('next fetch')}</span>
			<small
				>{stats.next_fetch_at
					? new Date(stats.next_fetch_at).toLocaleString([], {
							weekday: 'short',
							hour: '2-digit',
							minute: '2-digit'
						})
					: ''}</small
			>
		</div>
	</section>

	<figure class="chart">
		<figcaption>{t('News stored per day, last 30 days')}</figcaption>
		<svg viewBox="0 0 300 80" role="img" aria-label={t('News stored per day, last 30 days')}>
			<line x1="0" y1="79.5" x2="300" y2="79.5" class="axis" />
			{#each stats.daily as d, i (d.day)}
				{@const h = Math.max(d.count ? 2 : 0, (d.count / maxDay) * 72)}
				<rect x={i * 10 + 1} y={79 - h} width="8" height={h} rx="2" class="bar" />
				<rect
					x={i * 10}
					y="0"
					width="10"
					height="80"
					fill="transparent"
					role="presentation"
					onmouseenter={() => (hover = { ...d, x: i * 10 + 5 })}
					onmouseleave={() => (hover = null)}
				/>
			{/each}
		</svg>
		{#if hover}
			<div class="tip" style="left: {(hover.x / 300) * 100}%">
				{new Date(hover.day + 'T00:00:00').toLocaleDateString()}: <strong>{hover.count}</strong>
			</div>
		{/if}
		<table class="sr-only">
			<caption>{t('News stored per day, last 30 days')}</caption>
			<tbody>
				{#each stats.daily as d (d.day)}<tr><th>{d.day}</th><td>{d.count}</td></tr>{/each}
			</tbody>
		</table>
	</figure>
{/if}

<Section id="follow" title={t('Follow a source')}>
	<form class="row" onsubmit={find}>
		<input
			aria-label={t('Address')}
			placeholder={t('A feed, a site, medium.com/@author, name.substack.com, a YouTube channel…')}
			bind:value={address}
			required
		/>
		<Button type="submit" variant="secondary" {busy}>{t('Find its feed')}</Button>
	</form>
	{#if preview}
		<div class="preview">
			<p><strong>{preview.title}</strong> <span class="muted small">{preview.url}</span></p>
			<ul class="small">
				{#each preview.recent as r, i (i)}<li>{r}</li>{/each}
			</ul>
			<div class="grid">
				<div class="field">
					<label for="sched">{t('Check it')}</label>
					<select id="sched" bind:value={schedule}>
						{#each presets as [v, l] (v)}<option value={v}>{t(l)}</option>{/each}
						<option value="custom">{t('Custom (cron)')}</option>
					</select>
					{#if schedule === 'custom'}
						<input aria-label="Cron" placeholder="30 6 * * 1-5" bind:value={custom} />
					{/if}
				</div>
				{#if topics.length}
					<fieldset class="field">
						<legend>{t('Topics')}</legend>
						{#each topics as tp (tp.id)}
							<label class="check"
								><input type="checkbox" value={tp.id} bind:group={picked} />{tp.name}</label
							>
						{/each}
					</fieldset>
				{/if}
			</div>
			<label class="check"
				><input type="checkbox" bind:checked={fullText} />{t(
					'Read each article’s page when the feed has an excerpt only (Medium, Substack previews, news sites)'
				)}</label
			>
			<Button {busy} onclick={follow}>{t('Follow')}</Button>
		</div>
	{/if}
</Section>

<Section id="import" title={t('Import')}>
	<div class="grid">
		<div>
			<label class="file"
				>{t('A CSV or OPML file')}
				<input
					type="file"
					accept=".csv,.opml,.xml,text/csv,text/x-opml"
					onchange={importFile}
				/></label
			>
			<p class="muted small">
				{t('OPML comes from Feedly, FreshRSS, Inoreader, and most readers; folders become topics.')}
				{t('CSV columns')}: <code>{csvHeader}</code> ({t('topics separated by |')}).
				<a
					href="data:text/csv;charset=utf-8,{encodeURIComponent(sampleCSV)}"
					download="axon-sources.csv">{t('Example CSV')}</a
				>
			</p>
		</div>
		<form onsubmit={importFresh} class="fresh">
			<strong>FreshRSS</strong>
			<input
				aria-label={t('FreshRSS address')}
				placeholder="https://rss.example.org"
				bind:value={fresh.url}
				required
			/>
			<input aria-label={t('User')} placeholder={t('User')} bind:value={fresh.user} required />
			<input
				aria-label={t('API password')}
				type="password"
				placeholder={t('API password')}
				autocomplete="off"
				bind:value={fresh.password}
				required
			/>
			<Button type="submit" variant="secondary" {busy}>{t('Import its subscriptions')}</Button>
			<p class="muted small">
				{t(
					'Set an API password in FreshRSS: Profile → API management. It is used once, not stored.'
				)}
			</p>
		</form>
	</div>
	{#if imported}
		<p role="status">
			{t('{a} added, {e} already followed, {t} topics created.', {
				a: imported.added,
				e: imported.existing,
				t: imported.topics_created
			})}
		</p>
		{#if imported.failed.length}
			<ul class="small bad">
				{#each imported.failed as f (f.line)}<li>
						{t('Line')}
						{f.line}: {f.url} — {f.reason}
					</li>{/each}
			</ul>
		{/if}
	{/if}
</Section>

<Section id="topics" title={t('Topics')}>
	<p class="muted small">
		{t(
			'Topics label the news: a keyword in the title, or two in the text, and the item gets the topic. Synonyms from the thesaurus count too.'
		)}
	</p>
	<ul class="topics">
		{#each topics as tp (tp.id)}
			<li>
				<strong>{tp.name}</strong>
				<span class="muted small">{tp.keywords.join(', ')}</span>
				<Button variant="ghost" onclick={() => editTopic(tp)}>{t('Edit')}</Button>
				<Button variant="ghost" onclick={() => deleteTopic(tp)}>{t('Remove')}</Button>
			</li>
		{/each}
	</ul>
	<form class="row" onsubmit={saveTopic}>
		<input
			aria-label={t('Topic')}
			placeholder={t('Topic, e.g. Databases')}
			bind:value={topicName}
			required
		/>
		<input
			aria-label={t('Keywords')}
			placeholder={t('Keywords, separated by commas')}
			bind:value={topicWords}
		/>
		<Button type="submit" variant="secondary">{editing ? t('Save') : t('Add the topic')}</Button>
	</form>
</Section>

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
							<div class="muted">{t('next')} {new Date(s.next_fetch_at).toLocaleString()}</div>
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

<Section id="shares" title={t('Shared links')}>
	<p class="muted small">
		{t(
			'Read-only links to a selection of your news, made from the reader with Export or share. Revoke one and it stops working at once.'
		)}
	</p>
	{#if shares.length}
		<div class="table">
			<table>
				<thead>
					<tr>
						<th>{t('Name')}</th>
						<th>{t('News')}</th>
						<th>{t('Expires')}</th>
						<th>{t('Views')}</th>
						<th><span class="sr-only">{t('Actions')}</span></th>
					</tr>
				</thead>
				<tbody>
					{#each shares as s (s.id)}
						<tr>
							<td><strong>{s.name}</strong></td>
							<td class="small">{s.description}</td>
							<td class="small" class:bad={expired(s)}>
								{expired(s) ? t('Expired') : new Date(s.expires_at).toLocaleDateString()}
							</td>
							<td class="small">
								{s.views}
								{#if s.last_viewed_at}<div class="muted">{ago(s.last_viewed_at)}</div>{/if}
							</td>
							<td>
								<Button variant="ghost" onclick={() => revoke(s)}
									>{t('Revoke')}<span class="sr-only"> {s.name}</span></Button
								>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<p class="muted small">{t('No links yet.')}</p>
	{/if}
</Section>

<style>
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
		gap: var(--space-3);
		margin: var(--space-4) 0;
	}
	.tile {
		display: grid;
		gap: 0.1rem;
		padding: var(--space-3);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
	}
	.tile strong {
		font-size: var(--text-2xl);
	}
	.tile small {
		color: var(--muted);
	}
	.chart {
		position: relative;
		margin: 0 0 var(--space-6);
	}
	.chart figcaption {
		font-size: var(--text-sm);
		color: var(--muted);
		margin-bottom: var(--space-1);
	}
	.chart svg {
		width: 100%;
		height: 8rem;
	}
	.bar {
		fill: var(--accent);
	}
	.axis {
		stroke: var(--border);
		stroke-width: 1;
	}
	.tip {
		position: absolute;
		top: 1.2rem;
		transform: translateX(-50%);
		background: var(--surface-solid);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.1rem var(--space-2);
		font-size: var(--text-sm);
		pointer-events: none;
		white-space: nowrap;
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
	}
	.row input {
		flex: 1 1 14rem;
		min-width: 0;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 18rem), 1fr));
		gap: var(--space-4);
		margin: var(--space-3) 0;
	}
	.field {
		display: grid;
		gap: var(--space-1);
		border: 0;
		padding: 0;
		margin: 0;
	}
	.check {
		display: flex;
		gap: var(--space-1);
		align-items: baseline;
	}
	.preview {
		margin-top: var(--space-3);
		padding: var(--space-3);
		border: 1px solid var(--border);
		border-radius: var(--radius);
	}
	.fresh {
		display: grid;
		gap: var(--space-2);
	}
	.file {
		display: grid;
		gap: var(--space-1);
		font-weight: 600;
	}
	.topics {
		list-style: none;
		padding: 0;
	}
	.topics li {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
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
