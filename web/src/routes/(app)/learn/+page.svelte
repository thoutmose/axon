<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { api, message, type Schemas } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { verdict } from '$lib/calibration';

	type Plan = Schemas['LearningPlan'];
	type RecallNote = Schemas['RecallNote'];
	type Calibration =
		paths['/v1/flashcards/calibration']['get']['responses']['200']['content']['application/json'];
	let plan = $state<Plan | null>(null);
	let cal = $state<Calibration | null>(null);
	let error = $state('');

	async function load() {
		error = '';
		try {
			plan = await api<Plan>('/v1/learning/plan');
		} catch (err) {
			error = message(err);
		}
	}
	load();
	api<Calibration>('/v1/flashcards/calibration')
		.then((c) => (cal = c))
		.catch(() => {});

	const pct = (x: number) => `${Math.round(x * 100)}%`;
	const nothing = $derived(
		!!plan &&
			plan.cards.due === 0 &&
			plan.recall.due.length === 0 &&
			plan.recall.suggested.length === 0 &&
			!plan.test.due
	);
	const when = (n: RecallNote) =>
		n.due_on
			? t('last {score} on {date}', {
					score: pct(n.last_score ?? 0),
					date: new Date(n.last_at ?? '').toLocaleDateString(appLocale())
				})
			: t('never recalled');
</script>

<svelte:head><title>{t('Learn · Axon')}</title></svelte:head>

<h1>{t('Learn')}</h1>
<LearnTabs />
<AgentDock {...pageAgents.learn} onapplied={() => load()} />

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !plan}
	<Status kind="loading" message={t('Planning today…')} />
{:else}
	<p class="lead">
		{#if nothing}
			{t('Nothing is due. Rest is part of spacing: what you learned settles better with gaps.')}
		{:else}
			{t('A little every day beats a long session now and then. Here is today.')}
		{/if}
		{#if plan.streak_days > 1}<span class="streak"
				>{t('{streak_days} days in a row', { streak_days: plan.streak_days })}</span
			>{/if}
	</p>

	<div class="grid">
		<section class="card glass" aria-labelledby="cards-title">
			<h2 id="cards-title">{t('Flashcards')}</h2>
			{#if plan.cards.due}
				<p class="big">
					{plan.cards.due}
					<span class="muted"
						>{t('to review · about {minutes} min', { minutes: plan.cards.minutes })}</span
					>
				</p>
				<p class="muted small">
					{t(
						plan.cards.new
							? '{reviews} due again, {fresh} new, all decks mixed.'
							: '{reviews} due again, all decks mixed.',
						{ reviews: plan.cards.reviews, fresh: plan.cards.new }
					)}
				</p>
				<Button href="/cards/review">{t('Start the review')}</Button>
			{:else}
				<p class="muted">
					{t('No cards due. The next ones come back just before you would forget them.')}
				</p>
			{/if}
			{#if plan.cards.reviewed >= 10}
				<p class="small">
					{t('You remembered')} <strong>{pct(plan.cards.retention)}</strong>
					{t('of known cards over 30 days (aim: {aim}).', { aim: pct(plan.cards.target) }) +
						(plan.cards.retention < plan.cards.target - 0.05
							? ' ' +
								t('Below the aim: review a little every day, and make the hardest cards clearer.')
							: '')}
				</p>
			{/if}
			{#if plan.cards.leeches}
				<p class="small">
					<strong>{plan.cards.leeches}</strong>
					{t(plan.cards.leeches === 1 ? 'card keeps slipping.' : 'cards keep slipping.')}
					<a href="/cards/review?practice=1">{t('Practise them')}</a>{t(
						', then rewrite them, add an example or a vivid image, or split them.'
					)}
				</p>
			{/if}
		</section>

		<section class="card glass" aria-labelledby="recall-title">
			<h2 id="recall-title">{t('Recall your notes')}</h2>
			<p class="muted small">
				{t(
					'Write what you remember of a note on a blank page, then see what you missed. Each time you recall it well, the next time comes later.'
				)}
			</p>
			{#if plan.recall.due.length || plan.recall.suggested.length}
				<ul class="rows">
					{#each [...plan.recall.due, ...plan.recall.suggested] as n (n.id)}
						<li>
							<a href="/notes/{n.id}/recall">{n.title}</a>
							<span class="muted small">{n.kind} · {when(n)}</span>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="muted">
					{t(
						'No note to recall today. Literature and permanent notes join a day after you write them.'
					)}
				</p>
			{/if}
		</section>

		<section class="card glass" aria-labelledby="test-title">
			<h2 id="test-title">{t('Weekly test')}</h2>
			{#if plan.test.due}
				<p>
					{t('A short mixed test of every deck: ten questions, answers written before you look.')}
				</p>
				<Button variant="secondary" href="/cards/test">{t('Take the test')}</Button>
			{:else if plan.test.last_at}
				<p class="muted">
					{t('Last test {value}: {value2} right. The next one comes a week later.', {
						value: new Date(plan.test.last_at).toLocaleDateString(appLocale()),
						value2: pct(plan.test.score ?? 0)
					})}
				</p>
			{:else}
				<p class="muted">{t('The first test comes once you have five cards.')}</p>
			{/if}
		</section>

		<section class="card glass" aria-labelledby="notes-title">
			<h2 id="notes-title">{t('Process your notes')}</h2>
			{#if plan.fleeting}
				<p>
					<strong>{plan.fleeting}</strong>
					{t(
						plan.fleeting === 1
							? 'fleeting note is older than two days. Turn each into a literature or permanent note in your own words, or let it go.'
							: 'fleeting notes are older than two days. Turn each into a literature or permanent note in your own words, or let it go.'
					)}
				</p>
				<Button variant="secondary" href="/notes?kind=fleeting">{t('Process them')}</Button>
			{:else}
				<p class="muted">
					{t('No fleeting notes waiting. Capture freely; process within a day or two.')}
				</p>
			{/if}
			<p class="small">
				{t('In a literature or permanent note, mark what to remember:')}
				<code>question :: answer</code>,
				<code>==key term==</code>{t(', or')} <code>**Term**: definition</code>{t(
					'. “Make cards” turns them into flashcards.'
				)}
			</p>
		</section>

		<section class="card glass" aria-labelledby="palace-title">
			<h2 id="palace-title">{t('Memory palaces')}</h2>
			<p class="muted small">
				{t(
					'Picture each thing to remember as a vivid, strange image at a place you know well, then walk through the places in your mind.'
				)}
			</p>
			<Button variant="secondary" href="/learn/palaces"
				>{plan.palaces
					? t(plan.palaces === 1 ? 'Your {n} palace' : 'Your {n} palaces', { n: plan.palaces })
					: t('Build a palace')}</Button
			>
		</section>
	</div>

	{#if cal && cal.reviews > 0}
		{@const v = verdict(cal)}
		<section class="card glass" aria-labelledby="measured-title">
			<h2 id="measured-title">{t('Your memory, measured')}</h2>
			<p>
				{t(
					'Over your last {reviews} reviews of cards you know, Axon predicted you would remember',
					{ reviews: cal.reviews }
				)}
				<strong>{pct(cal.predicted)}</strong>{t('; you remembered')}
				<strong>{pct(cal.remembered)}</strong>{t('. Your target is {value}.', {
					value: pct(cal.target)
				})}
			</p>
			<p class="small">
				{#if v === 'few'}
					{t(
						'It is too early to judge: after 20 reviews, this says whether the schedule fits you.'
					)}
				{:else if v === 'below'}
					{t(
						'Below your target: reviews come a little late for you. Raise the retention in Preferences, or take fewer new cards a day.'
					)}
				{:else if v === 'above'}
					{t(
						'Above your target: you could space reviews further (a lower retention) and spend less time for the same result.'
					)}
				{:else}
					{t('On target: the schedule fits how you remember.')}
				{/if}
			</p>
			{#if v !== 'few'}
				<table class="small">
					<caption class="visually-hidden"
						>{t('Predicted against remembered, by prediction')}</caption
					>
					<thead
						><tr
							><th scope="col">{t('Predicted')}</th><th scope="col">{t('Reviews')}</th><th
								scope="col">{t('Remembered')}</th
							></tr
						></thead
					>
					<tbody>
						{#each cal.buckets as b (b.from)}
							<tr>
								<td>{pct(b.from)}–{pct(b.to)}</td>
								<td>{b.reviews}</td>
								<td>{pct(b.remembered)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</section>
	{/if}

	<details class="card glass why">
		<summary>{t('How Axon helps you remember')}</summary>
		<ul>
			<li>
				<strong>{t('Spaced practice.')}</strong>
				{t(
					'Cards and notes come back just before you would forget them (FSRS for cards, expanding intervals for notes), a few new cards a day.'
				)}
			</li>
			<li>
				<strong>{t('Retrieval practice.')}</strong>
				{t(
					'You recall before you see: the answer to a card, a whole note on a blank page, a weekly test.'
				)}
			</li>
			<li>
				<strong>{t('Interleaving.')}</strong>
				{t('A review of everything mixes decks, so you first work out which idea applies.')}
			</li>
			<li>
				<strong>{t('Elaboration.')}</strong>
				{t('Each card can say why and how it is true; the Teacher writes “why” questions.')}
			</li>
			<li>
				<strong>{t('Concrete examples.')}</strong>
				{t('Each card can carry an example, so the idea is not only abstract.')}
			</li>
			<li>
				<strong>{t('Dual coding.')}</strong>
				{t('Words and a picture of the same idea: add a picture or ask the Teacher for a drawing.')}
			</li>
			<li>
				<strong>{t('Vivid images and memory palaces.')}</strong>
				{t(
					'The stranger the image, the better it sticks; places you know hold them in order (Moonwalking with Einstein).'
				)}
			</li>
			<li>
				<strong>{t('Deliberate practice.')}</strong>
				{t(
					'Cards that keep slipping are named and practised on their own, so you get past the plateau.'
				)}
			</li>
		</ul>
		<p class="muted small">
			{t('After Weinstein, Sumeracki & Caviglioli,')} <em>{t('Understanding How We Learn')}</em>{t(
				', and Joshua Foer,'
			)}
			<em>{t('Moonwalking with Einstein')}</em>.
		</p>
	</details>
{/if}

<style>
	.lead {
		font-size: var(--text-lg);
	}
	.streak {
		margin-left: var(--space-2);
		padding: 0 var(--space-2);
		border-radius: 999px;
		background: var(--accent-soft);
		font-size: var(--text-sm);
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 20rem), 1fr));
		gap: var(--space-3);
		margin-bottom: var(--space-4);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	.card h2 {
		margin-top: 0;
		font-size: var(--text-lg);
	}
	.big {
		font-size: var(--text-xl);
		font-weight: 650;
		margin: 0;
	}
	.big .muted {
		font-size: var(--text-base);
		font-weight: 400;
	}
	.small {
		font-size: var(--text-sm);
	}
	.rows {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-2);
	}
	.rows li {
		display: flex;
		flex-direction: column;
	}
	.why summary {
		cursor: pointer;
		font-weight: 650;
	}
	.why li {
		margin-bottom: var(--space-1);
	}
</style>
