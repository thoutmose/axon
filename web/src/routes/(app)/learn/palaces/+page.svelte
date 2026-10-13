<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Palace = Schemas['Palace'];
	let palaces = $state<Palace[] | null>(null);
	let error = $state('');
	let title = $state('');
	let loci = $state('');

	async function load() {
		error = '';
		try {
			palaces = (await api<{ items: Palace[] }>('/v1/palaces')).items;
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function create(e: SubmitEvent) {
		e.preventDefault();
		try {
			const p = await api<Palace>('/v1/palaces', {
				method: 'POST',
				body: {
					title,
					loci: loci
						.split('\n')
						.map((l) => l.trim())
						.filter(Boolean)
				}
			});
			toast(t('Palace built.'));
			await goto(`/learn/palaces/${p.id}`);
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>{t('Memory palaces · Axon')}</title></svelte:head>

<h1>{t('Memory palaces')}</h1>
<LearnTabs />
<p class="muted hint">
	{t(
		'The method of loci, as memory champions use it: choose a place you know well, list the spots you pass in order, then picture each thing to remember as a vivid, strange image at one spot. To recall, walk through the place in your mind. Put a flashcard in a palace from its Edit dialog.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !palaces}
	<Status kind="loading" message={t('Loading palaces…')} />
{:else}
	{#if palaces.length}
		<ul class="palaces">
			{#each palaces as p (p.id)}
				<li class="card glass">
					<a class="name" href="/learn/palaces/{p.id}">{p.title}</a>
					<p class="muted small">
						{t(p.loci.length === 1 ? '{n} place' : '{n} places', { n: p.loci.length }) +
							(p.loci.length
								? `: ${p.loci.slice(0, 3).join(' → ')}${p.loci.length > 3 ? ' → …' : ''}`
								: '')}
					</p>
				</li>
			{/each}
		</ul>
	{:else}
		<Status
			kind="empty"
			message={t('No palace yet. Start with your home: the route from the door.')}
		/>
	{/if}

	<section class="card glass" aria-labelledby="new-title">
		<h2 id="new-title">{t('Build a palace')}</h2>
		<form onsubmit={create}>
			<TextField id="palace-title" label={t('Place')} bind:value={title} required maxlength={200} />
			<div class="field">
				<label for="palace-loci">{t('Spots, in walking order, one per line')}</label>
				<textarea
					id="palace-loci"
					rows="6"
					bind:value={loci}
					placeholder={t('Front door\nCoat hook\nKitchen sink\nSofa\nBookshelf')}
				></textarea>
			</div>
			<Button type="submit" variant="secondary">{t('Build')}</Button>
		</form>
	</section>
{/if}

<style>
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.palaces {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 16rem), 1fr));
		gap: var(--space-3);
		margin: var(--space-4) 0;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	.name {
		font-weight: 650;
		color: var(--fg);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin: var(--space-3) 0;
	}
	textarea {
		font: inherit;
	}
</style>
