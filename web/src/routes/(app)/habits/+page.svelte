<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import { habitTemplates } from '$lib/entityTemplates';
	import { evidenceNames, protocols, type Protocol } from '$lib/protocols';
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
	// A protocol fills in the habit form, in the interface's language (EP-118).
	function fromProtocol(p: Protocol) {
		const h = p.habit;
		const tr = (v?: string) => (v ? t(v) : '');
		form = {
			...blank(),
			...h,
			title: tr(h.title),
			identity: tr(h.identity),
			stack_after: tr(h.stack_after),
			intention_place: tr(h.intention_place),
			cue: tr(h.cue),
			craving: tr(h.craving),
			two_minute: tr(h.two_minute)
		};
		document.getElementById('new-title')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
		toast(t('Filled in. Adjust it to your life, then create it.'));
	}
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
					? t('Rest day noted.')
					: status === 'two_minute'
						? t('Small start, counted.')
						: t('Done.')
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

	const labels: Record<string, string> = $derived({
		done: t('Done today'),
		two_minute: t('Two-minute version today'),
		skipped: t('Resting today')
	});
</script>

<svelte:head><title>{t('Habits · Axon')}</title></svelte:head>

<h1>{t('Habits')}</h1>
<AgentDock {...pageAgents.habits} onapplied={() => load()} />
<p class="muted hint">
	{t(
		'Make it obvious, easy, and satisfying. Missing a day is normal; the next small step is what counts.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !habits}
	<Status kind="loading" message={t('Loading habits…')} />
{:else}
	{#if habits.length === 0}
		<Status
			kind="empty"
			message={t(
				'No habits yet. Start with something that takes two minutes, right after something you already do.'
			)}
		/>
	{:else}
		<ul class="habits">
			{#each habits as h (h.id)}
				<li
					class="card"
					class:quiet={!h.state.scheduled_today}
					class:checked={!!h.state.checked_in}
				>
					<div class="top">
						<a href="/habits/{h.id}" class="title">{h.title}</a>
						<span class="muted small">{scheduleLabel(h.days)}</span>
					</div>
					{#if h.kind === 'break'}<span class="badge">{t('Breaking')}</span>{/if}
					{#each plan(h) as line (line)}<p class="plan small">{line}</p>{/each}
					{#if h.identity}
						<p class="muted small">
							{t(
								h.votes === 1 ? '{votes} vote for “{identity}”' : '{votes} votes for “{identity}”',
								{
									votes: h.votes,
									identity: h.identity
								}
							)}
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
								aria-label={t("Undo today's check-in for {title}", { title: h.title })}
								>{t('Undo')}</Button
							>
						</div>
					{:else if h.state.scheduled_today}
						<div class="row">
							<Button
								onclick={() => check(h, 'done')}
								aria-label={t('Done: {title}', { title: h.title })}>{t('Done')}</Button
							>
							{#if h.two_minute}
								<Button
									variant="secondary"
									onclick={() => check(h, 'two_minute')}
									aria-label={t('Did the two-minute version: {two_minute}', {
										two_minute: h.two_minute
									})}>{t('Two minutes: {two_minute}', { two_minute: h.two_minute })}</Button
								>
							{/if}
							<Button
								variant="ghost"
								onclick={() => check(h, 'skipped')}
								aria-label={t('Rest today: {title}', { title: h.title })}>{t('Rest today')}</Button
							>
						</div>
					{:else}
						<p class="muted small">{t('Not scheduled today.')}</p>
					{/if}
					{#if h.state.streak > 1}
						<p class="muted small">
							{t('Kept {streak} scheduled days in a row.', { streak: h.state.streak })}
						</p>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	{#if stacks.length}
		<section class="card" aria-labelledby="stacks-title">
			<h2 id="stacks-title">{t('Your habit stacks')}</h2>
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
		<h2 id="new-title">{t('Design a habit')}</h2>
		<p class="muted small">
			{t(
				"James Clear's four laws: make good habits obvious, attractive, easy, and satisfying; make bad ones invisible, unattractive, difficult, and unsatisfying. Start small, and build the identity, not only the result."
			)}
		</p>
		<TemplateChips
			label={t('Start from a template')}
			items={habitTemplates.map((tpl) => ({
				title: tpl.kind === 'break' ? t('Stop: {title}', { title: t(tpl.title) }) : t(tpl.title),
				hint: t(tpl.identity)
			}))}
			onpick={(i) => {
				form = { ...blank(), ...habitTemplates[i] };
				toast('Filled in. Adjust it to your life, then create it.');
			}}
		/>
		<form onsubmit={create}>
			<fieldset class="kind">
				<legend>{t('I want to')}</legend>
				<label
					><input type="radio" value="build" bind:group={form.kind} />
					{t('build a good habit')}</label
				>
				<label
					><input type="radio" value="break" bind:group={form.kind} />
					{t('break a bad habit')}</label
				>
			</fieldset>
			<TextField
				id="habit-title"
				label={t('Habit')}
				placeholder={form.kind === 'build' ? t('Read one page') : t('Scroll my phone in bed')}
				bind:value={form.title}
				maxlength={500}
				required
			/>
			<TextField
				id="habit-identity"
				label={t('Who do you want to become?')}
				hint={t('Every check-in is a vote for this identity.')}
				placeholder={t('I am a person who reads every day')}
				bind:value={form.identity}
				maxlength={500}
			/>
			<h3>1. {t(laws[form.kind][0].law)}</h3>
			{#if form.kind === 'build'}
				<TextField
					id="habit-stack"
					label={t('After…')}
					hint={t('Habit stacking: “After I [current habit], I will [new habit].”')}
					placeholder={t('After I pour my coffee')}
					bind:value={form.stack_after}
					maxlength={500}
				/>
				<div class="pair">
					<div class="field">
						<label for="habit-time">{t('I will do it at')}</label>
						<input id="habit-time" type="time" bind:value={form.intention_time} />
					</div>
					<TextField
						id="habit-place"
						label={t('In or at')}
						placeholder={t('in the kitchen')}
						bind:value={form.intention_place}
						maxlength={500}
					/>
				</div>
			{/if}
			<TextField
				id="habit-cue"
				label={t(laws[form.kind][0].prompt)}
				bind:value={form.cue}
				maxlength={500}
			/>
			<h3>2. {t(laws[form.kind][1].law)}</h3>
			<TextField
				id="habit-craving"
				label={t(laws[form.kind][1].prompt)}
				bind:value={form.craving}
				maxlength={500}
			/>
			<h3>3. {t(laws[form.kind][2].law)}</h3>
			<TextField
				id="habit-two-minute"
				label={form.kind === 'build' ? t('Two-minute version') : t(laws.break[2].prompt)}
				hint={form.kind === 'build'
					? t('The smallest step that still counts on a hard day.')
					: t('Make the bad habit take more steps.')}
				placeholder={form.kind === 'build'
					? t('Open the book')
					: t('Leave the phone charging in the kitchen')}
				bind:value={form.two_minute}
				maxlength={500}
			/>
			<h3>4. {t(laws[form.kind][3].law)}</h3>
			<TextField
				id="habit-reward"
				label={t(laws[form.kind][3].prompt)}
				bind:value={form.reward}
				maxlength={500}
			/>
			<details>
				<summary>{t('Habit contract (optional)')}</summary>
				<TextField
					id="habit-partner"
					label={t('Accountability partner')}
					hint={t('Axon sends nothing to them; share it yourself.')}
					bind:value={form.contract_partner}
					maxlength={200}
				/>
				<TextField
					id="habit-pledge"
					label={t('Pledge')}
					placeholder={t('If I skip a week, I buy Sam lunch.')}
					bind:value={form.contract_pledge}
					maxlength={2000}
				/>
			</details>
			{#if preview.length}
				<div class="preview" aria-live="polite">
					<strong>{t('Your plan')}</strong>
					{#each preview as line (line)}<p>{line}</p>{/each}
				</div>
			{/if}
			<fieldset class="days">
				<legend>{t('Days')}</legend>
				{#each weekdays as d (d.id)}
					<label
						><input type="checkbox" value={d.id} bind:group={form.days} />
						<span aria-hidden="true">{t(d.short)}</span><span class="visually-hidden"
							>{t(d.long)}</span
						></label
					>
				{/each}
			</fieldset>
			<Button type="submit" variant="secondary" disabled={form.days.length === 0}
				>{t('Create habit')}</Button
			>
		</form>
	</section>

	<section class="card" aria-labelledby="protocols-title">
		<h2 id="protocols-title">{t('Protocols for health, after Andrew Huberman')}</h2>
		<p class="muted small">
			{t(
				'Habits the neuroscientist recommends on the Huberman Lab podcast, with the studies behind them and how strong they are. For information, not medical advice.'
			)}
		</p>
		<ul class="protocols">
			{#each protocols as p (p.id)}
				<li>
					<details>
						<summary
							><span class="ptitle">{t(p.title)}</span>
							<span class="evidence {p.evidence}">{t(evidenceNames[p.evidence])}</span></summary
						>
						<p>{t(p.how)}</p>
						<p class="muted">{t(p.why)}</p>
						{#if p.caution}<p class="caution" role="note">{t(p.caution)}</p>{/if}
						<p class="small">
							{t('Sources:')}
							{#each p.sources as src, i (src.url)}{#if i},
								{/if}<a href={src.url} rel="noopener noreferrer" target="_blank">{src.label}</a
								>{/each}
						</p>
						<Button variant="secondary" onclick={() => fromProtocol(p)}
							>{t('Make it a habit')}</Button
						>
					</details>
				</li>
			{/each}
		</ul>
	</section>

	<section class="card" aria-labelledby="scorecard-title">
		<h2 id="scorecard-title">{t('Habits scorecard')}</h2>
		<p class="muted small">
			{t(
				'List what you do in a day, in order, and mark each behaviour good, neutral, or bad. Seeing it is the first step; nothing here is scored.'
			)}
		</p>
		{#if scorecard.length}
			<ol class="scorecard">
				{#each scorecard as item, i (i)}
					<li>
						<span class="grow">{item.behavior}</span>
						<span
							class="ratings"
							role="group"
							aria-label={t('Rating for {behavior}', { behavior: item.behavior })}
						>
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
							aria-label={t('Remove {behavior}', { behavior: item.behavior })}>✕</Button
						>
					</li>
				{/each}
			</ol>
		{/if}
		<form class="add-behavior" onsubmit={addBehavior}>
			<TextField
				id="scorecard-behavior"
				label={t('Add a behaviour')}
				placeholder={t('Wake up, check my phone, make coffee…')}
				bind:value={behavior}
				maxlength={200}
			/>
			<Button type="submit" variant="secondary">{t('Add')}</Button>
		</form>
	</section>
{/if}

<style>
	.protocols {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.protocols li {
		border-bottom: 1px solid var(--border);
		padding: var(--space-2) 0;
	}
	.protocols summary {
		cursor: pointer;
		display: flex;
		gap: var(--space-2);
		align-items: baseline;
		flex-wrap: wrap;
	}
	.ptitle {
		font-weight: 600;
	}
	.evidence {
		font-size: var(--text-sm);
		border-radius: 999px;
		padding: 0 var(--space-2);
		border: 1px solid var(--border);
	}
	.evidence.strong {
		color: var(--ok);
	}
	.evidence.debated {
		color: var(--muted);
	}
	.caution {
		color: var(--bad);
		font-size: var(--text-sm);
	}
	/* Checked today: crossed out, until tomorrow (EP-116). */
	.checked .title {
		color: var(--muted);
		text-decoration: line-through;
	}
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
