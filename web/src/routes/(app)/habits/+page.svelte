<script lang="ts">
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import { habitTemplates } from '$lib/entityTemplates';
	import { chains, laws, plan, scheduleLabel, weekdays, type Habit } from '$lib/habits';
	import TemplateChips from '$lib/TemplateChips.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let habits = $state<Habit[] | null>(null);
	let error = $state('');
	const blank = () => ({
		title: '',
		kind: 'build' as 'build' | 'break',
		identity: '',
		stack_after: '',
		intention_time: '',
		intention_place: '',
		cue: '',
		craving: '',
		two_minute: '',
		reward: '',
		contract_partner: '',
		contract_pledge: '',
		days: [1, 2, 3, 4, 5, 6, 7]
	});
	let form = $state(blank());
	const preview = $derived(form.title ? plan(form as unknown as Habit) : []);
	const stacks = $derived(habits ? chains(habits) : []);

	type Item = { behavior: string; rating: '+' | '=' | '-' };
	let scorecard = $state<Item[]>([]);
	let behavior = $state('');
	api<{ items: Item[] }>('/v1/habits/scorecard')
		.then((s) => (scorecard = s.items))
		.catch(() => {});
	async function saveScorecard(next: Item[]) {
		try {
			scorecard = (
				await api<{ items: Item[] }>('/v1/habits/scorecard', {
					method: 'PUT',
					body: { items: next }
				})
			).items;
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	function addBehavior(e: SubmitEvent) {
		e.preventDefault();
		if (!behavior.trim()) return;
		saveScorecard([...scorecard, { behavior, rating: '=' }]);
		behavior = '';
	}
	const ratingNames = { '+': 'good', '=': 'neutral', '-': 'bad' } as const;

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
			form = blank();
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
<AgentDock {...pageAgents.habits} onapplied={() => load()} />
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
					{#if h.kind === 'break'}<span class="badge">Breaking</span>{/if}
					{#each plan(h) as line (line)}<p class="plan small">{line}</p>{/each}
					{#if h.identity}
						<p class="muted small">
							{h.votes}
							{h.votes === 1 ? 'vote' : 'votes'} for “{h.identity}”
						</p>
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

	{#if stacks.length}
		<section class="card" aria-labelledby="stacks-title">
			<h2 id="stacks-title">Your habit stacks</h2>
			{#each stacks as chain (chain[0].id)}
				<ol class="chain">
					{#each chain as link (link.id)}<li>
							<a href="/habits/{link.id}">{link.title}</a>
						</li>{/each}
				</ol>
			{/each}
		</section>
	{/if}

	<section class="card" aria-labelledby="new-title">
		<h2 id="new-title">Design a habit</h2>
		<p class="muted small">
			James Clear's four laws: make good habits obvious, attractive, easy, and satisfying; make bad
			ones invisible, unattractive, difficult, and unsatisfying. Start small, and build the
			identity, not only the result.
		</p>
		<TemplateChips
			label="Start from a template"
			items={habitTemplates.map((t) => ({
				title: t.kind === 'break' ? `Stop: ${t.title}` : t.title,
				hint: t.identity
			}))}
			onpick={(i) => {
				form = { ...blank(), ...habitTemplates[i] };
				toast('Filled in. Adjust it to your life, then create it.');
			}}
		/>
		<form onsubmit={create}>
			<fieldset class="kind">
				<legend>I want to</legend>
				<label><input type="radio" value="build" bind:group={form.kind} /> build a good habit</label
				>
				<label><input type="radio" value="break" bind:group={form.kind} /> break a bad habit</label>
			</fieldset>
			<TextField
				id="habit-title"
				label="Habit"
				placeholder={form.kind === 'build' ? 'Read one page' : 'Scroll my phone in bed'}
				bind:value={form.title}
				maxlength={500}
				required
			/>
			<TextField
				id="habit-identity"
				label="Who do you want to become?"
				hint="Every check-in is a vote for this identity."
				placeholder="I am a person who reads every day"
				bind:value={form.identity}
				maxlength={500}
			/>
			<h3>1. {laws[form.kind][0].law}</h3>
			{#if form.kind === 'build'}
				<TextField
					id="habit-stack"
					label="After…"
					hint="Habit stacking: “After I [current habit], I will [new habit].”"
					placeholder="After I pour my coffee"
					bind:value={form.stack_after}
					maxlength={500}
				/>
				<div class="pair">
					<div class="field">
						<label for="habit-time">I will do it at</label>
						<input id="habit-time" type="time" bind:value={form.intention_time} />
					</div>
					<TextField
						id="habit-place"
						label="In or at"
						placeholder="in the kitchen"
						bind:value={form.intention_place}
						maxlength={500}
					/>
				</div>
			{/if}
			<TextField
				id="habit-cue"
				label={laws[form.kind][0].prompt}
				bind:value={form.cue}
				maxlength={500}
			/>
			<h3>2. {laws[form.kind][1].law}</h3>
			<TextField
				id="habit-craving"
				label={laws[form.kind][1].prompt}
				bind:value={form.craving}
				maxlength={500}
			/>
			<h3>3. {laws[form.kind][2].law}</h3>
			<TextField
				id="habit-two-minute"
				label={form.kind === 'build' ? 'Two-minute version' : laws.break[2].prompt}
				hint={form.kind === 'build'
					? 'The smallest step that still counts on a hard day.'
					: 'Make the bad habit take more steps.'}
				placeholder={form.kind === 'build'
					? 'Open the book'
					: 'Leave the phone charging in the kitchen'}
				bind:value={form.two_minute}
				maxlength={500}
			/>
			<h3>4. {laws[form.kind][3].law}</h3>
			<TextField
				id="habit-reward"
				label={laws[form.kind][3].prompt}
				bind:value={form.reward}
				maxlength={500}
			/>
			<details>
				<summary>Habit contract (optional)</summary>
				<TextField
					id="habit-partner"
					label="Accountability partner"
					hint="Axon sends nothing to them; share it yourself."
					bind:value={form.contract_partner}
					maxlength={200}
				/>
				<TextField
					id="habit-pledge"
					label="Pledge"
					placeholder="If I skip a week, I buy Sam lunch."
					bind:value={form.contract_pledge}
					maxlength={2000}
				/>
			</details>
			{#if preview.length}
				<div class="preview" aria-live="polite">
					<strong>Your plan</strong>
					{#each preview as line (line)}<p>{line}</p>{/each}
				</div>
			{/if}
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

	<section class="card" aria-labelledby="scorecard-title">
		<h2 id="scorecard-title">Habits scorecard</h2>
		<p class="muted small">
			List what you do in a day, in order, and mark each behaviour good, neutral, or bad. Seeing it
			is the first step; nothing here is scored.
		</p>
		{#if scorecard.length}
			<ol class="scorecard">
				{#each scorecard as item, i (i)}
					<li>
						<span class="grow">{item.behavior}</span>
						<span class="ratings" role="group" aria-label="Rating for {item.behavior}">
							{#each ['+', '=', '-'] as const as r (r)}
								<button
									type="button"
									class="rate r{r === '+' ? 'good' : r === '-' ? 'bad' : 'neutral'}"
									aria-pressed={item.rating === r}
									aria-label="{ratingNames[r]}: {item.behavior}"
									onclick={() =>
										saveScorecard(scorecard.map((x, j) => (j === i ? { ...x, rating: r } : x)))}
									>{r === '-' ? '−' : r}</button
								>
							{/each}
						</span>
						<Button
							variant="ghost"
							onclick={() => saveScorecard(scorecard.filter((_, j) => j !== i))}
							aria-label="Remove {item.behavior}">✕</Button
						>
					</li>
				{/each}
			</ol>
		{/if}
		<form class="add-behavior" onsubmit={addBehavior}>
			<TextField
				id="scorecard-behavior"
				label="Add a behaviour"
				placeholder="Wake up, check my phone, make coffee…"
				bind:value={behavior}
				maxlength={200}
			/>
			<Button type="submit" variant="secondary">Add</Button>
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
	.badge {
		font-size: var(--text-sm);
		background: var(--bad-soft);
		color: var(--bad);
		border-radius: 999px;
		padding: 0 var(--space-2);
	}
	.plan {
		margin: var(--space-1) 0;
		font-style: italic;
	}
	.kind {
		border: 0;
		padding: 0;
		margin: 0 0 var(--space-3);
		display: flex;
		gap: var(--space-4);
		flex-wrap: wrap;
	}
	.kind legend {
		font-weight: 600;
		margin-bottom: var(--space-1);
	}
	h3 {
		font-size: var(--text-base);
		margin: var(--space-4) 0 var(--space-2);
	}
	.pair {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr));
		gap: var(--space-3);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
	}
	.field label {
		font-weight: 600;
	}
	.field input {
		font: inherit;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	details {
		margin: var(--space-3) 0;
	}
	summary {
		cursor: pointer;
		color: var(--accent);
	}
	.preview {
		border-left: 3px solid var(--accent);
		padding: var(--space-2) var(--space-3);
		margin: var(--space-3) 0;
		background: var(--accent-soft);
		border-radius: var(--radius-sm);
	}
	.preview p {
		margin: var(--space-1) 0 0;
	}
	.chain {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		list-style: none;
		padding: 0;
	}
	.chain li:not(:last-child)::after {
		content: ' →';
		color: var(--muted);
	}
	.scorecard {
		padding-left: var(--space-6);
	}
	.scorecard li {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-1) 0;
		border-bottom: 1px solid var(--border);
	}
	.grow {
		flex: 1;
	}
	.ratings {
		display: flex;
		gap: var(--space-1);
	}
	.rate {
		font: inherit;
		min-width: 2.25rem;
		min-height: 2.25rem;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		cursor: pointer;
	}
	.rate[aria-pressed='true'].rgood {
		background: var(--ok);
		color: var(--bg);
	}
	.rate[aria-pressed='true'].rneutral {
		background: var(--muted);
		color: var(--bg);
	}
	.rate[aria-pressed='true'].rbad {
		background: var(--bad);
		color: var(--bg);
	}
	.add-behavior {
		display: flex;
		gap: var(--space-2);
		align-items: flex-end;
		flex-wrap: wrap;
	}
</style>
