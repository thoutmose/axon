<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api, message } from '$lib/api';
	import { interval, ratings, type Flashcard } from '$lib/flashcards';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Due = Flashcard & { next?: Record<'again' | 'hard' | 'good' | 'easy', string> };
	const deck = $derived(page.url.searchParams.get('deck') ?? '');
	// Deliberate practice: the weakest cards, until each is recalled; the
	// schedule is not touched.
	const practice = $derived(page.url.searchParams.get('practice') === '1');
	let queue = $state<Due[] | null>(null);
	let total = $state(0);
	let done = $state(0);
	let shown = $state(false);
	let typeAnswer = $state(false);
	let attempt = $state('');
	let started = Date.now();
	let error = $state('');
	const card = $derived(queue?.[0] ?? null);

	async function load() {
		error = '';
		try {
			if (practice) {
				queue = (await api<{ items: Due[] }>('/v1/flashcards/practice?limit=20')).items;
				total = queue.length;
			} else {
				const res = await api<{ items: Due[]; due_total: number }>(
					`/v1/flashcards/due?limit=100${deck ? `&deck=${encodeURIComponent(deck)}` : ''}`
				);
				queue = res.items;
				total = Math.max(total, res.items.length + done);
			}
			shown = false;
			attempt = '';
			started = Date.now();
		} catch (err) {
			error = message(err);
		}
	}
	onMount(() => {
		api<{ learning?: { type_answer: boolean } }>('/v1/me/settings')
			.then((s) => (typeAnswer = s.learning?.type_answer ?? false))
			.catch(() => {});
		load();
	});

	function next() {
		shown = false;
		attempt = '';
		started = Date.now();
	}

	async function rate(value: number) {
		if (!card || !shown) return;
		try {
			await api(`/v1/flashcards/${card.id}/review`, {
				method: 'POST',
				body: { rating: value, duration_ms: Math.min(3_600_000, Date.now() - started) }
			});
			done++;
			queue = queue!.slice(1);
			next();
			// Cards answered "again" come back within the session: fetch what is due now.
			if (queue.length === 0) await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	/** Practice: a missed card goes to the back of the queue until it is recalled. */
	function mark(recalled: boolean) {
		if (!card || !shown) return;
		queue = recalled ? queue!.slice(1) : [...queue!.slice(1), card];
		if (recalled) done++;
		next();
	}

	function onkeydown(e: KeyboardEvent) {
		const el = e.target as HTMLElement;
		if (!shown && el.tagName === 'TEXTAREA' && e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
			e.preventDefault();
			shown = true;
			return;
		}
		if (['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName)) return;
		if (!shown && (e.key === ' ' || e.key === 'Enter')) {
			e.preventDefault();
			shown = true;
			return;
		}
		if (shown && practice && (e.key === '1' || e.key === '2')) {
			e.preventDefault();
			mark(e.key === '2');
			return;
		}
		const r = ratings.find((x) => x.key === e.key);
		if (shown && !practice && r) {
			e.preventDefault();
			rate(r.value);
		}
	}
</script>

<svelte:window {onkeydown} />
<svelte:head><title>{t(practice ? 'Practice' : 'Review')} · Axon</title></svelte:head>

<p class="crumbs"><a href="/cards">{t('← Flashcards')}</a></p>
<h1>
	{practice ? t('Practise weak cards') : deck ? t('Review: {deck}', { deck }) : t('Review')}
</h1>
<LearnTabs />
{#if practice}
	<p class="muted small">
		{t(
			'Deliberate practice: the cards you forget most and those closest to being forgotten. A missed card comes back until you get it. This does not change when cards are due.'
		)}
	</p>
{:else if !deck}
	<p class="muted small">
		{t('All decks, mixed: working out which idea applies is part of learning.')}
	</p>
{/if}

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !queue}
	<Status kind="loading" message={t('Loading cards…')} />
{:else if !card}
	<section class="done card glass" aria-live="polite">
		<h2>{t('All done for now')}</h2>
		<p>
			{done}
			{done === 1 ? 'card' : 'cards'}
			{practice
				? t('practised. Practice pays when it is hard: come back tomorrow.')
				: t('reviewed. The next ones come back just before you would forget them.')}
		</p>
		<div class="row">
			<Button href="/learn">{t('Today’s plan')}</Button>
			<Button variant="secondary" href="/cards">{t('Back to decks')}</Button>
		</div>
	</section>
{:else}
	<p class="muted small" aria-live="polite">
		{t(practice ? '{done} of {total} practised · {deck}' : '{done} of {total} reviewed · {deck}', {
			done,
			total,
			deck: card.deck
		})}
	</p>
	<section class="flash card glass" aria-labelledby="question">
		<p class="label muted small">{t('Question')}</p>
		<h2 id="question">{card.front}</h2>
		{#if !shown && typeAnswer}
			<label class="attempt" for="attempt"
				>{t('Write your answer first')}
				<span class="muted small">{t('(Ctrl+Enter shows the answer)')}</span></label
			>
			<textarea id="attempt" rows="3" bind:value={attempt}></textarea>
		{/if}
		{#if shown}
			<hr />
			{#if attempt.trim()}
				<p class="label muted small">{t('Your answer')}</p>
				<p class="mine">{attempt}</p>
			{/if}
			<p class="label muted small">{t('Answer')}</p>
			{#if card.has_image}<img
					src="/api/v1/flashcards/{card.id}/image"
					alt={t('Illustration for: {front}', { front: card.front })}
				/>{/if}
			<p class="answer">{card.back}</p>
			{#if card.elaboration || card.example || card.mnemonic}
				<dl class="stick">
					{#if card.elaboration}<dt>{t('Why')}</dt>
						<dd>{card.elaboration}</dd>{/if}
					{#if card.example}<dt>{t('Example')}</dt>
						<dd>{card.example}</dd>{/if}
					{#if card.mnemonic}<dt>{t('Picture it')}</dt>
						<dd>{card.mnemonic}</dd>{/if}
				</dl>
			{/if}
			{#if card.leech}
				<p class="leech" role="note">
					{t('This card keeps slipping ({lapses} lapses). Reviewing it again will not fix it:', {
						lapses: card.lapses
					})}
					<a href="/cards?deck={encodeURIComponent(card.deck)}"
						>{t('rewrite it, add an example or a vivid image, or split it')}</a
					>.
				</p>
			{/if}
		{/if}
	</section>
	{#if !shown}
		<Button onclick={() => (shown = true)}>{t('Show answer')} <kbd>Space</kbd></Button>
	{:else if practice}
		<div class="ratings" role="group" aria-label={t('Did you recall it?')}>
			<Button variant="secondary" onclick={() => mark(false)}>{t('Missed')} <kbd>1</kbd></Button>
			<Button onclick={() => mark(true)}>{t('Got it')} <kbd>2</kbd></Button>
		</div>
	{:else}
		<div class="ratings" role="group" aria-label={t('How well did you remember?')}>
			{#each ratings as r (r.value)}
				<Button variant={r.value === 3 ? 'primary' : 'secondary'} onclick={() => rate(r.value)}>
					{t(r.label)}
					{#if card.next}<span class="when"
							>{interval(new Date(), new Date(card.next[r.next]))}</span
						>{/if}
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
	.row {
		display: flex;
		gap: var(--space-2);
		justify-content: center;
		flex-wrap: wrap;
	}
	.attempt {
		display: block;
		margin-top: var(--space-4);
		text-align: left;
	}
	#attempt {
		width: 100%;
		font: inherit;
	}
	.mine {
		white-space: pre-wrap;
		color: var(--muted);
	}
	.stick {
		text-align: left;
		margin: var(--space-4) auto 0;
		max-width: 40rem;
	}
	.stick dt {
		font-weight: 650;
		margin-top: var(--space-2);
	}
	.stick dd {
		margin: 0;
		white-space: pre-wrap;
	}
	.leech {
		margin-top: var(--space-4);
		font-size: var(--text-sm);
	}
</style>
