<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { deleted } from '$lib/undo';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, message, type Schemas } from '$lib/api';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Palace = Schemas['Palace'];
	type Placed = NonNullable<Palace['cards']>[number];
	const id = $derived(page.params.id ?? '');
	let palace = $state<Palace | null>(null);
	let error = $state('');
	let loci = $state('');
	let confirmDelete = $state(false);
	// The walk: one place at a time; recall what is there, then look.
	let walking = $state(false);
	let step = $state(0);
	let shown = $state(false);

	async function load() {
		error = '';
		try {
			palace = await api<Palace>(`/v1/palaces/${id}`);
			loci = palace.loci.join('\n');
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void id;
		load();
	});

	const at = (i: number): Placed[] => (palace?.cards ?? []).filter((c) => c.locus === i);
	const stops = $derived(palace ? palace.loci.map((_, i) => i).filter((i) => at(i).length) : []);

	async function saveLoci(e: SubmitEvent) {
		e.preventDefault();
		try {
			palace = await api<Palace>(`/v1/palaces/${id}`, {
				method: 'PATCH',
				body: {
					loci: loci
						.split('\n')
						.map((l) => l.trim())
						.filter(Boolean)
				}
			});
			toast(t('Saved.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove() {
		try {
			await api(`/v1/palaces/${id}`, { method: 'DELETE' });
			const gone = id;
			deleted(t('Palace deleted.'), gone, () => goto(`/learn/palaces/${gone}`));
			await goto('/learn/palaces');
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function go(delta: number) {
		step = Math.min(Math.max(step + delta, 0), stops.length - 1);
		shown = false;
	}
</script>

<svelte:head><title>{palace?.title ?? t('Memory palace')} · Axon</title></svelte:head>

<p class="crumbs"><a href="/learn/palaces">{t('← Memory palaces')}</a></p>
<h1>{palace?.title ?? t('Memory palace')}</h1>
<LearnTabs />

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !palace}
	<Status kind="loading" message={t('Loading the palace…')} />
{:else if walking && stops.length}
	{@const i = stops[step]}
	<section class="card glass walk" aria-live="polite">
		<p class="muted small">
			{t('Place {value} of {length}', { value: step + 1, length: stops.length })}
		</p>
		<h2>{palace.loci[i]}</h2>
		{#if !shown}
			<p>{t('What did you leave here? Picture it, say it, then look.')}</p>
			<Button onclick={() => (shown = true)}>{t('Show')}</Button>
		{:else}
			<ul class="placed">
				{#each at(i) as c (c.id)}
					<li>
						<strong>{c.front}</strong>
						<p>{c.back}</p>
						{#if c.has_image}<img
								src="/api/v1/flashcards/{c.id}/image"
								alt={t('Illustration for: {front}', { front: c.front })}
							/>{/if}
						{#if c.mnemonic}<p class="muted">
								{t('Picture it: {mnemonic}', { mnemonic: c.mnemonic })}
							</p>{/if}
					</li>
				{/each}
			</ul>
		{/if}
		<div class="row">
			<Button variant="secondary" disabled={step === 0} onclick={() => go(-1)}
				>{t('Previous place')}</Button
			>
			{#if step < stops.length - 1}
				<Button onclick={() => go(1)}>{t('Next place')}</Button>
			{:else}
				<Button onclick={() => (walking = false)}>{t('End the walk')}</Button>
			{/if}
		</div>
	</section>
{:else}
	{#if palace.description}<p>{palace.description}</p>{/if}
	<div class="row">
		<Button
			disabled={!stops.length}
			onclick={() => {
				walking = true;
				step = 0;
				shown = false;
			}}>{t('Walk the palace')}</Button
		>
		<Button variant="danger" onclick={() => (confirmDelete = true)}>{t('Delete')}</Button>
	</div>
	{#if !stops.length}
		<p class="muted">
			{t('No card is placed here yet. Open a card on the')} <a href="/cards">{t('Flashcards')}</a>
			{t('page, choose this palace and a place, and give it a vivid memory image.')}
		</p>
	{/if}

	<ol class="loci">
		{#each palace.loci as l, i (i)}
			<li>
				<strong>{l}</strong>
				{#each at(i) as c (c.id)}
					<span class="muted small"> · {c.front}</span>
				{/each}
			</li>
		{/each}
	</ol>

	<section class="card glass" aria-labelledby="edit-title">
		<h2 id="edit-title">{t('Places')}</h2>
		<form onsubmit={saveLoci}>
			<label for="loci">{t('One per line, in walking order')}</label>
			<textarea id="loci" rows="8" bind:value={loci}></textarea>
			<p class="muted small">{t('Removing a place takes the cards there out of the palace.')}</p>
			<Button type="submit" variant="secondary">{t('Save places')}</Button>
		</form>
	</section>
{/if}

<Dialog bind:open={confirmDelete} title={t('Delete this palace?')}>
	<p>{t('Its cards are kept; they just leave the palace.')}</p>
	<div class="row">
		<Button variant="danger" onclick={remove}>{t('Delete')}</Button>
		<Button variant="secondary" onclick={() => (confirmDelete = false)}>{t('Cancel')}</Button>
	</div>
</Dialog>

<style>
	.crumbs a {
		color: var(--muted);
		text-decoration: none;
	}
	.small {
		font-size: var(--text-sm);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin: var(--space-3) 0;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.walk {
		text-align: center;
	}
	.placed {
		list-style: none;
		padding: 0;
	}
	.placed img {
		max-width: 100%;
		max-height: 14rem;
	}
	.loci li {
		margin-bottom: var(--space-1);
	}
	textarea {
		width: 100%;
		font: inherit;
	}
</style>
