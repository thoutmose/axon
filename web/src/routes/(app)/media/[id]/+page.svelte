<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, message, type Schemas } from '$lib/api';
	import Connections from '$lib/Connections.svelte';
	import { t } from '$lib/i18n.svelte';
	import { left, stars } from '$lib/media';
	import MediaCover from '$lib/MediaCover.svelte';
	import { refreshNotes } from '$lib/noteIndex.svelte';
	import { mediaNote } from '$lib/noteTemplates';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// One film, series, or book (EP-135): the cover large, what it is about,
	// where you are, your rating and review, and what it connects to.
	type Item = Schemas['MediaItem'];
	let item = $state<Item | null>(null);
	let error = $state('');
	let progress = $state(0);
	let review = $state('');
	let busy = $state(false);

	const kindName = $derived({ film: t('Film'), series: t('Series'), book: t('Book') });
	const unit = $derived({ film: t('minutes'), series: t('episodes'), book: t('pages') });
	const steps: Record<Item['kind'], number[]> = { book: [10, 20, 50], series: [1], film: [30] };
	const statusName = $derived({
		want: t('Want to'),
		doing: t('In progress'),
		done: t('Finished'),
		dropped: t('Dropped')
	});

	async function load() {
		try {
			item = await api<Item>(`/v1/media/${page.params.id}`);
			progress = item.progress;
			review = item.review;
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void page.params.id;
		load();
	});

	async function patch(body: Record<string, unknown>, said = t('Saved.')) {
		busy = true;
		try {
			item = await api<Item>(`/v1/media/${item!.id}`, { method: 'PATCH', body });
			progress = item.progress;
			toast(said);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	// Its note (EP-136): the film, series, or book template filled from the
	// item, linked to it, opened to write.
	async function writeNote() {
		busy = true;
		try {
			const note = await api<{ id: string }>('/v1/notes', {
				method: 'POST',
				body: mediaNote(item!, new Date())
			});
			await api('/v1/links', {
				method: 'POST',
				body: { source_id: note.id, relation: 'relates_to', target_id: item!.id }
			});
			await refreshNotes();
			await goto(`/notes/${note.id}?edit=1`);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	const advance = (n: number) =>
		patch({
			progress: item!.total ? Math.min(item!.total, item!.progress + n) : item!.progress + n,
			...(item!.status === 'want' ? { status: 'doing' } : {})
		});
</script>

<svelte:head><title>{item?.title ?? t('Media')} · Axon</title></svelte:head>

<p><a href="/media">← {t('Films, series, and books')}</a></p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !item}
	<Status kind="loading" message={t('Loading…')} />
{:else}
	{@const rest = left(item)}
	<article class="hero">
		<div class="poster"><MediaCover {item} kind={kindName[item.kind]} /></div>
		<div class="about">
			<p class="muted small">
				{kindName[item.kind]}{item.year ? ` · ${item.year}` : ''} · {statusName[item.status]}
			</p>
			<h1>{item.title}</h1>
			{#if item.creators}<p class="by">{item.creators}</p>{/if}
			{#if item.rating}
				<p class="stars" title="{item.rating}/10">
					<span aria-hidden="true">{stars(item.rating)}</span>
					<span class="visually-hidden">{t('Rated')} {item.rating}/10</span>
				</p>
			{/if}
			{#if item.genres.length}
				<p class="genres">
					{#each item.genres as g (g)}<span class="chip">{g}</span>{/each}
				</p>
			{/if}
			{#if item.overview}<p class="overview">{item.overview}</p>{/if}
			<div class="row">
				<Button variant="secondary" {busy} onclick={writeNote}>{t('Write a note')}</Button>
			</div>

			<section class="card" aria-labelledby="where">
				<h2 id="where">{t('Where you are')}</h2>
				{#if item.total}
					<progress
						max={item.total}
						value={item.progress}
						aria-label="{item.progress} {t('of')} {item.total} {unit[item.kind]}"
					></progress>
					<p>
						<strong>{item.progress}</strong> / {item.total}
						{unit[item.kind]} · {t('{n} left', { n: rest ?? 0 })}
					</p>
				{/if}
				<div class="row">
					{#each steps[item.kind] as n (n)}
						<Button variant="secondary" {busy} onclick={() => advance(n)}
							>+{n} {unit[item.kind]}</Button
						>
					{/each}
					<form
						class="row"
						onsubmit={(e) => {
							e.preventDefault();
							patch({ progress });
						}}
					>
						<label
							>{t('Progress')}
							<input type="number" min="0" bind:value={progress} /></label
						>
						<Button type="submit" variant="ghost" {busy}>{t('Save')}</Button>
					</form>
				</div>
				<div class="row">
					{#if item.status !== 'doing' && item.status !== 'done'}
						<Button onclick={() => patch({ status: 'doing' })}>{t('Start')}</Button>
					{/if}
					{#if item.status === 'doing'}
						<Button onclick={() => patch({ status: 'done' })}>{t('Finish')}</Button>
					{/if}
					<label
						>{t('Rating')}
						<select
							value={item.rating ?? 0}
							onchange={(e) => patch({ rating: Number(e.currentTarget.value) })}
						>
							<option value={0}>{t('Not rated')}</option>
							{#each [10, 9, 8, 7, 6, 5, 4, 3, 2, 1] as n (n)}<option value={n}>{n}/10</option
								>{/each}
						</select></label
					>
				</div>
			</section>

			<section class="card" aria-labelledby="review-title">
				<h2 id="review-title">{t('Review')}</h2>
				<textarea rows="5" maxlength="10000" bind:value={review} aria-labelledby="review-title"
				></textarea>
				<div class="row">
					<Button variant="secondary" {busy} onclick={() => patch({ review })}>{t('Save')}</Button>
				</div>
			</section>
		</div>
	</article>

	<Connections id={item.id} type="media" />
{/if}

<style>
	.hero {
		display: grid;
		grid-template-columns: minmax(0, 16rem) minmax(0, 1fr);
		gap: var(--space-6);
		align-items: start;
		margin-bottom: var(--space-4);
	}
	@media (max-width: 40rem) {
		.hero {
			grid-template-columns: 1fr;
		}
		.poster {
			max-width: 12rem;
		}
	}
	.poster {
		border-radius: var(--radius);
		box-shadow: 0 8px 24px rgb(0 0 0 / 0.3);
	}
	h1 {
		margin: 0;
	}
	.by {
		font-size: var(--text-lg);
		margin: var(--space-1) 0;
	}
	.small {
		font-size: var(--text-sm);
	}
	.stars {
		color: var(--warn);
		font-size: 1.25rem;
		letter-spacing: 0.1em;
		margin: var(--space-1) 0;
	}
	.genres {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
	}
	.chip {
		font-size: var(--text-sm);
		border-radius: 999px;
		padding: 0 var(--space-2);
		background: var(--accent-soft);
	}
	.overview {
		max-width: 65ch;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-top: var(--space-4);
		display: grid;
		gap: var(--space-3);
	}
	.card h2 {
		margin: 0;
	}
	progress {
		width: 100%;
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: end;
	}
	label {
		display: grid;
		gap: var(--space-1);
	}
	label input {
		max-width: 7rem;
	}
	textarea {
		font: inherit;
		width: 100%;
	}
</style>
