<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { deleted } from '$lib/undo';
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import type { Flashcard } from '$lib/flashcards';
	import AnkiSync from '$lib/AnkiSync.svelte';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Deck = { name: string; total: number; due: number; new: number };
	type Palace = { id: string; title: string; loci: string[] };
	const deck = $derived(page.url.searchParams.get('deck') ?? '');
	let decks = $state<Deck[] | null>(null);
	let cards = $state<Flashcard[]>([]);
	let error = $state('');
	const empty = { elaboration: '', example: '', mnemonic: '' };
	let form = $state({ front: '', back: '', deck: '', ...empty });
	let palaces = $state<Palace[]>([]);
	let picture = $state<File | null>(null);
	let editing = $state<Flashcard | null>(null);
	let editOpen = $state(false);
	let edit = $state({
		front: '',
		back: '',
		deck: '',
		...empty,
		palace_id: '',
		locus: 0
	});
	const editPalace = $derived(palaces.find((p) => p.id === edit.palace_id));
	let version = $state(Date.now());

	async function load() {
		error = '';
		try {
			decks = (await api<{ items: Deck[] }>('/v1/decks')).items;
			palaces = (await api<{ items: Palace[] }>('/v1/palaces')).items;
			cards = deck
				? (await api<{ items: Flashcard[] }>(`/v1/flashcards?deck=${encodeURIComponent(deck)}`))
						.items
				: [];
			form.deck = form.deck || deck || decks[0]?.name || '';
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void deck;
		load();
	});

	async function upload(id: string, file: File) {
		const res = await fetch(`/api/v1/flashcards/${id}/image`, {
			method: 'PUT',
			headers: { 'Content-Type': file.type },
			body: file
		});
		if (!res.ok) throw new Error((await res.json()).detail ?? res.statusText);
		version = Date.now();
	}

	async function add(e: SubmitEvent) {
		e.preventDefault();
		try {
			const c = await api<Flashcard>('/v1/flashcards', {
				method: 'POST',
				body: {
					front: form.front,
					back: form.back,
					deck: form.deck || 'Default',
					elaboration: form.elaboration,
					example: form.example,
					mnemonic: form.mnemonic
				}
			});
			if (picture) await upload(c.id, picture);
			toast(t('Card added.'));
			form = { front: '', back: '', deck: form.deck, ...empty };
			picture = null;
			if (deck !== c.deck) await goto(`/cards?deck=${encodeURIComponent(c.deck)}`);
			else await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function open(c: Flashcard) {
		editing = c;
		edit = {
			front: c.front,
			back: c.back,
			deck: c.deck,
			elaboration: c.elaboration,
			example: c.example,
			mnemonic: c.mnemonic,
			palace_id: c.palace_id ?? '',
			locus: c.locus ?? 0
		};
		editOpen = true;
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		if (!editing) return;
		try {
			const { palace_id, locus, ...text } = edit;
			await api(`/v1/flashcards/${editing.id}`, {
				method: 'PATCH',
				body: palace_id ? { ...text, palace_id, locus } : { ...text, palace_id: '' }
			});
			editOpen = false;
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function act(c: Flashcard, what: 'suspend' | 'resume' | 'delete' | 'unpicture') {
		try {
			if (what === 'delete') await api(`/v1/flashcards/${c.id}`, { method: 'DELETE' });
			else if (what === 'unpicture')
				await api(`/v1/flashcards/${c.id}/image`, { method: 'DELETE' });
			else
				await api(`/v1/flashcards/${c.id}`, {
					method: 'PATCH',
					body: { suspended: what === 'suspend' }
				});
			editOpen = false;
			if (what === 'delete') deleted(t('Card deleted.'), c.id, () => load());
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const dueTotal = $derived((decks ?? []).reduce((a, d) => a + d.due, 0));
</script>

<svelte:head><title>{t('Flashcards · Axon')}</title></svelte:head>

<h1>{t('Flashcards')}</h1>
<LearnTabs />
<AgentDock {...pageAgents.learn} onapplied={() => load()} />
<p class="muted hint">
	{t(
		'Retrieval practice with spaced repetition: cards come back just before you would forget them (FSRS, the scheduler Anki uses), a few new ones a day, with decks mixed. Give a card the why, an example, and a vivid image so it sticks. The Teacher agent can write illustrated cards from your literature and permanent notes.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !decks}
	<Status kind="loading" message={t('Loading cards…')} />
{:else}
	<div class="actions">
		<Button href="/cards/review" disabled={dueTotal === 0}
			>{t('Review {dueTotal} due', { dueTotal })}</Button
		>
		<Button variant="secondary" href="/cards/review?practice=1">{t('Practise weak cards')}</Button>
		<Button variant="ghost" href="/cards/test">{t('Mixed test')}</Button>
		<Button
			variant="ghost"
			href="/agents?agent=teacher&ask={encodeURIComponent(
				t(
					'Write illustrated flashcards from my newest literature and permanent notes, with the why, an example, and a vivid memory image.'
				)
			)}">{t('✨ Illustrated cards with the Teacher')}</Button
		>
		{#if decks.length}<a class="export" href="/api/v1/flashcards/export.apkg" rel="external"
				>{t('Export all to Anki (.apkg)')}</a
			>{/if}
	</div>

	{#if decks.length}
		<ul class="decks">
			{#each decks as d (d.name)}
				<li class="card glass" class:current={d.name === deck}>
					<a class="name" href="/cards?deck={encodeURIComponent(d.name)}">{d.name}</a>
					<p class="muted small">
						{t('{total} cards · {due} due · {new} new', { total: d.total, due: d.due, new: d.new })}
					</p>
					<div class="row">
						<Button
							variant="secondary"
							href="/cards/review?deck={encodeURIComponent(d.name)}"
							disabled={d.due === 0}
							aria-label={t('Review {name}', { name: d.name })}>{t('Review')}</Button
						>
						<Button
							variant="ghost"
							href="/cards/test?deck={encodeURIComponent(d.name)}"
							aria-label={t('Practice test on {name}', { name: d.name })}>{t('Test')}</Button
						>
						<a
							class="small"
							href="/api/v1/flashcards/export.apkg?deck={encodeURIComponent(d.name)}"
							rel="external"
							aria-label={t('Export {name} to Anki', { name: d.name })}>{t('.apkg')}</a
						>
					</div>
				</li>
			{/each}
		</ul>
	{:else}
		<Status kind="empty" message={t('No cards yet. Write your first one below.')} />
	{/if}

	<AnkiSync />

	<section class="card glass" aria-labelledby="add-title">
		<h2 id="add-title">{t('Add a card')}</h2>
		<form onsubmit={add}>
			<div class="field">
				<label for="card-front">{t('Question')}</label>
				<textarea id="card-front" rows="2" maxlength="10000" required bind:value={form.front}
				></textarea>
			</div>
			<div class="field">
				<label for="card-back">{t('Answer')}</label>
				<textarea id="card-back" rows="3" maxlength="10000" bind:value={form.back}></textarea>
			</div>
			<div class="grid">
				<TextField
					id="card-deck"
					label={t('Deck')}
					list="deck-names"
					bind:value={form.deck}
					maxlength={200}
				/>
				<div class="field">
					<label for="card-picture">{t('Picture (optional)')}</label>
					<input
						id="card-picture"
						type="file"
						accept="image/png,image/jpeg"
						onchange={(e) => (picture = e.currentTarget.files?.[0] ?? null)}
					/>
				</div>
			</div>
			<details class="stick">
				<summary>{t('Make it stick (optional)')}</summary>
				<div class="field">
					<label for="card-elaboration">{t('Why is it true? How does it work?')}</label>
					<textarea id="card-elaboration" rows="2" maxlength="2000" bind:value={form.elaboration}
					></textarea>
				</div>
				<div class="field">
					<label for="card-example">{t('A concrete example')}</label>
					<textarea id="card-example" rows="2" maxlength="2000" bind:value={form.example}
					></textarea>
				</div>
				<div class="field">
					<label for="card-mnemonic">{t('A vivid memory image')}</label>
					<textarea
						id="card-mnemonic"
						rows="2"
						maxlength="2000"
						placeholder={t('Exaggerated, funny, or absurd images are remembered best.')}
						bind:value={form.mnemonic}
					></textarea>
				</div>
			</details>
			<datalist id="deck-names"
				>{#each decks as d (d.name)}<option value={d.name}></option>{/each}</datalist
			>
			<Button type="submit" variant="secondary">{t('Add card')}</Button>
		</form>
	</section>

	{#if deck}
		<section class="card glass" aria-labelledby="deck-title">
			<h2 id="deck-title">{deck}</h2>
			{#if cards.length}
				<ul class="cards">
					{#each cards as c (c.id)}
						<li class:suspended={c.suspended}>
							{#if c.has_image}<img
									src="/api/v1/flashcards/{c.id}/image?v={version}"
									alt=""
									width="64"
									height="48"
								/>{/if}
							<div class="grow">
								<strong>{c.front}</strong>
								{#if c.leech}<span
										class="leech"
										title={t(
											'Forgotten {lapses} times: rewrite it, add an example or a vivid image, or split it in two.',
											{ lapses: c.lapses }
										)}>{t('Keeps slipping')}</span
									>{/if}
								<span class="muted small"
									>{t(c.suspended ? 'Suspended · {reps} reviews' : 'Due {date} · {reps} reviews', {
										date: new Date(c.due).toLocaleDateString(appLocale()),
										reps: c.reps
									})}</span
								>
							</div>
							<Button
								variant="ghost"
								onclick={() => open(c)}
								aria-label={t('Edit {front}', { front: c.front })}>{t('Edit')}</Button
							>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="muted">{t('This deck is empty.')}</p>
			{/if}
		</section>
	{/if}
{/if}

<Dialog bind:open={editOpen} title={t('Edit card')}>
	{#if editing}
		<form onsubmit={save}>
			<div class="field">
				<label for="edit-front">{t('Question')}</label>
				<textarea id="edit-front" rows="2" maxlength="10000" required bind:value={edit.front}
				></textarea>
			</div>
			<div class="field">
				<label for="edit-back">{t('Answer')}</label>
				<textarea id="edit-back" rows="3" maxlength="10000" bind:value={edit.back}></textarea>
			</div>
			<TextField
				id="edit-deck"
				label={t('Deck')}
				list="deck-names"
				bind:value={edit.deck}
				maxlength={200}
			/>
			<details class="stick" open={!!(edit.elaboration || edit.example || edit.mnemonic)}>
				<summary>{t('Make it stick (optional)')}</summary>
				<div class="field">
					<label for="edit-elaboration">{t('Why is it true? How does it work?')}</label>
					<textarea id="edit-elaboration" rows="2" maxlength="2000" bind:value={edit.elaboration}
					></textarea>
				</div>
				<div class="field">
					<label for="edit-example">{t('A concrete example')}</label>
					<textarea id="edit-example" rows="2" maxlength="2000" bind:value={edit.example}
					></textarea>
				</div>
				<div class="field">
					<label for="edit-mnemonic">{t('A vivid memory image')}</label>
					<textarea
						id="edit-mnemonic"
						rows="2"
						maxlength="2000"
						placeholder={t('Exaggerated, funny, or absurd images are remembered best.')}
						bind:value={edit.mnemonic}
					></textarea>
				</div>
			</details>
			{#if palaces.length}
				<div class="grid">
					<div class="field">
						<label for="edit-palace">{t('Memory palace')}</label>
						<select id="edit-palace" bind:value={edit.palace_id}>
							<option value="">{t('None')}</option>
							{#each palaces as p (p.id)}<option value={p.id}>{p.title}</option>{/each}
						</select>
					</div>
					{#if editPalace}
						<div class="field">
							<label for="edit-locus">{t('Place')}</label>
							<select id="edit-locus" bind:value={edit.locus}>
								{#each editPalace.loci as l, i (i)}<option value={i}>{i + 1}. {l}</option>{/each}
							</select>
						</div>
					{/if}
				</div>
			{/if}
			<div class="field">
				<label for="edit-picture"
					>{editing.has_image ? t('Replace the picture') : t('Add a picture')}</label
				>
				<input
					id="edit-picture"
					type="file"
					accept="image/png,image/jpeg"
					onchange={async (e) => {
						const f = e.currentTarget.files?.[0];
						if (f && editing) {
							try {
								await upload(editing.id, f);
								toast('Picture saved.');
								await load();
							} catch (err) {
								toast(message(err), 'error');
							}
						}
					}}
				/>
			</div>
			<div class="row wrap">
				<Button type="submit">{t('Save')}</Button>
				<Button
					variant="secondary"
					onclick={() => editing && act(editing, editing.suspended ? 'resume' : 'suspend')}
					>{editing.suspended ? t('Resume') : t('Suspend')}</Button
				>
				{#if editing.has_image}<Button
						variant="ghost"
						onclick={() => editing && act(editing, 'unpicture')}>{t('Remove picture')}</Button
					>{/if}
				<Button variant="danger" onclick={() => editing && act(editing, 'delete')}
					>{t('Delete')}</Button
				>
			</div>
		</form>
	{/if}
</Dialog>

<style>
	.stick summary {
		cursor: pointer;
		margin: var(--space-2) 0;
		color: var(--muted);
	}
	.leech {
		font-size: var(--text-sm);
		margin-left: var(--space-2);
		padding: 0 var(--space-2);
		border-radius: 999px;
		background: var(--warning-soft, var(--accent-soft));
	}
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.actions {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin: var(--space-4) 0;
	}
	.export {
		align-self: center;
	}
	.decks {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 16rem), 1fr));
		gap: var(--space-3);
		margin: 0 0 var(--space-6);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.decks .card {
		margin: 0;
	}
	.current {
		border-color: var(--accent);
	}
	.name {
		font-weight: 650;
		color: var(--fg);
		text-decoration: none;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		align-items: center;
	}
	.wrap {
		flex-wrap: wrap;
		margin-top: var(--space-3);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
	}
	.field label {
		font-weight: 600;
	}
	textarea {
		font: inherit;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 14rem), 1fr));
		gap: var(--space-4);
	}
	.cards {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.cards li {
		display: flex;
		gap: var(--space-3);
		align-items: center;
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
	}
	.cards li.suspended {
		opacity: 0.6;
	}
	.cards img {
		object-fit: cover;
		border-radius: var(--radius-sm);
	}
	.grow {
		flex: 1;
		display: grid;
		min-width: 0;
		overflow-wrap: anywhere;
	}
</style>
