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

	const features = [
		{
			id: 'notes',
			title: 'Notes that connect.',
			text: 'Plain Markdown with [[wikilinks]], backlinks, and PARA folders. Your vault stays yours and syncs both ways.',
			tiles: ['Zettel/permanent/proofs.md', '[[induction]] · 3 backlinks', '2-Areas/maths']
		},
		{
			id: 'tasks',
			title: 'Work, made calm.',
			text: 'Capture anything with one key. Today, week, and GTD views, repeating tasks, and a daily check-in.',
			tiles: [
				'☐ Call the plumber · @phone',
				'☐ Renew passport · due Friday',
				'☑ Stretch for ten minutes'
			]
		},
		{
			id: 'goals',
			title: 'Goals you can explain.',
			text: 'Milestones, linked tasks, and habits. Every progress figure shows exactly how it was worked out.',
			tiles: ['Learn to swim 1 km', '■■■■■□□□□□ 50%', '2 of 4 milestones and tasks done']
		},
		{
			id: 'habits',
			title: 'Habits without guilt.',
			text: 'A cue, a stack, and a two-minute version for hard days. Miss one, and Axon helps you start again.',
			tiles: ['After I pour my coffee', 'Read one page', 'Never miss twice']
		},
		{
			id: 'agents',
			title: 'AI that asks first.',
			text: 'Agents on the provider you choose, or a model on your own hardware. Every change waits for your approval.',
			tiles: ['Proposal · status → next', 'Approve   Reject', 'Budget $2.00 · spent $0.12']
		}
	];
</script>

<svelte:head><title>Axon · Your self-hosted life, connected</title></svelte:head>

<section class="hero" aria-labelledby="hero-title">
	<Scenery />
	<div class="shade" aria-hidden="true"></div>
	<div class="hero-copy">
		<p class="eyebrow">{t('Self-hosted life OS')}</p>
		<h1 id="hero-title">{t('Your life,')}<br />{t('connected.')}</h1>
		<p class="lead">
			{t(
				'Notes, tasks, goals, habits, and reflection in one calm graph that runs on your own server.'
			)}
		</p>
		{#if status && !status.signed_in}
			<div class="actions">
				<Button href="/signin">{t('Sign in')}</Button>
				{#if status.registration_open}<Button href="/register" variant="secondary"
						>{t('Create your account')}</Button
					>{/if}
				{#if status.demo_available}<Button variant="secondary" busy={entering} onclick={tryDemo}
						>{t('Try the demo')}</Button
					>{/if}
			</div>
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
		<h2 id="teaser-title">{t('Axon in under a minute')}</h2>
		<p>{t('A tour of the demo, with music Axon composed itself.')}</p>
		<!-- Played on request: nothing downloads until then, and nothing moves on its own.
		     Its captions are drawn into the picture; the tracks hold the same text, in two languages. -->
		<video controls preload="none" playsinline poster="/teaser/axon.jpg" width="1280" height="720">
			<source src="/teaser/axon.webm" type="video/webm" />
			<source src="/teaser/axon.mp4" type="video/mp4" />
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
			<div class="stage {f.id}" aria-hidden="true">
				<div class="visual glass">
					{#each f.tiles as t (t)}<div class="tile">{t}</div>{/each}
				</div>
			</div>
		</section>
	{/each}
</div>

<p class="status" role="status">
	<span class="dot {data.api.state}" aria-hidden="true"></span>
	{#if data.api.state === 'up'}
		Server running · version {data.api.version}
	{:else}
		Server not available. {data.api.reason}
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
	.feature.flip .copy {
		order: 2;
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
