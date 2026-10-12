<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import { byDay, monthGrid, monthLabel, monthRange, shiftMonth } from '$lib/calendar';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Own = Schemas['CalendarEvent'];
	type Source = Schemas['CalendarSource'];
	type Occurrence = {
		source_id: string;
		title: string;
		location: string;
		start: string;
		end: string;
		all_day: boolean;
		date: string;
		last_date: string;
	};
	// Events of followed calendars (EP-58) sit in the grid beside Axon's own, read only.
	type External = {
		type: 'external';
		kind: 'external';
		id: string;
		date: string;
		title: string;
		done: false;
		color: string;
		time: string;
		location: string;
		calendar: string;
	};
	type Event = Own | External;
	type Feed = Schemas['CalendarFeed'];
	type Kind = 'task' | 'goal' | 'milestone';

	const kinds: { id: Kind; label: string }[] = [
		{ id: 'task', label: 'Tasks' },
		{ id: 'goal', label: 'Goal target dates' },
		{ id: 'milestone', label: 'Milestones' }
	];
	const prefix: Record<string, string> = {
		scheduled: '',
		due: 'Due',
		target: 'Goal target',
		milestone: 'Milestone'
	};

	const todayISO = new Date().toLocaleDateString('sv'); // YYYY-MM-DD in local time
	const thisMonth = todayISO.slice(0, 7);
	const month = $derived(page.url.searchParams.get('month') ?? thisMonth);
	// The month grid is the default; the agenda lists the same events by day.
	const layout = $derived(page.url.searchParams.get('layout') === 'agenda' ? 'agenda' : 'month');
	const weeks = $derived(monthGrid(month));
	const weekdays = $derived(
		weeks[0].map((d) =>
			new Date(`${d}T00:00:00Z`).toLocaleDateString(undefined, {
				weekday: 'short',
				timeZone: 'UTC'
			})
		)
	);
	const query = (changes: Record<string, string | null>) => {
		const u = new URLSearchParams(page.url.searchParams);
		for (const [k, v] of Object.entries(changes)) {
			if (v === null) u.delete(k);
			else u.set(k, v);
		}
		return `?${u}`;
	};
	const shown = $derived(
		(page.url.searchParams.get('types') ?? 'task,goal,milestone').split(',') as Kind[]
	);
	let own = $state<Own[] | null>(null);
	let occurrences = $state<Occurrence[]>([]);
	let sources = $state<Source[]>([]);
	let showFollowed = $state(true);
	let error = $state('');
	const external = $derived.by<External[]>(() => {
		if (!showFollowed) return [];
		const byId = new Map(sources.map((src) => [src.id, src]));
		const out: External[] = [];
		for (const o of occurrences) {
			const src = byId.get(o.source_id);
			const time = o.all_day
				? ''
				: new Date(o.start).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
			// An event over several days shows on each of them.
			for (let d = o.date; d <= o.last_date; d = nextDay(d)) {
				out.push({
					type: 'external',
					kind: 'external',
					id: `${o.source_id}-${o.start}-${d}`,
					date: d,
					title: o.title,
					done: false,
					color: src?.color ?? '#5aa9ff',
					time,
					location: o.location,
					calendar: src?.name ?? ''
				});
			}
		}
		return out;
	});
	const events = $derived<Event[] | null>(own ? [...external, ...own] : null);
	const byDate = $derived(new Map(byDay(events ?? [])));
	function nextDay(iso: string): string {
		const d = new Date(`${iso}T00:00:00Z`);
		d.setUTCDate(d.getUTCDate() + 1);
		return d.toISOString().slice(0, 10);
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
			toast(`Following “${src.name}”: ${src.events} events.`);
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
			toast(after.error ? after.error : `Updated “${after.name}”.`, after.error ? 'error' : 'ok');
			await Promise.all([loadSources(), load()]);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function unfollow(src: Source) {
		try {
			await api(`/v1/calendar/sources/${src.id}`, { method: 'DELETE' });
			toast(`No longer following “${src.name}”.`);
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
		return null;
	}

	function dayLabel(iso: string): string {
		return new Date(`${iso}T00:00:00Z`).toLocaleDateString(undefined, {
			weekday: 'long',
			day: 'numeric',
			month: 'long',
			timeZone: 'UTC'
		});
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
			toast(`Revoked “${f.name}”. Its URL no longer works.`);
			await loadFeeds();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function copy() {
		await navigator.clipboard.writeText(newFeedUrl);
		toast('Copied.');
	}
</script>

<svelte:head><title>Calendar · Axon</title></svelte:head>

<h1>Calendar</h1>

<div class="toolbar">
	<nav aria-label="Month" class="months">
		<a href={query({ month: shiftMonth(month, -1) })} aria-label="Previous month">←</a>
		<h2 aria-live="polite">{monthLabel(month)}</h2>
		<a href={query({ month: shiftMonth(month, 1) })} aria-label="Next month">→</a>
		{#if month !== thisMonth}<a href={query({ month: null })} class="small">This month</a>{/if}
	</nav>
	<nav aria-label="Calendar view" class="layouts">
		<a href={query({ layout: null })} aria-current={layout === 'month' ? 'page' : undefined}
			>Month</a
		>
		<a href={query({ layout: 'agenda' })} aria-current={layout === 'agenda' ? 'page' : undefined}
			>Agenda</a
		>
	</nav>
	<fieldset class="filters">
		<legend class="visually-hidden">Show</legend>
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
			<label><input type="checkbox" bind:checked={showFollowed} /> Calendars you follow</label>
		{/if}
	</fieldset>
</div>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !events}
	<Status kind="loading" message="Loading the calendar…" />
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
						<td class:out={!d.startsWith(month)} class:today={d === todayISO}>
							<span class="num" aria-label={dayLabel(d)}>{Number(d.slice(8))}</span>
							{#if byDate.get(d)}
								<ul>
									{#each byDate.get(d) ?? [] as e (e.id + e.kind)}
										<li
											class="ev {e.type}"
											class:done={e.done}
											style={e.type === 'external' ? `--c: ${e.color}` : undefined}
										>
											{#if e.type === 'external'}<span class="visually-hidden">{e.calendar}:</span
												>{#if e.time}<span class="time">{e.time}</span
													>{/if}{:else if prefix[e.kind]}<span class="visually-hidden"
													>{prefix[e.kind]}:</span
												>{/if}
											{#if href(e)}<a href={href(e)} title={e.title}>{e.title}</a>{:else}<span
													title={e.title}>{e.title}</span
												>{/if}
											{#if e.done}<span class="visually-hidden">(done)</span>{/if}
										</li>
									{/each}
								</ul>
							{/if}
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</table>
	<p class="legend">
		<span class="ev task">Task</span><span class="ev goal">Goal target</span><span
			class="ev milestone">Milestone</span
		>{#each sources as src (src.id)}<span class="ev external" style="--c: {src.color}"
				>{src.name}</span
			>{/each}
	</p>
{:else if events.length === 0}
	<Status
		kind="empty"
		message="Nothing dated this month. Give tasks a date, or goals and milestones a target date."
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
								<span class="chip external" style="--c: {e.color}">{e.calendar}</span>
								{#if e.time}<span class="time">{e.time}</span>{/if}
							{:else}
								<span class="chip {e.type}">{prefix[e.kind] || 'Task'}</span>
							{/if}
							{#if href(e)}<a href={href(e)}>{e.title}</a>{:else}<span>{e.title}</span>{/if}
							{#if e.done}<span class="visually-hidden">(done)</span>{/if}
						</li>
					{/each}
				</ul>
			</li>
		{/each}
	</ol>
{/if}

<section class="card" aria-labelledby="follow-title">
	<h2 id="follow-title">Calendars you follow</h2>
	<p class="muted">
		See your other calendars here, read only: Google Calendar (Settings → your calendar → “Secret
		address in iCal format”), Nextcloud or any CalDAV server (“Copy subscription link”), Outlook
		(“Publish calendar” → ICS), or any webcal:// or .ics address. Axon checks them every 30 minutes
		and never changes them.
	</p>
	{#if sources.length}
		<ul class="feeds">
			{#each sources as src (src.id)}
				<li>
					<span class="swatch" style="background: {src.color}" aria-hidden="true"></span>
					<span class="grow"
						><strong>{src.name}</strong>
						<span class="muted small"
							>{src.host} · {src.events} events{#if src.fetched_at}
								· checked {new Date(src.fetched_at).toLocaleString()}{/if}</span
						>
						{#if src.error}<span class="small bad" role="status">{src.error}</span>{/if}</span
					>
					<Button
						variant="ghost"
						onclick={() => refreshSource(src)}
						aria-label="Check {src.name} now">Check now</Button
					>
					<Button
						variant="danger"
						onclick={() => unfollow(src)}
						aria-label="Stop following {src.name}">Remove</Button
					>
				</li>
			{/each}
		</ul>
	{/if}
	<form onsubmit={follow}>
		<div class="row">
			<TextField
				id="source-name"
				label="Name"
				placeholder="Work"
				bind:value={sourceName}
				maxlength={100}
				required
			/>
			<div class="field">
				<label for="source-color">Colour</label>
				<input id="source-color" type="color" bind:value={sourceColor} />
			</div>
		</div>
		<TextField
			id="source-url"
			label="Calendar address (iCal / ICS)"
			placeholder="https://calendar.google.com/calendar/ical/…/basic.ics"
			bind:value={sourceUrl}
			maxlength={2000}
			required
		/>
		<p class="muted small">
			The address is a secret link: Axon keeps it encrypted and shows only its host.
		</p>
		<Button type="submit" variant="secondary" busy={adding}>Follow</Button>
	</form>
</section>

<section class="card" aria-labelledby="feeds-title">
	<h2 id="feeds-title">Subscribe in your calendar app</h2>
	<p class="muted">
		A read-only iCal feed with titles only, for the types you choose. Anyone with its URL can read
		it, so revoke a feed you no longer use.
	</p>
	{#if newFeedUrl}
		<div class="new-feed" role="status">
			<p><strong>Copy this URL now. It will not be shown again.</strong></p>
			<code>{newFeedUrl}</code>
			<div class="row">
				<Button variant="secondary" onclick={copy}>Copy URL</Button>
				<Button variant="ghost" onclick={() => (newFeedUrl = '')}>Done</Button>
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
							>{f.types.join(', ')} · {f.last_used_at
								? `last read ${new Date(f.last_used_at).toLocaleString()}`
								: 'not used yet'}</span
						></span
					>
					<Button variant="danger" onclick={() => revoke(f)} aria-label="Revoke {f.name}"
						>Revoke</Button
					>
				</li>
			{/each}
		</ul>
	{/if}
	<form onsubmit={createFeed}>
		<TextField
			id="feed-name"
			label="Feed name"
			placeholder="Phone calendar"
			bind:value={feedName}
			maxlength={100}
			required
		/>
		<fieldset class="filters">
			<legend>Include</legend>
			{#each kinds as k (k.id)}
				<label><input type="checkbox" value={k.id} bind:group={feedTypes} /> {k.label}</label>
			{/each}
		</fieldset>
		<Button type="submit" variant="secondary" disabled={feedTypes.length === 0}>Create feed</Button>
	</form>
</section>

<style>
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
