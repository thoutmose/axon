<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import { byDay, monthLabel, monthRange, shiftMonth } from '$lib/calendar';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Event = Schemas['CalendarEvent'];
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

	const thisMonth = new Date().toISOString().slice(0, 7);
	const month = $derived(page.url.searchParams.get('month') ?? thisMonth);
	const shown = $derived(
		(page.url.searchParams.get('types') ?? 'task,goal,milestone').split(',') as Kind[]
	);
	let events = $state<Event[] | null>(null);
	let error = $state('');
	let feeds = $state<Feed[]>([]);
	let feedName = $state('');
	let feedTypes = $state<Kind[]>(['task', 'milestone']);
	let newFeedUrl = $state('');

	async function load() {
		error = '';
		const { from, to } = monthRange(month);
		try {
			const params = new URLSearchParams({ from, to, types: shown.join(',') });
			events = shown.length ? (await api<{ items: Event[] }>(`/v1/calendar?${params}`)).items : [];
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
		events = null;
		load();
	});
	$effect(() => {
		loadFeeds();
	});

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
		if (e.type === 'goal') return `/goals/${e.id}`;
		if (e.type === 'milestone' && e.goal_id) return `/goals/${e.goal_id}`;
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
		<a href="?month={shiftMonth(month, -1)}&types={shown.join(',')}" aria-label="Previous month"
			>←</a
		>
		<h2 aria-live="polite">{monthLabel(month)}</h2>
		<a href="?month={shiftMonth(month, 1)}&types={shown.join(',')}" aria-label="Next month">→</a>
		{#if month !== thisMonth}<a href="?types={shown.join(',')}" class="small">This month</a>{/if}
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
	</fieldset>
</div>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !events}
	<Status kind="loading" message="Loading the calendar…" />
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
							<span class="chip {e.type}">{prefix[e.kind] || 'Task'}</span>
							{#if href(e)}<a href={href(e)}>{e.title}</a>{:else}<span>{e.title}</span>{/if}
							{#if e.done}<span class="visually-hidden">(done)</span>{/if}
						</li>
					{/each}
				</ul>
			</li>
		{/each}
	</ol>
{/if}

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
