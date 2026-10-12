<script lang="ts">
	import { api, message, type Schemas } from '$lib/api';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';

	type Plan = Schemas['LearningPlan'];
	type RecallNote = Schemas['RecallNote'];
	let plan = $state<Plan | null>(null);
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
			? `last ${pct(n.last_score ?? 0)} on ${new Date(n.last_at ?? '').toLocaleDateString()}`
			: 'never recalled';
</script>

<svelte:head><title>Learn · Axon</title></svelte:head>

<h1>Learn</h1>
<LearnTabs />

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !plan}
	<Status kind="loading" message="Planning today…" />
{:else}
	<p class="lead">
		{#if nothing}
			Nothing is due. Rest is part of spacing: what you learned settles better with gaps.
		{:else}
			A little every day beats a long session now and then. Here is today.
		{/if}
		{#if plan.streak_days > 1}<span class="streak">{plan.streak_days} days in a row</span>{/if}
	</p>

	<div class="grid">
		<section class="card glass" aria-labelledby="cards-title">
			<h2 id="cards-title">Flashcards</h2>
			{#if plan.cards.due}
				<p class="big">
					{plan.cards.due} <span class="muted">to review · about {plan.cards.minutes} min</span>
				</p>
				<p class="muted small">
					{plan.cards.reviews} due again{plan.cards.new ? `, ${plan.cards.new} new` : ''}, all decks
					mixed.
				</p>
				<Button href="/cards/review">Start the review</Button>
			{:else}
				<p class="muted">
					No cards due. The next ones come back just before you would forget them.
				</p>
			{/if}
			{#if plan.cards.reviewed >= 10}
				<p class="small">
					You remembered <strong>{pct(plan.cards.retention)}</strong> of known cards over 30 days
					(aim: {pct(plan.cards.target)}).
					{plan.cards.retention < plan.cards.target - 0.05
						? 'Below the aim: review a little every day, and make the hardest cards clearer.'
						: ''}
				</p>
			{/if}
			{#if plan.cards.leeches}
				<p class="small">
					<strong>{plan.cards.leeches}</strong>
					{plan.cards.leeches === 1 ? 'card keeps' : 'cards keep'} slipping.
					<a href="/cards/review?practice=1">Practise them</a>, then rewrite them, add an example or
					a vivid image, or split them.
				</p>
			{/if}
		</section>

		<section class="card glass" aria-labelledby="recall-title">
			<h2 id="recall-title">Recall your notes</h2>
			<p class="muted small">
				Write what you remember of a note on a blank page, then see what you missed. Each time you
				recall it well, the next time comes later.
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
					No note to recall today. Literature and permanent notes join a day after you write them.
				</p>
			{/if}
		</section>

		<section class="card glass" aria-labelledby="test-title">
			<h2 id="test-title">Weekly test</h2>
			{#if plan.test.due}
				<p>A short mixed test of every deck: ten questions, answers written before you look.</p>
				<Button variant="secondary" href="/cards/test">Take the test</Button>
			{:else if plan.test.last_at}
				<p class="muted">
					Last test {new Date(plan.test.last_at).toLocaleDateString()}: {pct(plan.test.score ?? 0)} right.
					The next one comes a week later.
				</p>
			{:else}
				<p class="muted">The first test comes once you have five cards.</p>
			{/if}
		</section>

		<section class="card glass" aria-labelledby="notes-title">
			<h2 id="notes-title">Process your notes</h2>
			{#if plan.fleeting}
				<p>
					<strong>{plan.fleeting}</strong>
					fleeting {plan.fleeting === 1 ? 'note is' : 'notes are'} older than two days. Turn each into
					a literature or permanent note in your own words, or let it go.
				</p>
				<Button variant="secondary" href="/notes?kind=fleeting">Process them</Button>
			{:else}
				<p class="muted">No fleeting notes waiting. Capture freely; process within a day or two.</p>
			{/if}
			<p class="small">
				In a literature or permanent note, mark what to remember: <code>question :: answer</code>,
				<code>==key term==</code>, or <code>**Term**: definition</code>. “Make cards” turns them
				into flashcards.
			</p>
		</section>

		<section class="card glass" aria-labelledby="palace-title">
			<h2 id="palace-title">Memory palaces</h2>
			<p class="muted small">
				Picture each thing to remember as a vivid, strange image at a place you know well, then walk
				through the places in your mind.
			</p>
			<Button variant="secondary" href="/learn/palaces"
				>{plan.palaces
					? `Your ${plan.palaces} ${plan.palaces === 1 ? 'palace' : 'palaces'}`
					: 'Build a palace'}</Button
			>
		</section>
	</div>

	<details class="card glass why">
		<summary>How Axon helps you remember</summary>
		<ul>
			<li>
				<strong>Spaced practice.</strong> Cards and notes come back just before you would forget them
				(FSRS for cards, expanding intervals for notes), a few new cards a day.
			</li>
			<li>
				<strong>Retrieval practice.</strong> You recall before you see: the answer to a card, a whole
				note on a blank page, a weekly test.
			</li>
			<li>
				<strong>Interleaving.</strong> A review of everything mixes decks, so you first work out which
				idea applies.
			</li>
			<li>
				<strong>Elaboration.</strong> Each card can say why and how it is true; the Teacher writes “why”
				questions.
			</li>
			<li>
				<strong>Concrete examples.</strong> Each card can carry an example, so the idea is not only abstract.
			</li>
			<li>
				<strong>Dual coding.</strong> Words and a picture of the same idea: add a picture or ask the Teacher
				for a drawing.
			</li>
			<li>
				<strong>Vivid images and memory palaces.</strong> The stranger the image, the better it sticks;
				places you know hold them in order (Moonwalking with Einstein).
			</li>
			<li>
				<strong>Deliberate practice.</strong> Cards that keep slipping are named and practised on their
				own, so you get past the plateau.
			</li>
		</ul>
		<p class="muted small">
			After Weinstein, Sumeracki & Caviglioli, <em>Understanding How We Learn</em>, and Joshua Foer,
			<em>Moonwalking with Einstein</em>.
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
