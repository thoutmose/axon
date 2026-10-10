<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import { scheduleLabel, weekdays, type Habit } from '$lib/habits';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let habits = $state<Habit[] | null>(null);
	let error = $state('');
	let form = $state({ title: '', two_minute: '', stack_after: '', days: [1, 2, 3, 4, 5, 6, 7] });

	async function load() {
		error = '';
		try {
			habits = (await api<{ items: Habit[] }>('/v1/habits')).items;
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function create(e: SubmitEvent) {
		e.preventDefault();
		try {
			const h = await api<Habit>('/v1/habits', { method: 'POST', body: form });
			form = { title: '', two_minute: '', stack_after: '', days: [1, 2, 3, 4, 5, 6, 7] };
			await goto(`/habits/${h.id}`);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function check(h: Habit, status: 'done' | 'two_minute' | 'skipped') {
		try {
			await api(`/v1/habits/${h.id}/checkins/${h.state.today}`, {
				method: 'PUT',
				body: { status }
			});
			toast(
				status === 'skipped'
					? 'Rest day noted.'
					: status === 'two_minute'
						? 'Small start, counted.'
						: 'Done.'
			);
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function undo(h: Habit) {
		try {
			await api(`/v1/habits/${h.id}/checkins/${h.state.today}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const labels: Record<string, string> = {
		done: 'Done today',
		two_minute: 'Two-minute version today',
		skipped: 'Resting today'
	};
</script>

<svelte:head><title>Habits · Axon</title></svelte:head>

<h1>Habits</h1>
<p class="muted hint">
	Make it obvious, easy, and satisfying. Missing a day is normal; the next small step is what
	counts.
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !habits}
	<Status kind="loading" message="Loading habits…" />
{:else}
	{#if habits.length === 0}
		<Status
			kind="empty"
			message="No habits yet. Start with something that takes two minutes, right after something you already do."
		/>
	{:else}
		<ul class="habits">
			{#each habits as h (h.id)}
				<li class="card" class:quiet={!h.state.scheduled_today}>
					<div class="top">
						<a href="/habits/{h.id}" class="title">{h.title}</a>
						<span class="muted small">{scheduleLabel(h.days)}</span>
					</div>
					{#if h.stack_after || h.cue}
						<p class="muted small">{[h.stack_after, h.cue].filter(Boolean).join(' · ')}</p>
					{/if}
					{#if h.state.recovery}
						<p class="recovery" role="note">{h.state.recovery.message}</p>
					{/if}
					{#if h.state.checked_in}
						<div class="row">
							<span class="done">✓ {labels[h.state.checked_in]}</span>
							<Button
								variant="ghost"
								onclick={() => undo(h)}
								aria-label="Undo today's check-in for {h.title}">Undo</Button
							>
						</div>
					{:else if h.state.scheduled_today}
						<div class="row">
							<Button onclick={() => check(h, 'done')} aria-label="Done: {h.title}">Done</Button>
							{#if h.two_minute}
								<Button
									variant="secondary"
									onclick={() => check(h, 'two_minute')}
									aria-label="Did the two-minute version: {h.two_minute}"
									>Two minutes: {h.two_minute}</Button
								>
							{/if}
							<Button
								variant="ghost"
								onclick={() => check(h, 'skipped')}
								aria-label="Rest today: {h.title}">Rest today</Button
							>
						</div>
					{:else}
						<p class="muted small">Not scheduled today.</p>
					{/if}
					{#if h.state.streak > 1}
						<p class="muted small">Kept {h.state.streak} scheduled days in a row.</p>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	<section class="card" aria-labelledby="new-title">
		<h2 id="new-title">New habit</h2>
		<form onsubmit={create}>
			<TextField
				id="habit-title"
				label="Habit"
				placeholder="Read one page"
				bind:value={form.title}
				maxlength={500}
				required
			/>
			<TextField
				id="habit-two-minute"
				label="Two-minute version"
				hint="The smallest step that still counts on a hard day."
				placeholder="Open the book"
				bind:value={form.two_minute}
				maxlength={500}
			/>
			<TextField
				id="habit-stack"
				label="After…"
				hint="Stack it after something you already do every day."
				placeholder="After I pour my coffee"
				bind:value={form.stack_after}
				maxlength={500}
			/>
			<fieldset class="days">
				<legend>Days</legend>
				{#each weekdays as d (d.id)}
					<label
						><input type="checkbox" value={d.id} bind:group={form.days} />
						<span aria-hidden="true">{d.short}</span><span class="visually-hidden">{d.long}</span
						></label
					>
				{/each}
			</fieldset>
			<Button type="submit" variant="secondary" disabled={form.days.length === 0}
				>Create habit</Button
			>
		</form>
	</section>
{/if}

<style>
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.habits {
		list-style: none;
		padding: 0;
		margin: var(--space-4) 0 var(--space-6);
		display: grid;
		gap: var(--space-3);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	section.card {
		padding: var(--space-4) var(--space-6) var(--space-6);
	}
	.quiet {
		opacity: 0.85;
	}
	.top {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	.title {
		font-weight: 600;
		color: var(--fg);
		text-decoration: none;
		overflow-wrap: anywhere;
	}
	.title:hover {
		color: var(--accent);
	}
	p {
		margin: var(--space-2) 0;
	}
	.recovery {
		background: var(--accent-soft);
		border-radius: var(--radius);
		padding: var(--space-2) var(--space-3);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		flex-wrap: wrap;
		margin-top: var(--space-2);
	}
	.done {
		color: var(--ok);
		font-weight: 600;
	}
	h2 {
		font-size: var(--text-lg);
	}
	.days {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		border: none;
		padding: 0;
		margin: 0 0 var(--space-4);
	}
	.days legend {
		font-weight: 600;
		margin-bottom: var(--space-2);
	}
	.days label {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}
	.days input {
		width: 1.1rem;
		height: 1.1rem;
		accent-color: var(--accent);
	}
</style>
