<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { page } from '$app/state';
	import { api, message } from '$lib/api';
	import { formatDate, appLocale } from '$lib/dates';
	import type { Note } from '$lib/notes';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Result = {
		score: number;
		ideas: { text: string; recalled: boolean }[];
		due_on: string;
		interval_days: number;
	};
	type Attempt = { score: number; recalled_at: string; due_on: string };

	const id = $derived(page.params.id ?? '');
	let note = $state<Note | null>(null);
	let history = $state<Attempt[]>([]);
	let error = $state('');
	let text = $state('');
	let result = $state<Result | null>(null);
	let sending = $state(false);
	const study = $derived(note?.kind === 'literature' || note?.kind === 'permanent');

	async function load() {
		error = '';
		result = null;
		try {
			// Only the title: the note itself stays hidden until the recall is done.
			note = await api<Note>(`/v1/notes/${id}`);
			history = (await api<{ items: Attempt[] }>(`/v1/notes/${id}/recall`)).items;
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void id;
		load();
	});

	async function send(e: SubmitEvent) {
		e.preventDefault();
		sending = true;
		try {
			result = await api<Result>(`/v1/notes/${id}/recall`, { method: 'POST', body: { text } });
			history = (await api<{ items: Attempt[] }>(`/v1/notes/${id}/recall`)).items;
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			sending = false;
		}
	}

	const pct = (x: number) => `${Math.round(x * 100)}%`;
	const words = $derived((text.match(/\S+/g) ?? []).length);
</script>

<svelte:head
	><title
		>{note ? t('Recall {title} · Axon', { title: note.title }) : t('Recall a note · Axon')}</title
	></svelte:head
>

<p class="crumbs"><a href="/learn">{t('← Learn')}</a></p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !note}
	<Status kind="loading" message={t('Loading…')} />
{:else}
	<h1>{t('Recall: {title}', { title: note.title })}</h1>
	{#if !study}
		<Status
			kind="empty"
			message={t(
				'Free recall is for literature and permanent notes. Rewrite this one in your own words first, then set its type.'
			)}
		/>
	{:else if !result}
		<p class="muted">
			{t(
				'Write everything you remember about this note, in your own words, without looking. Getting stuck is the point: the effort of recalling is what makes memories last.'
			)}
		</p>
		<form onsubmit={send}>
			<label for="recall" class="visually-hidden">{t('What you remember')}</label>
			<!-- svelte-ignore a11y_autofocus -->
			<textarea id="recall" rows="12" maxlength="20000" required autofocus bind:value={text}
			></textarea>
			<div class="row">
				<Button type="submit" disabled={sending || !text.trim()}
					>{t('Compare with the note')}</Button
				>
				<span class="muted small">{t('{words} words', { words })}</span>
			</div>
		</form>
	{:else}
		<section class="card glass" aria-live="polite">
			<h2>{t('{value} of the key ideas', { value: pct(result.score) })}</h2>
			<ul class="ideas">
				{#each result.ideas as i (i.text)}
					<li class:missed={!i.recalled}>
						<span aria-hidden="true">{i.recalled ? '✓' : '○'}</span>
						<span class="visually-hidden">{i.recalled ? t('Recalled:') : t('Missed:')}</span>
						{i.text}
					</li>
				{/each}
			</ul>
			<p>
				{t(
					result.score >= 0.8
						? 'Well recalled.'
						: result.score >= 0.5
							? 'Half there: reread what you missed, then explain it to yourself.'
							: 'Reread the note now, then try again tomorrow.'
				) +
					' ' +
					t(
						result.interval_days === 1
							? 'Next recall in {n} day ({date}).'
							: 'Next recall in {n} days ({date}).',
						{
							n: result.interval_days,
							date: new Date(result.due_on).toLocaleDateString(appLocale())
						}
					)}
			</p>
			<div class="row">
				<Button href="/notes/{id}">{t('Read the note')}</Button>
				<Button variant="secondary" href="/learn">{t('Back to today')}</Button>
			</div>
		</section>
	{/if}

	{#if history.length}
		<section aria-labelledby="history-title">
			<h2 id="history-title" class="small-title">{t('Earlier attempts')}</h2>
			<ul class="history">
				{#each history as h (h.recalled_at)}
					<li>
						{new Date(h.recalled_at).toLocaleDateString(appLocale())}: {pct(h.score)}
						<span class="muted">{t('· next {value}', { value: formatDate(h.due_on) })}</span>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
{/if}

<style>
	.crumbs a {
		color: var(--muted);
		text-decoration: none;
	}
	textarea {
		width: 100%;
		font: inherit;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		flex-wrap: wrap;
		margin: var(--space-3) 0;
	}
	.small {
		font-size: var(--text-sm);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.ideas {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-1);
	}
	.missed {
		color: var(--muted);
	}
	.small-title {
		font-size: var(--text-base);
	}
	.history {
		font-size: var(--text-sm);
	}
</style>
