<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, apiResponse, message, type Schemas } from '$lib/api';
	import { formatDate, localDate, appLocale } from '$lib/dates';
	import {
		byDay,
		monthGrid,
		monthLabel,
		monthRange,
		shiftMonth,
		spreadEvents,
		type External,
		type Occurrence
	} from '$lib/calendar';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Own = Schemas['CalendarEvent'];
	type Source = Schemas['CalendarSource'];
	type Event = Own | External;
	type Feed = Schemas['CalendarFeed'];
	type Kind = 'task' | 'goal' | 'milestone' | 'study' | 'certification' | 'habit';

	const kinds: { id: Kind; label: string }[] = $derived([
		{ id: 'task', label: t('Tasks') },
		{ id: 'goal', label: t('Goal target dates') },
		{ id: 'milestone', label: t('Milestones') },
		{ id: 'study', label: t('Study blocks') },
		{ id: 'certification', label: t('Certifications') },
		{ id: 'habit', label: t('Habits') }
	]);
	// What an iCal feed can hold.
	const feedKinds = $derived(kinds.filter((k) => ['task', 'goal', 'milestone'].includes(k.id)));
	const prefix: Record<string, string> = $derived({
		scheduled: '',
		due: t('Due'),
		target: t('Goal target'),
		milestone: t('Milestone'),
		block: t('Study'),
		exam: t('Exam'),
		expiry: t('Expiry'),
		habit: t('Habit')
	});

	const todayISO = localDate(new Date());
	const thisMonth = todayISO.slice(0, 7);
	const month = $derived(page.url.searchParams.get('month') ?? thisMonth);
	// The month grid is the default; the agenda lists the same events by day.
	const layout = $derived(page.url.searchParams.get('layout') === 'agenda' ? 'agenda' : 'month');
	const weeks = $derived(monthGrid(month));
	const weekdays = $derived(weeks[0].map((d) => formatDate(d, { weekday: 'short' })));
	const query = (changes: Record<string, string | null>) => {
		const u = new URLSearchParams(page.url.searchParams);
		for (const [k, v] of Object.entries(changes)) {
			if (v === null) u.delete(k);
			else u.set(k, v);
		}
		return `?${u}`;
	};
	const shown = $derived(
		(page.url.searchParams.get('types') ?? 'task,goal,milestone,study,certification,habit').split(
			','
		) as Kind[]
	);
	let own = $state<Own[] | null>(null);
	let occurrences = $state<Occurrence[]>([]);
	let sources = $state<Source[]>([]);
	let showFollowed = $state(true);
	let error = $state('');
	const external = $derived(
		showFollowed
			? spreadEvents(occurrences, sources, (iso) =>
					new Date(iso).toLocaleTimeString(appLocale(), { hour: '2-digit', minute: '2-digit' })
				)
			: []
	);
	const events = $derived<Event[] | null>(own ? [...external, ...own] : null);
	const byDate = $derived(new Map(byDay(events ?? [])));
	// In the month grid, a day's habits (EP-167) are one line: how many are
	// done, each named in its tooltip; the agenda lists them one by one.
	const listed = (d: string) => (byDate.get(d) ?? []).filter((e) => e.type !== 'habit');
	const habitsOn = (d: string) => (byDate.get(d) ?? []).filter((e) => e.type === 'habit');
	// Drag and drop (EP-64): a task moves to another day, keeping what the
	// date means (scheduled or due); undated next actions can be dropped on a
	// day to schedule them. Each has a date field too, for the keyboard. The
	// newest come first: a started roadmap adds many undated exit checks.
	type TaskLite = {
		id: string;
		title: string;
		scheduled_on: string | null;
		due_on: string | null;
		created_at: string;
	};
	let undated = $state<TaskLite[]>([]);
	let undatedMore = $state(0);
	let overDay = $state('');
	async function loadUndated() {
		try {
			const r = await api<{ items: TaskLite[] }>('/v1/tasks?view=next&limit=100');
			const all = r.items
				.filter((t) => !t.scheduled_on && !t.due_on)
				.sort((a, b) => b.created_at.localeCompare(a.created_at));
			undated = all.slice(0, 20);
			undatedMore = all.length - undated.length;
		} catch {
			undated = [];
		}
	}
	async function setDate(id: string, field: 'scheduled_on' | 'due_on', day: string) {
		try {
			const { etag } = await apiResponse(`/v1/tasks/${id}`);
			await api(`/v1/tasks/${id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: { [field]: day }
			});
			toast(`Moved to ${dayLabel(day)}.`);
			await Promise.all([load(), loadUndated()]);
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	function drag(e: DragEvent, id: string, field: 'scheduled_on' | 'due_on') {
		e.dataTransfer?.setData('application/x-axon-task', JSON.stringify({ id, field }));
		if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
	}
	function drop(e: DragEvent, day: string) {
		overDay = '';
		const raw = e.dataTransfer?.getData('application/x-axon-task');
		if (!raw) return;
		e.preventDefault();
		const { id, field } = JSON.parse(raw) as { id: string; field: 'scheduled_on' | 'due_on' };
		setDate(id, field, day);
	}
	function over(e: DragEvent, day: string) {
		if (e.dataTransfer?.types.includes('application/x-axon-task')) {
			e.preventDefault();
			overDay = day;
		}
	}
	let sourceName = $state('');
	let sourceUrl = $state('');
	let sourceColor = $state('#22aa66');
	let adding = $state(false);
	let feeds = $state<Feed[]>([]);
	let feedName = $state('');
	let feedTypes = $state<Kind[]>(['task', 'milestone']);
	let newFeedUrl = $state('');

	async function load() {
		error = '';
		const { from, to } =
			layout === 'month' ? { from: weeks[0][0], to: weeks.at(-1)!.at(-1)! } : monthRange(month);
		try {
			const params = new URLSearchParams({ from, to, types: shown.join(',') });
			const [mine, theirs] = await Promise.all([
				shown.length ? api<{ items: Own[] }>(`/v1/calendar?${params}`) : { items: [] },
				api<{ items: Occurrence[] }>(`/v1/calendar/events?${new URLSearchParams({ from, to })}`)
			]);
			own = mine.items;
			occurrences = theirs.items;
		} catch (err) {
			error = message(err);
		}
	}

	async function loadFeeds() {
		try {
			feeds = (await api<{ items: Feed[] }>('/v1/me/calendar-feeds')).items;
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	$effect(() => {
		void month;
		void shown;
		void layout;
		own = null;
		load();
	});
	$effect(() => {
		loadFeeds();
		loadSources();
		loadUndated();
	});

	async function loadSources() {
		try {
			sources = (await api<{ items: Source[] }>('/v1/calendar/sources')).items;
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function follow(e: SubmitEvent) {
		e.preventDefault();
		adding = true;
		try {
			const src = await api<Source>('/v1/calendar/sources', {
				method: 'POST',
				body: { name: sourceName, url: sourceUrl, color: sourceColor }
			});
			toast(t('Following “{name}”: {events} events.', { name: src.name, events: src.events }));
			sourceName = '';
			sourceUrl = '';
			await Promise.all([loadSources(), load()]);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			adding = false;
		}
	}

	async function refreshSource(src: Source) {
		try {
			const after = await api<Source>(`/v1/calendar/sources/${src.id}/refresh`, { method: 'POST' });
			toast(
				after.error ? after.error : t('Updated “{name}”.', { name: after.name }),
				after.error ? 'error' : 'ok'
			);
			await Promise.all([loadSources(), load()]);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function unfollow(src: Source) {
		try {
			await api(`/v1/calendar/sources/${src.id}`, { method: 'DELETE' });
			toast(t('No longer following “{name}”.', { name: src.name }));
			await Promise.all([loadSources(), load()]);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function setTypes(id: Kind, on: boolean) {
		const next = on ? [...shown, id] : shown.filter((t) => t !== id);
		const u = new URL(page.url);
		u.searchParams.set(
			'types',
			kinds
				.map((k) => k.id)
				.filter((k) => next.includes(k))
				.join(',')
		);
		goto(u, { keepFocus: true, noScroll: true, replaceState: true });
	}

	function href(e: Event): string | null {
		if (e.type === 'external') return null;
		if (e.type === 'goal') return `/goals/${e.id}`;
		if (e.type === 'milestone' && e.goal_id) return `/goals/${e.goal_id}`;
		if (e.type === 'task') return `/tasks/${e.id}`;
		// A roadmap's weekly rhythm (EP-97): the day's block opens the roadmap.
		if (e.type === 'study' && e.roadmap_id) return `/roadmaps/${e.roadmap_id}`;
		if (e.type === 'certification') return '/learn/certifications';
		if (e.type === 'habit') return `/habits/${e.id}`;
		return null;
	}

	function dayLabel(iso: string): string {
		return formatDate(iso, { weekday: 'long', day: 'numeric', month: 'long' });
	}

	async function createFeed(e: SubmitEvent) {
		e.preventDefault();
		try {
			const res = await api<{ feed: Feed; path: string }>('/v1/me/calendar-feeds', {
				method: 'POST',
				body: { name: feedName, types: feedTypes }
			});
			newFeedUrl = page.url.origin + res.path;
			feedName = '';
			await loadFeeds();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function revoke(f: Feed) {
		try {
			await api(`/v1/me/calendar-feeds/${f.id}`, { method: 'DELETE' });
			toast(t('Revoked “{name}”. Its URL no longer works.', { name: f.name }));
			await loadFeeds();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function copy() {
		await navigator.clipboard.writeText(newFeedUrl);
		toast(t('Copied.'));
	}
</script>

<svelte:head><title>{t('Calendar · Axon')}</title></svelte:head>

<h1>{t('Calendar')}</h1>

<div class="toolbar">
	<nav aria-label={t('Month')} class="months">
		<a href={query({ month: shiftMonth(month, -1) })} aria-label={t('Previous month')}>←</a>
		<h2 aria-live="polite">{monthLabel(month)}</h2>
		<a href={query({ month: shiftMonth(month, 1) })} aria-label={t('Next month')}>→</a>
		{#if month !== thisMonth}<a href={query({ month: null })} class="small">{t('This month')}</a
			>{/if}
	</nav>
	<nav aria-label={t('Calendar view')} class="layouts">
		<a href={query({ layout: null })} aria-current={layout === 'month' ? 'page' : undefined}
			>{t('Month')}</a
		>
		<a href={query({ layout: 'agenda' })} aria-current={layout === 'agenda' ? 'page' : undefined}
			>{t('Agenda')}</a
		>
	</nav>
	<fieldset class="filters">
		<legend class="visually-hidden">{t('Show')}</legend>
		{#each kinds as k (k.id)}
			<label>
				<input
					type="checkbox"
					checked={shown.includes(k.id)}
					onchange={(e) => setTypes(k.id, e.currentTarget.checked)}
				/>
				{k.label}
			</label>
		{/each}
		{#if sources.length}
			<label
				><input type="checkbox" bind:checked={showFollowed} /> {t('Calendars you follow')}</label
			>
		{/if}
	</fieldset>
</div>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !events}
	<Status kind="loading" message={t('Loading the calendar…')} />
{:else if layout === 'month'}
	<table class="grid">
		<caption class="visually-hidden">{monthLabel(month)}</caption>
		<thead>
			<tr
				>{#each weekdays as w (w)}<th scope="col">{w}</th>{/each}</tr
			>
		</thead>
		<tbody>
			{#each weeks as week (week[0])}
				<tr>
					{#each week as d (d)}
						<td
							class:out={!d.startsWith(month)}
							class:today={d === todayISO}
							class:over={overDay === d}
							ondragover={(e) => over(e, d)}
							ondragleave={() => (overDay = '')}
							ondrop={(e) => drop(e, d)}
						>
							<span class="num" aria-label={dayLabel(d)}>{Number(d.slice(8))}</span>
							{#if listed(d).length}
								<ul>
									{#each listed(d) as e (e.id + e.kind)}
										<li
											class="ev {e.type}"
											class:done={e.done}
											style={e.type === 'external' ? `--c: ${e.color}` : undefined}
											draggable={e.type === 'task' && !e.done ? 'true' : undefined}
											ondragstart={e.type === 'task'
												? (ev) => drag(ev, e.id, e.kind === 'due' ? 'due_on' : 'scheduled_on')
												: undefined}
										>
											{#if e.type === 'external'}<span class="visually-hidden">{e.calendar}:</span
												>{#if e.time}<span class="time">{e.time}</span
													>{/if}{:else if prefix[e.kind]}<span class="visually-hidden"
													>{prefix[e.kind]}:</span
												>{/if}
											{#if href(e)}<a href={href(e)} title={e.title}>{e.title}</a>{:else}<span
													title={e.title}>{e.title}</span
												>{/if}
											{#if e.done}<span class="visually-hidden">{t('(done)')}</span>{/if}
										</li>
									{/each}
								</ul>
							{/if}
							{#if habitsOn(d).length}
								{@const hs = habitsOn(d)}
								<a
									class="ev habit"
									href="/habits"
									title={hs.map((h) => (h.done ? '✓ ' : '· ') + h.title).join('\n')}
									>{t('Habits {done}/{n}', {
										done: hs.filter((h) => h.done).length,
										n: hs.length
									})}</a
								>
							{/if}
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</table>
	<p class="legend">
		<span class="ev task">{t('Task')}</span><span class="ev goal">{t('Goal target')}</span><span
			class="ev milestone">{t('Milestone')}</span
		><span class="ev study">{t('Study block')}</span><span class="ev certification"
			>{t('Certification')}</span
		><span class="ev habit">{t('Habit')}</span>{#each sources as src (src.id)}<span
				class="ev external"
				style:--c={src.color}>{src.name}</span
			>{/each}
	</p>
{:else if events.length === 0}
	<Status
		kind="empty"
		message={t(
			'Nothing dated this month. Give tasks a date, or goals and milestones a target date.'
		)}
	/>
{:else}
	<ol class="days">
		{#each byDay(events) as [date, items] (date)}
			<li>
				<h3>{dayLabel(date)}</h3>
				<ul>
					{#each items as e (e.id + e.kind)}
						<li class:done={e.done}>
							{#if e.type === 'external'}
								<span class="chip external" style:--c={e.color}>{e.calendar}</span>
								{#if e.time}<span class="time">{e.time}</span>{/if}
							{:else}
								<span class="chip {e.type}">{prefix[e.kind] || t('Task')}</span>
							{/if}
							{#if href(e)}<a href={href(e)}>{e.title}</a>{:else}<span>{e.title}</span>{/if}
							{#if e.done}<span class="visually-hidden">{t('(done)')}</span>{/if}
						</li>
					{/each}
				</ul>
			</li>
		{/each}
	</ol>
{/if}

{#if undated.length}
	<section class="card" aria-labelledby="undated-title">
		<h2 id="undated-title">{t('Next actions without a date')}</h2>
		<p class="muted small">{t('Drag one onto a day to schedule it, or pick a date.')}</p>
		<ul class="undated">
			{#each undated as item (item.id)}
				<li draggable="true" ondragstart={(e) => drag(e, item.id, 'scheduled_on')}>
					<span class="grow">{item.title}</span>
					<label class="visually-hidden" for="when-{item.id}"
						>{t('Schedule {title} on', { title: item.title })}</label
					>
					<input
						id="when-{item.id}"
						type="date"
						onchange={(e) =>
							e.currentTarget.value && setDate(item.id, 'scheduled_on', e.currentTarget.value)}
					/>
				</li>
			{/each}
		</ul>
		{#if undatedMore}<a class="small" href="/tasks?view=next"
				>{t('{n} more in Next actions', { n: undatedMore })}</a
			>{/if}
	</section>
{/if}

<section class="card" aria-labelledby="follow-title">
	<h2 id="follow-title">{t('Calendars you follow')}</h2>
	<p class="muted">
		{t(
			'See your other calendars here, read only: Google Calendar (Settings → your calendar → “Secret address in iCal format”), Nextcloud or any CalDAV server (“Copy subscription link”), Outlook (“Publish calendar” → ICS), or any webcal:// or .ics address. Axon checks them every 30 minutes and never changes them.'
		)}
	</p>
	{#if sources.length}
		<ul class="feeds">
			{#each sources as src (src.id)}
				<li>
					<span class="swatch" style:background={src.color} aria-hidden="true"></span>
					<span class="grow"
						><strong>{src.name}</strong>
						<span class="muted small"
							>{t('{host} · {events} events', {
								host: src.host,
								events: src.events
							})}{#if src.fetched_at}
								{t('· checked {value}', {
									value: new Date(src.fetched_at).toLocaleString(appLocale())
								})}{/if}</span
						>
						{#if src.error}<span class="small bad" role="status">{src.error}</span>{/if}</span
					>
					<Button
						variant="ghost"
						onclick={() => refreshSource(src)}
						aria-label={t('Check {name} now', { name: src.name })}>{t('Check now')}</Button
					>
					<Button
						variant="danger"
						onclick={() => unfollow(src)}
						aria-label={t('Stop following {name}', { name: src.name })}>{t('Remove')}</Button
					>
				</li>
			{/each}
		</ul>
	{/if}
	<form onsubmit={follow}>
		<div class="row">
			<TextField
				id="source-name"
				label={t('Name')}
				placeholder={t('Work')}
				bind:value={sourceName}
				maxlength={100}
				required
			/>
			<div class="field">
				<label for="source-color">{t('Colour')}</label>
				<input id="source-color" type="color" bind:value={sourceColor} />
			</div>
		</div>
		<TextField
			id="source-url"
			label={t('Calendar address (iCal / ICS)')}
			placeholder="https://calendar.google.com/calendar/ical/…/basic.ics"
			bind:value={sourceUrl}
			maxlength={2000}
			required
		/>
		<p class="muted small">
			{t('The address is a secret link: Axon keeps it encrypted and shows only its host.')}
		</p>
		<Button type="submit" variant="secondary" busy={adding}>{t('Follow')}</Button>
	</form>
</section>

<section class="card" aria-labelledby="feeds-title">
	<h2 id="feeds-title">{t('Subscribe in your calendar app')}</h2>
	<p class="muted">
		{t(
			'A read-only iCal feed with titles only, for the types you choose. Anyone with its URL can read it, so revoke a feed you no longer use.'
		)}
	</p>
	{#if newFeedUrl}
		<div class="new-feed" role="status">
			<p><strong>{t('Copy this URL now. It will not be shown again.')}</strong></p>
			<code>{newFeedUrl}</code>
			<div class="row">
				<Button variant="secondary" onclick={copy}>{t('Copy URL')}</Button>
				<Button variant="ghost" onclick={() => (newFeedUrl = '')}>{t('Done')}</Button>
			</div>
		</div>
	{/if}
	{#if feeds.length}
		<ul class="feeds">
			{#each feeds as f (f.id)}
				<li>
					<span class="grow"
						><strong>{f.name}</strong>
						<span class="muted small"
							>{f.types.join(', ') +
								' · ' +
								(f.last_used_at
									? t('last read {when}', {
											when: new Date(f.last_used_at).toLocaleString(appLocale())
										})
									: t('not used yet'))}</span
						></span
					>
					<Button
						variant="danger"
						onclick={() => revoke(f)}
						aria-label={t('Revoke {name}', { name: f.name })}>{t('Revoke')}</Button
					>
				</li>
			{/each}
		</ul>
	{/if}
	<form onsubmit={createFeed}>
		<TextField
			id="feed-name"
			label={t('Feed name')}
			placeholder={t('Phone calendar')}
			bind:value={feedName}
			maxlength={100}
			required
		/>
		<fieldset class="filters">
			<legend>{t('Include')}</legend>
			{#each feedKinds as k (k.id)}
				<label><input type="checkbox" value={k.id} bind:group={feedTypes} /> {k.label}</label>
			{/each}
		</fieldset>
		<Button type="submit" variant="secondary" disabled={feedTypes.length === 0}
			>{t('Create feed')}</Button
		>
	</form>
</section>

<style>
	td.over {
		outline: 2px dashed var(--accent);
		outline-offset: -2px;
	}
	li[draggable='true'] {
		cursor: grab;
	}
	.undated {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: minmax(0, 1fr);
		gap: var(--space-1);
	}
	.undated li {
		display: flex;
		flex-wrap: wrap; /* on a phone, the date goes under the title */
		gap: var(--space-2);
		align-items: center;
		padding: var(--space-1) var(--space-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
	}
	.ev.external {
		border-left: 3px solid var(--c);
	}
	.chip.external {
		border: 1px solid var(--c);
	}
	.time {
		font-variant-numeric: tabular-nums;
		margin-right: var(--space-1);
		opacity: 0.8;
	}
	.swatch {
		width: 0.9rem;
		height: 0.9rem;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.bad {
		display: block;
		color: var(--bad);
	}
	.toolbar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-4);
		flex-wrap: wrap;
		margin: var(--space-4) 0 var(--space-6);
	}
	.months {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}
	.months h2 {
		margin: 0;
		font-size: var(--text-lg);
		min-width: 10rem;
		text-align: center;
	}
	.months a {
		text-decoration: none;
		padding: var(--space-1) var(--space-2);
	}
	.small {
		font-size: var(--text-sm);
	}
	.filters {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		border: none;
		padding: 0;
		margin: 0 0 var(--space-4);
	}
	.filters legend {
		font-weight: 600;
		margin-bottom: var(--space-2);
	}
	.filters label {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}
	.filters input {
		width: 1.1rem;
		height: 1.1rem;
		accent-color: var(--accent);
	}
	.days {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-8);
	}
	.days > li {
		margin-bottom: var(--space-4);
	}
	.days h3 {
		font-size: var(--text-base);
		margin: 0 0 var(--space-2);
		color: var(--muted);
	}
	.days ul {
		list-style: none;
		padding: 0;
		margin: 0;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
	}
	.days ul li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-4);
		border-bottom: 1px solid var(--border);
		overflow-wrap: anywhere;
	}
	.days ul li:last-child {
		border-bottom: none;
	}
	.done {
		color: var(--muted);
		text-decoration: line-through;
	}
	.chip {
		flex-shrink: 0;
		font-size: var(--text-sm);
		border-radius: 999px;
		padding: 0 var(--space-2);
		background: var(--accent-soft);
		color: var(--fg);
	}
	.layouts {
		display: flex;
		gap: var(--space-1);
	}
	.layouts a {
		padding: var(--space-1) var(--space-3);
		border-radius: 999px;
		color: var(--muted);
		text-decoration: none;
	}
	.layouts a[aria-current='page'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
	}
	.grid {
		width: 100%;
		table-layout: fixed;
		border-collapse: separate;
		border-spacing: 0;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		overflow: hidden;
		margin-bottom: var(--space-4);
	}
	.grid th {
		font-size: var(--text-sm);
		color: var(--muted);
		font-weight: 600;
		padding: var(--space-2);
		border-bottom: 1px solid var(--border);
	}
	.grid td {
		vertical-align: top;
		height: 7.5rem;
		padding: var(--space-1) var(--space-2);
		border-right: 1px solid var(--border);
		border-bottom: 1px solid var(--border);
	}
	.grid td:last-child {
		border-right: none;
	}
	.grid tr:last-child td {
		border-bottom: none;
	}
	.grid td.out {
		background: rgb(127 127 127 / 0.06);
	}
	.grid td.out .num {
		color: var(--muted);
	}
	.num {
		display: inline-grid;
		place-items: center;
		min-width: 1.75rem;
		height: 1.75rem;
		font-size: var(--text-sm);
		font-weight: 600;
		border-radius: 999px;
	}
	.today .num {
		background: var(--accent);
		color: var(--accent-fg);
	}
	.grid ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		/* Events shrink to their day's width and end in an ellipsis. */
		grid-template-columns: minmax(0, 1fr);
		gap: 2px;
	}
	.ev {
		display: block;
		font-size: var(--text-sm);
		border-radius: var(--radius-sm);
		padding: 0 var(--space-1);
		background: var(--accent-soft);
		border-left: 3px solid var(--chart-1);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.ev a {
		color: var(--fg);
		text-decoration: none;
	}
	.ev a:hover {
		text-decoration: underline;
	}
	.ev.goal {
		border-left-color: var(--chart-2);
	}
	.ev.milestone {
		border-left-color: var(--chart-3);
	}
	/* A study block is a plan, not a commitment: the task colour, dashed. */
	.ev.certification {
		border-left-color: var(--accent);
	}
	.ev.habit {
		display: block;
		border-left-color: var(--chart-2);
		font-size: var(--text-xs, 0.75rem);
	}
	.ev.study {
		border-left-style: dashed;
		background: transparent;
	}
	.ev.done {
		text-decoration: line-through; /* no opacity: the text keeps its contrast */
		background: transparent;
	}
	.legend {
		display: flex;
		gap: var(--space-3);
		margin-bottom: var(--space-8);
	}
	@media (max-width: 40rem) {
		.grid td {
			height: 4.5rem;
			padding: 2px;
		}
		.ev {
			font-size: 0.7rem;
		}
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
	}
	@media (max-width: 30rem) {
		.card {
			padding: var(--space-4);
		}
	}
	.card h2 {
		font-size: var(--text-lg);
	}
	.new-feed {
		border: 1px solid var(--accent);
		border-radius: var(--radius);
		padding: var(--space-3);
		margin-bottom: var(--space-4);
	}
	.new-feed p {
		margin: 0 0 var(--space-2);
	}
	.new-feed code {
		display: block;
		overflow-wrap: anywhere;
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		margin-bottom: var(--space-2);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.feeds {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-4);
	}
	.feeds li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
	}
	.grow {
		flex: 1;
		min-width: 10rem;
		display: grid;
	}
</style>
