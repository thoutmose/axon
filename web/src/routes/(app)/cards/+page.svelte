<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import type { Flashcard } from '$lib/flashcards';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Deck = { name: string; total: number; due: number; new: number };
	const deck = $derived(page.url.searchParams.get('deck') ?? '');
	let decks = $state<Deck[] | null>(null);
	let cards = $state<Flashcard[]>([]);
	let error = $state('');
	let form = $state({ front: '', back: '', deck: '' });
	let picture = $state<File | null>(null);
	let editing = $state<Flashcard | null>(null);
	let editOpen = $state(false);
	let edit = $state({ front: '', back: '', deck: '' });
	let version = $state(Date.now());

	async function load() {
		error = '';
		try {
			decks = (await api<{ items: Deck[] }>('/v1/decks')).items;
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
				body: { front: form.front, back: form.back, deck: form.deck || 'Default' }
			});
			if (picture) await upload(c.id, picture);
			toast('Card added.');
			form = { front: '', back: '', deck: form.deck };
			picture = null;
			if (deck !== c.deck) await goto(`/cards?deck=${encodeURIComponent(c.deck)}`);
			else await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function open(c: Flashcard) {
		editing = c;
		edit = { front: c.front, back: c.back, deck: c.deck };
		editOpen = true;
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		if (!editing) return;
		try {
			await api(`/v1/flashcards/${editing.id}`, { method: 'PATCH', body: edit });
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
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const dueTotal = $derived((decks ?? []).reduce((a, d) => a + d.due, 0));
</script>

<svelte:head><title>Flashcards · Axon</title></svelte:head>

<h1>Flashcards</h1>
<LearnTabs />
<p class="muted hint">
	Retrieval practice with spaced repetition: cards come back just before you would forget them
	(FSRS, the scheduler Anki uses). Ask the Teacher agent to write cards from your notes.
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !decks}
	<Status kind="loading" message="Loading cards…" />
{:else}
	<div class="actions">
		<Button href="/cards/review" disabled={dueTotal === 0}>Review {dueTotal} due</Button>
		{#if decks.length}<a class="export" href="/api/v1/flashcards/export.apkg" rel="external"
				>Export all to Anki (.apkg)</a
			>{/if}
	</div>

	{#if decks.length}
		<ul class="decks">
			{#each decks as d (d.name)}
				<li class="card glass" class:current={d.name === deck}>
					<a class="name" href="/cards?deck={encodeURIComponent(d.name)}">{d.name}</a>
					<p class="muted small">{d.total} cards · {d.due} due · {d.new} new</p>
					<div class="row">
						<Button
							variant="secondary"
							href="/cards/review?deck={encodeURIComponent(d.name)}"
							disabled={d.due === 0}
							aria-label="Review {d.name}">Review</Button
						>
						<Button
							variant="ghost"
							href="/cards/test?deck={encodeURIComponent(d.name)}"
							aria-label="Practice test on {d.name}">Test</Button
						>
						<a
							class="small"
							href="/api/v1/flashcards/export.apkg?deck={encodeURIComponent(d.name)}"
							rel="external"
							aria-label="Export {d.name} to Anki">.apkg</a
						>
					</div>
				</li>
			{/each}
		</ul>
	{:else}
		<Status kind="empty" message="No cards yet. Write your first one below." />
	{/if}

	<section class="card glass" aria-labelledby="add-title">
		<h2 id="add-title">Add a card</h2>
		<form onsubmit={add}>
			<div class="field">
				<label for="card-front">Question</label>
				<textarea id="card-front" rows="2" maxlength="10000" required bind:value={form.front}
				></textarea>
			</div>
			<div class="field">
				<label for="card-back">Answer</label>
				<textarea id="card-back" rows="3" maxlength="10000" bind:value={form.back}></textarea>
			</div>
			<div class="grid">
				<TextField
					id="card-deck"
					label="Deck"
					list="deck-names"
					bind:value={form.deck}
					maxlength={200}
				/>
				<div class="field">
					<label for="card-picture">Picture (optional)</label>
					<input
						id="card-picture"
						type="file"
						accept="image/png,image/jpeg"
						onchange={(e) => (picture = e.currentTarget.files?.[0] ?? null)}
					/>
				</div>
			</div>
			<datalist id="deck-names"
				>{#each decks as d (d.name)}<option value={d.name}></option>{/each}</datalist
			>
			<Button type="submit" variant="secondary">Add card</Button>
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
								<span class="muted small"
									>{c.suspended ? 'Suspended' : `Due ${new Date(c.due).toLocaleDateString()}`} · {c.reps}
									reviews</span
								>
							</div>
							<Button variant="ghost" onclick={() => open(c)} aria-label="Edit {c.front}"
								>Edit</Button
							>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="muted">This deck is empty.</p>
			{/if}
		</section>
	{/if}
{/if}

<Dialog bind:open={editOpen} title="Edit card">
	{#if editing}
		<form onsubmit={save}>
			<div class="field">
				<label for="edit-front">Question</label>
				<textarea id="edit-front" rows="2" maxlength="10000" required bind:value={edit.front}
				></textarea>
			</div>
			<div class="field">
				<label for="edit-back">Answer</label>
				<textarea id="edit-back" rows="3" maxlength="10000" bind:value={edit.back}></textarea>
			</div>
			<TextField
				id="edit-deck"
				label="Deck"
				list="deck-names"
				bind:value={edit.deck}
				maxlength={200}
			/>
			<div class="field">
				<label for="edit-picture"
					>{editing.has_image ? 'Replace the picture' : 'Add a picture'}</label
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
				<Button type="submit">Save</Button>
				<Button
					variant="secondary"
					onclick={() => editing && act(editing, editing.suspended ? 'resume' : 'suspend')}
					>{editing.suspended ? 'Resume' : 'Suspend'}</Button
				>
				{#if editing.has_image}<Button
						variant="ghost"
						onclick={() => editing && act(editing, 'unpicture')}>Remove picture</Button
					>{/if}
				<Button variant="danger" onclick={() => editing && act(editing, 'delete')}>Delete</Button>
			</div>
		</form>
	{/if}
</Dialog>

<style>
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
