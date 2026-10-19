<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, message, type AuthStatus } from '$lib/api';
	import { toast } from '$lib/ui/toast.svelte';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Scenery from '$lib/ui/Scenery.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let status = $state<AuthStatus | null>(null);

	onMount(async () => {
		status = await api<AuthStatus>('/v1/auth/status').catch(() => null);
		if (status?.signed_in) await goto('/today', { replaceState: true });
	});

	let entering = $state(false);
	async function tryDemo() {
		entering = true;
		try {
			await api('/v1/auth/demo', { method: 'POST' });
			await goto('/today');
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			entering = false;
		}
	}

	// One section per part of life, each with a page of the demo; `tone` picks its colour field.
	const features = $derived([
		{
			id: 'today',
			tone: 'goals',
			title: t('Your day, at a glance.'),
			text: t(
				'Tasks due, habits, goals, and a daily check-in on one page, with the next step of your roadmap.'
			),
			shot: t('Today: tasks, habits, and goals for the day')
		},
		{
			id: 'roadmap',
			tone: 'agents',
			title: t('Your own roadmap.'),
			text: t(
				'Set up a personal roadmap for your self-learning journey: domains, certifications, projects, and a radar of your hard and soft skills.'
			),
			shot: t('The roadmap: hard and soft skills, domain by domain')
		},
		{
			id: 'learn',
			tone: 'goals',
			title: t('Learn so it sticks.'),
			text: t(
				'Flashcards with spaced repetition, free recall, memory palaces, and weekly tests, in sync with Anki.'
			),
			shot: t('Learn: flashcards, recall, and memory palaces')
		},
		{
			id: 'tasks',
			tone: 'tasks',
			title: t('Work, made calm.'),
			text: t(
				'Capture anything with one key. Board, GTD, and calendar views, reminders, and focus sessions.'
			),
			shot: t('Tasks on a board')
		},
		{
			id: 'health',
			tone: 'habits',
			title: t('Body and mind.'),
			text: t(
				'Mood and energy, water, coffee and tea, meals from a photo, training, and habits backed by evidence.'
			),
			shot: t('Health: mood, water, and caffeine')
		},
		{
			id: 'decisions',
			tone: 'tasks',
			title: t('Decide better, spend wisely.'),
			text: t(
				'Budgets, accounts, and recurring transactions in clear charts, and decisions you can check against what happened.'
			),
			shot: t('Decisions: how well your forecasts came true')
		},
		{
			id: 'watch',
			tone: 'notes',
			title: t('The world, digested.'),
			text: t(
				'A tech watch from feeds, YouTube, and podcasts, summed up, drawn as a graph, and read aloud. And your collection of books, films, and games.'
			),
			shot: t('The tech watch')
		},
		{
			id: 'notes',
			tone: 'notes',
			title: t('Notes that connect.'),
			text: t(
				'Plain Markdown with [[wikilinks]], backlinks, and PARA folders. Your vault stays yours and syncs both ways.'
			),
			shot: t('Notes as a knowledge graph')
		},
		{
			id: 'agents',
			tone: 'agents',
			title: t('AI that asks first.'),
			text: t(
				'Jarvis and agents on the provider you choose, or a model on your own hardware. Every change waits for your approval.'
			),
			tiles: [t('Proposal · status → next'), t('Approve   Reject'), t('Budget $2.00 · spent $0.12')]
		}
	]);
</script>

<svelte:head><title>{t('Axon · Your self-hosted life, connected')}</title></svelte:head>

<section class="hero" aria-labelledby="hero-title">
	<Scenery />
	<div class="shade" aria-hidden="true"></div>
	<div class="hero-copy">
		<p class="eyebrow">{t('Self-hosted life OS')}</p>
		<h1 id="hero-title">{t('Your life,')}<br />{t('connected.')}</h1>
		<p class="lead">
			{t(
				'Your plan, learning, health, money, and notes in one calm system that runs on your own server.'
			)}
		</p>
		{#if status && !status.signed_in}
			<div class="actions">
				<Button href="/signin">{t('Sign in')}</Button>
				<!-- Registration closed: accounts here are the owner's to give; anyone
				     else can run their own Axon, from its source. -->
				<Button
					href={status.registration_open ? '/register' : 'https://github.com/thoutmose/axon'}
					variant="secondary">{t('Create your account')}</Button
				>
				{#if status.demo_available}<Button variant="secondary" busy={entering} onclick={tryDemo}
						>{t('Try the demo')}</Button
					>{/if}
			</div>
			{#if !status.registration_open}<p class="demo-note">
					{t(
						'Accounts on this Axon are by invitation. Axon is free software: run your own from GitHub.'
					)}
				</p>{/if}
			{#if status.demo_available}<p class="demo-note">
					{t(
						'The demo is a shared account with sample data, reset every night. No passkey needed.'
					)}
				</p>{/if}
		{/if}
	</div>
</section>

{#if status?.demo_available}
	<section class="teaser" aria-labelledby="teaser-title">
		<h2 id="teaser-title">{t('Axon in a minute')}</h2>
		<p>{t('A tour of the demo, with music Axon composed itself.')}</p>
		<!-- Played on request: nothing downloads until then, and nothing moves on its own.
		     Its captions are drawn into the picture; the tracks hold the same text, in two languages. -->
		<video controls preload="none" playsinline poster="/teaser/axon.jpg" width="1280" height="720">
			<source src="/teaser/axon-trailer-4k.mp4" type="video/mp4" />
			<track kind="captions" src="/teaser/axon.en.vtt" srclang="en" label="English" />
			<track kind="captions" src="/teaser/axon.fr.vtt" srclang="fr" label="Français" />
		</video>
	</section>
{/if}

<div class="features">
	{#each features as f, i (f.id)}
		<section class="feature" class:flip={i % 2 === 1} aria-labelledby="f-{f.id}">
			<div class="copy">
				<h2 id="f-{f.id}">{t(f.title)}</h2>
				<p>{t(f.text)}</p>
			</div>
			{#if f.shot}
				<div class="stage {f.tone}">
					<img
						class="shot"
						src="/teaser/pages/{f.id}.jpg"
						alt={f.shot}
						width="1440"
						height="900"
						loading="lazy"
						decoding="async"
					/>
				</div>
			{:else}
				<div class="stage {f.tone}" aria-hidden="true">
					<div class="visual glass">
						{#each f.tiles ?? [] as tileText (tileText)}<div class="tile">{tileText}</div>{/each}
					</div>
				</div>
			{/if}
		</section>
	{/each}
</div>

<p class="status" role="status">
	<span class="dot {data.api.state}" aria-hidden="true"></span>
	{#if data.api.state === 'up'}
		{t('Server running · version {version}', { version: data.api.version })}
	{:else}
		{t('Server not available. {reason}', { reason: data.api.reason })}
	{/if}
</p>

<style>
	/* Buttons on scenery: frosted white instead of a grey surface. */
	.hero :global(.btn.secondary),
	.hero :global(.btn.ghost) {
		background: rgb(255 255 255 / 0.18);
		color: #fff;
		border-color: rgb(255 255 255 / 0.35);
	}
	.hero :global(.btn.secondary:hover) {
		background: rgb(255 255 255 / 0.28);
	}
	.hero {
		position: relative;
		min-height: min(88vh, 52rem);
		display: grid;
		align-items: end;
		overflow: hidden;
		color: #fff;
	}
	.shade {
		position: absolute;
		inset: 0;
		background: linear-gradient(180deg, rgb(0 0 0 / 0.05) 30%, rgb(0 0 0 / 0.55) 100%);
	}
	.hero-copy {
		position: relative;
		max-width: var(--wide-width);
		width: 100%;
		margin: 0 auto;
		padding: var(--space-16) var(--gutter) clamp(3rem, 10vh, 6rem);
	}
	.eyebrow {
		text-transform: uppercase;
		letter-spacing: 0.14em;
		font-size: var(--text-sm);
		font-weight: 600;
		opacity: 0.9;
	}
	h1 {
		font-size: var(--text-display);
		line-height: 0.98;
		margin-bottom: var(--space-6);
		text-shadow: 0 2px 30px rgb(0 0 0 / 0.25);
	}
	.lead {
		font-size: var(--text-xl);
		max-width: 34rem;
		color: rgb(255 255 255 / 0.92);
		text-shadow: 0 1px 12px rgb(0 0 0 / 0.35);
	}
	.actions {
		display: flex;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	.demo-note {
		font-size: var(--text-sm);
		color: rgb(255 255 255 / 0.92);
		text-shadow: 0 1px 12px rgb(0 0 0 / 0.5);
		margin-top: var(--space-3);
	}
	.teaser {
		max-width: var(--wide-width);
		margin: 0 auto;
		padding: var(--space-16) var(--gutter) 0;
		text-align: center;
	}
	.teaser h2 {
		font-size: var(--text-3xl);
		letter-spacing: -0.03em;
	}
	.teaser p {
		color: var(--muted);
		font-size: var(--text-lg);
	}
	.teaser video {
		width: 100%;
		height: auto;
		margin-top: var(--space-6);
		border-radius: var(--radius-xl);
		border: 1px solid var(--border);
		box-shadow: var(--shadow);
		background: #000;
	}
	.features {
		max-width: var(--wide-width);
		margin: 0 auto;
		padding: var(--space-16) var(--gutter) 0;
		display: grid;
		gap: clamp(3rem, 10vw, 7rem);
	}
	.feature {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 20rem), 1fr));
		gap: var(--space-8);
		align-items: center;
	}
	/* Alternate sides only side by side: in one column, each text stays above its picture. */
	@media (min-width: 48rem) {
		.feature.flip .copy {
			order: 2;
		}
	}
	.feature h2 {
		font-size: var(--text-3xl);
		letter-spacing: -0.03em;
	}
	.feature p {
		font-size: var(--text-lg);
		color: var(--muted);
		max-width: 30rem;
	}
	/* A colour field behind each card, so its glass has something to blur. */
	.stage {
		position: relative;
		padding: clamp(1.5rem, 5vw, 3rem);
		border-radius: calc(var(--radius-xl) + 0.75rem);
		overflow: hidden;
		isolation: isolate;
	}
	.stage::before {
		content: '';
		position: absolute;
		inset: 0;
		z-index: -1;
		background:
			radial-gradient(circle at 20% 25%, var(--c1), transparent 55%),
			radial-gradient(circle at 85% 80%, var(--c2), transparent 55%),
			linear-gradient(135deg, var(--c3), var(--c4));
	}
	.notes {
		--c1: #7cc4ff;
		--c2: #b28cff;
		--c3: #dbe9ff;
		--c4: #efe3ff;
	}
	.tasks {
		--c1: #7ff0c4;
		--c2: #58b7ff;
		--c3: #dcfaf0;
		--c4: #dceeff;
	}
	.goals {
		--c1: #ffb36b;
		--c2: #ff6f91;
		--c3: #fff0dc;
		--c4: #ffe0ea;
	}
	.habits {
		--c1: #b4f07a;
		--c2: #43c6ac;
		--c3: #f0fbdc;
		--c4: #dcf7f0;
	}
	.agents {
		--c1: #a98bff;
		--c2: #ff8ad8;
		--c3: #ece4ff;
		--c4: #ffe4f5;
	}
	.visual {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-xl);
		box-shadow: var(--shadow);
		padding: var(--space-6);
		display: grid;
		gap: var(--space-3);
	}
	.shot {
		display: block;
		width: 100%;
		height: auto;
		border-radius: var(--radius-xl);
		border: 1px solid var(--border);
		box-shadow: var(--shadow);
	}
	.tile {
		background: var(--surface-solid);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-3) var(--space-4);
		font-size: var(--text-sm);
	}
	.status {
		max-width: var(--wide-width);
		margin: var(--space-16) auto 0;
		padding: 0 var(--gutter);
		display: flex;
		align-items: center;
		gap: var(--space-2);
		color: var(--muted);
		font-size: var(--text-sm);
	}
	.dot {
		width: 0.5rem;
		height: 0.5rem;
		border-radius: 50%;
		background: var(--bad);
	}
	.dot.up {
		background: var(--ok);
	}
</style>
