<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api, message } from '$lib/api';
	import { sample, type Flashcard } from '$lib/flashcards';
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

	function mark(ok: boolean) {
		if (ok) right++;
		i++;
		answer = '';
		revealed = false;
	}
	const q = $derived(questions?.[i] ?? null);
</script>

<svelte:head><title>Practice test · Axon</title></svelte:head>

<p class="crumbs"><a href="/cards?deck={encodeURIComponent(deck)}">← {deck || 'Flashcards'}</a></p>
<h1>Practice test{deck ? `: ${deck}` : ''}</h1>
<LearnTabs />
<p class="muted hint">
	Up to ten random cards. Write your answer before you look: testing yourself is one of the most
	effective ways to learn. A test does not change when cards are due.
</p>

{#if error}
	<Status kind="error" message={error} />
{:else if !questions}
	<Status kind="loading" message="Preparing the test…" />
{:else if questions.length === 0}
	<Status kind="empty" message="This deck has no cards to test yet." />
{:else if q}
	<p class="muted small">Question {i + 1} of {questions.length}</p>
	<section class="card glass" aria-labelledby="test-question">
		<h2 id="test-question">{q.front}</h2>
		<label for="test-answer">Your answer</label>
		<textarea id="test-answer" rows="3" bind:value={answer} disabled={revealed}></textarea>
		{#if !revealed}
			<Button onclick={() => (revealed = true)}>Check</Button>
		{:else}
			<div class="expected">
				<p class="muted small">Expected</p>
				{#if q.has_image}<img
						src="/api/v1/flashcards/{q.id}/image"
						alt="Illustration for: {q.front}"
					/>{/if}
				<p>{q.back}</p>
			</div>
			<div class="row" role="group" aria-label="Did you get it right?">
				<Button onclick={() => mark(true)}>I got it right</Button>
				<Button variant="secondary" onclick={() => mark(false)}>Not yet</Button>
			</div>
		{/if}
	</section>
{:else}
	<section class="card glass" aria-live="polite">
		<h2>Score: {right} of {questions.length}</h2>
		<p>
			{right === questions.length
				? 'Every answer recalled.'
				: 'The ones you missed are the ones worth reviewing next.'}
		</p>
		<div class="row">
			<Button href="/cards/review?deck={encodeURIComponent(deck)}">Review the deck</Button>
			<Button variant="secondary" onclick={() => location.reload()}>Another test</Button>
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
