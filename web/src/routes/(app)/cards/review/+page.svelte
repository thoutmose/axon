<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api, message } from '$lib/api';
	import { interval, ratings, type Flashcard } from '$lib/flashcards';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Due = Flashcard & { next: Record<'again' | 'hard' | 'good' | 'easy', string> };
	const deck = $derived(page.url.searchParams.get('deck') ?? '');
	let queue = $state<Due[] | null>(null);
	let total = $state(0);
	let done = $state(0);
	let shown = $state(false);
	let started = Date.now();
	let error = $state('');
	const card = $derived(queue?.[0] ?? null);

	async function load() {
		error = '';
		try {
			const res = await api<{ items: Due[]; due_total: number }>(
				`/v1/flashcards/due?limit=100${deck ? `&deck=${encodeURIComponent(deck)}` : ''}`
			);
			queue = res.items;
			total = Math.max(total, res.items.length + done);
			shown = false;
			started = Date.now();
		} catch (err) {
			error = message(err);
		}
	}
	onMount(load);

	async function rate(value: number) {
		if (!card || !shown) return;
		try {
			await api(`/v1/flashcards/${card.id}/review`, {
				method: 'POST',
				body: { rating: value, duration_ms: Math.min(3_600_000, Date.now() - started) }
			});
			done++;
			queue = queue!.slice(1);
			shown = false;
			started = Date.now();
			// Cards answered "again" come back within the session: fetch what is due now.
			if (queue.length === 0) await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function onkeydown(e: KeyboardEvent) {
		const el = e.target as HTMLElement;
		if (['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName)) return;
		if (!shown && (e.key === ' ' || e.key === 'Enter')) {
			e.preventDefault();
			shown = true;
			return;
		}
		const r = ratings.find((x) => x.key === e.key);
		if (shown && r) {
			e.preventDefault();
			rate(r.value);
		}
	}
</script>

<svelte:window {onkeydown} />
<svelte:head><title>Review · Axon</title></svelte:head>

<p class="crumbs"><a href="/cards">← Flashcards</a></p>
<h1>Review{deck ? `: ${deck}` : ''}</h1>
<LearnTabs />

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !queue}
	<Status kind="loading" message="Loading cards…" />
{:else if !card}
	<section class="done card glass" aria-live="polite">
		<h2>All done for now</h2>
		<p>
			{done}
			{done === 1 ? 'card' : 'cards'} reviewed. The next ones come back just before you would forget them.
		</p>
		<Button href="/cards">Back to decks</Button>
	</section>
{:else}
	<p class="muted small" aria-live="polite">{done} of {total} reviewed · {card.deck}</p>
	<section class="flash card glass" aria-labelledby="question">
		<p class="label muted small">Question</p>
		<h2 id="question">{card.front}</h2>
		{#if shown}
			<hr />
			<p class="label muted small">Answer</p>
			{#if card.has_image}<img
					src="/api/v1/flashcards/{card.id}/image"
					alt="Illustration for: {card.front}"
				/>{/if}
			<p class="answer">{card.back}</p>
		{/if}
	</section>
	{#if !shown}
		<Button onclick={() => (shown = true)}>Show answer <kbd>Space</kbd></Button>
	{:else}
		<div class="ratings" role="group" aria-label="How well did you remember?">
			{#each ratings as r (r.value)}
				<Button variant={r.value === 3 ? 'primary' : 'secondary'} onclick={() => rate(r.value)}>
					{r.label}
					<span class="when">{interval(new Date(), new Date(card.next[r.next]))}</span>
					<kbd>{r.key}</kbd>
				</Button>
			{/each}
		</div>
	{/if}
{/if}

<style>
	.crumbs a {
		color: var(--muted);
		text-decoration: none;
	}
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
	.flash {
		text-align: center;
		min-height: 12rem;
	}
	.flash h2 {
		font-size: var(--text-xl);
		overflow-wrap: anywhere;
	}
	.label {
		text-transform: uppercase;
		letter-spacing: 0.06em;
		margin: 0;
	}
	.answer {
		white-space: pre-wrap;
		font-size: var(--text-lg);
	}
	img {
		max-width: 100%;
		max-height: 18rem;
		border-radius: var(--radius);
	}
	hr {
		border: 0;
		border-top: 1px solid var(--border);
		margin: var(--space-4) 0;
	}
	.ratings {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(8rem, 1fr));
		gap: var(--space-2);
	}
	.when {
		font-size: var(--text-sm);
		opacity: 0.85;
	}
	kbd {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		opacity: 0.7;
		margin-left: var(--space-1);
	}
	.done {
		text-align: center;
	}
</style>
