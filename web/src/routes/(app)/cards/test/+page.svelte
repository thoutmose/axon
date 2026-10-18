<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api, message } from '$lib/api';
	import { deckName, sample, type Flashcard } from '$lib/flashcards';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';

	const deck = $derived(page.url.searchParams.get('deck') ?? '');
	let questions = $state<Flashcard[] | null>(null);
	let i = $state(0);
	let answer = $state('');
	let revealed = $state(false);
	let right = $state(0);
	let error = $state('');

	onMount(async () => {
		try {
			const all = (
				await api<{ items: Flashcard[] }>(`/v1/flashcards?deck=${encodeURIComponent(deck)}`)
			).items;
			questions = sample(
				all.filter((c) => !c.suspended),
				10
			);
		} catch (err) {
			error = message(err);
		}
	});

	let missed = $state<Flashcard[]>([]);
	function mark(ok: boolean) {
		if (ok) right++;
		else if (q) missed = [...missed, q];
		i++;
		answer = '';
		revealed = false;
		// The result feeds the plan, which offers the next test a week later.
		if (questions && i === questions.length) {
			api('/v1/learning/tests', {
				method: 'POST',
				body: { deck, questions: questions.length, correct: right }
			}).catch(() => {});
		}
	}
	const q = $derived(questions?.[i] ?? null);
</script>

<svelte:head><title>{t('Practice test · Axon')}</title></svelte:head>

<p class="crumbs">
	<a href="/cards?deck={encodeURIComponent(deck)}">← {deck || t('Flashcards')}</a>
</p>
<h1>{deck ? t('Practice test: {deck}', { deck: deckName(deck) }) : t('Practice test')}</h1>
<LearnTabs />
<p class="muted hint">
	{t(
		deck
			? 'Up to ten random cards. Write your answer before you look: testing yourself is one of the most effective ways to learn, and getting some wrong is part of it. A test does not change when cards are due.'
			: 'Up to ten random cards, from every deck, mixed. Write your answer before you look: testing yourself is one of the most effective ways to learn, and getting some wrong is part of it. A test does not change when cards are due.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} />
{:else if !questions}
	<Status kind="loading" message={t('Preparing the test…')} />
{:else if questions.length === 0}
	<Status kind="empty" message={t('This deck has no cards to test yet.')} />
{:else if q}
	<p class="muted small">
		{t('Question {value} of {length}', { value: i + 1, length: questions.length })}
	</p>
	<section class="card glass" aria-labelledby="test-question">
		<h2 id="test-question">{q.front}</h2>
		<label for="test-answer">{t('Your answer')}</label>
		<textarea id="test-answer" rows="3" bind:value={answer} disabled={revealed}></textarea>
		{#if !revealed}
			<Button onclick={() => (revealed = true)}>{t('Check')}</Button>
		{:else}
			<div class="expected">
				<p class="muted small">{t('Expected')}</p>
				{#if q.has_image}<img
						src="/api/v1/flashcards/{q.id}/image"
						alt={t('Illustration for: {front}', { front: q.front })}
					/>{/if}
				<p>{q.back}</p>
			</div>
			<div class="row" role="group" aria-label={t('Did you get it right?')}>
				<Button onclick={() => mark(true)}>{t('I got it right')}</Button>
				<Button variant="secondary" onclick={() => mark(false)}>{t('Not yet')}</Button>
			</div>
		{/if}
	</section>
{:else}
	<section class="card glass" aria-live="polite">
		<h2>{t('Score: {right} of {length}', { right, length: questions.length })}</h2>
		<p>
			{right === questions.length
				? t('Every answer recalled.')
				: t('The ones you missed are the ones worth reviewing next.')}
		</p>
		{#if missed.length}
			<ul class="missed">
				{#each missed as m (m.id)}<li><strong>{m.front}</strong> — {m.back}</li>{/each}
			</ul>
		{/if}
		<div class="row">
			<Button href={deck ? `/cards/review?deck=${encodeURIComponent(deck)}` : '/learn'}
				>{deck ? t('Review the deck') : t('Today’s plan')}</Button
			>
			<Button variant="secondary" onclick={() => location.reload()}>{t('Another test')}</Button>
		</div>
	</section>
{/if}

<style>
	.crumbs a {
		color: var(--muted);
		text-decoration: none;
	}
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-6);
		margin-bottom: var(--space-4);
	}
	label {
		display: block;
		font-weight: 600;
		margin-bottom: var(--space-1);
	}
	textarea {
		font: inherit;
		width: 100%;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		margin-bottom: var(--space-3);
	}
	.expected {
		border-left: 3px solid var(--ok);
		padding-left: var(--space-3);
		margin-bottom: var(--space-3);
	}
	img {
		max-width: 100%;
		max-height: 14rem;
		border-radius: var(--radius);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
</style>
