<script lang="ts">
	import NoteAnalysis from '$lib/NoteAnalysis.svelte';
	import { page } from '$app/state';
	import { beforeNavigate, goto } from '$app/navigation';
	import { onDestroy } from 'svelte';
	import { ApiError, api, apiResponse, message } from '$lib/api';
	import Connections from '$lib/Connections.svelte';
	import MarkdownEditor from '$lib/MarkdownEditor.svelte';
	import { noteIndex, refreshNotes } from '$lib/noteIndex.svelte';
	import { createNote, render, stem, wikilinks, type Note } from '$lib/notes';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Mention = { id: string; title: string; path: string };

	const id = $derived(page.params.id ?? '');
	const editing = $derived(page.url.searchParams.get('edit') === '1');
	let note = $state<Note | null>(null);
	let etag = '';
	let error = $state('');
	let path = $state('');
	let content = $state('');
	let saving = $state(false);
	let conflict = $state(false);
	let confirmDelete = $state(false);
	let mentions = $state<Mention[]>([]);
	let timer: ReturnType<typeof setTimeout> | undefined;

	// Every note by lower-cased stem and title (the stem wins, as on the
	// server), so links resolve as you type.
	const names = $derived.by(() => {
		const m = new Map<string, string>();
		for (const n of noteIndex.notes) m.set(n.title.toLowerCase(), n.id);
		for (const n of noteIndex.notes) m.set(stem(n.path).toLowerCase(), n.id);
		return m;
	});
	const unsaved = $derived(!!note && content !== (note.content ?? ''));
	const moved = $derived(!!note && path !== note.path);
	const html = $derived(
		render(
			editing ? content : (note?.content ?? ''),
			(t) => names.get(t),
			(id) => noteIndex.notes.find((n) => n.id === id)?.title
		)
	);
	const missing = $derived(
		wikilinks(editing ? content : (note?.content ?? '')).filter((t) => !names.has(t.toLowerCase()))
	);
	const words = $derived((content.match(/\S+/g) ?? []).length);

	async function load() {
		error = '';
		conflict = false;
		try {
			const res = await apiResponse<Note>(`/v1/notes/${id}`);
			note = res.data;
			etag = res.etag;
			path = note.path;
			content = note.content ?? '';
			mentions = (await api<{ items: Mention[] }>(`/v1/notes/${id}/mentions`)).items;
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void id;
		note = null;
		load();
	});

	// Autosave the text a moment after typing stops; moving the file is explicit.
	$effect(() => {
		void content;
		clearTimeout(timer);
		if (editing && unsaved && !conflict) timer = setTimeout(() => save(false), 1200);
	});
	onDestroy(() => clearTimeout(timer));

	beforeNavigate((nav) => {
		if ((unsaved || moved) && !confirm('Leave without saving your changes?')) nav.cancel();
	});

	function setEditing(on: boolean) {
		const u = new URL(page.url);
		if (on) u.searchParams.set('edit', '1');
		else u.searchParams.delete('edit');
		goto(u, { keepFocus: true, noScroll: true, replaceState: true });
	}

	// Vim mode comes from Preferences → Keyboard.
	let keyboard = $state({ vim: false });
	api<{ keyboard: { vim: boolean } }>('/v1/me/settings')
		.then((s) => (keyboard = s.keyboard))
		.catch(() => {});

	// A save asked for while one is in flight runs right after it, so the
	// latest text always reaches the server (an autosave must not swallow :w).
	let saveAgain = false;
	async function save(withPath: boolean, version = etag) {
		if (!note) return;
		if (saving) {
			saveAgain = true;
			return;
		}
		saving = true;
		const sent = content;
		try {
			const res = await apiResponse<Note>(`/v1/notes/${id}`, {
				method: 'PUT',
				headers: { 'If-Match': version },
				body: { path: withPath ? path : note.path, content: sent }
			});
			note = res.data;
			etag = res.etag;
			if (withPath) {
				toast('Moved.');
				await refreshNotes();
			} else if (note.title !== noteIndex.notes.find((n) => n.id === id)?.title) {
				await refreshNotes();
			}
		} catch (err) {
			if (err instanceof ApiError && err.status === 412) conflict = true;
			else toast(message(err), 'error');
		} finally {
			saving = false;
			if (saveAgain && content !== sent) {
				saveAgain = false;
				save(false);
			}
			saveAgain = false;
		}
	}

	/** After a conflict: keep this version, replacing the one saved elsewhere. */
	async function keepMine() {
		const { etag: latest } = await apiResponse<Note>(`/v1/notes/${id}`);
		conflict = false;
		await save(false, latest);
	}

	/** After a conflict: take the saved version; this edit is copied first. */
	async function takeTheirs() {
		await navigator.clipboard?.writeText(content).catch(() => {});
		toast('Your version was copied to the clipboard.');
		await load();
	}

	function onkeydown(e: KeyboardEvent) {
		if (editing && (e.ctrlKey || e.metaKey) && e.key === 's') {
			e.preventDefault();
			clearTimeout(timer);
			save(moved);
		}
	}

	async function create(target: string) {
		try {
			await createNote(`# ${target}\n\n`);
			toast(`Created “${target}”.`);
			await refreshNotes();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	/** Turns the first plain mention of this note's title into a [[wikilink]]. */
	async function linkMention(m: Mention) {
		if (!note) return;
		try {
			const res = await apiResponse<Note>(`/v1/notes/${m.id}`);
			const text = res.data.content ?? '';
			const title = note.title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
			const re = new RegExp(`(?<!\\[\\[[^\\]]*)\\b(${title})\\b`, 'i');
			if (!re.test(text)) return;
			await api(`/v1/notes/${m.id}`, {
				method: 'PUT',
				headers: { 'If-Match': res.etag },
				body: { path: res.data.path, content: text.replace(re, '[[$1]]') }
			});
			toast(`Linked from “${m.title}”.`);
			mentions = mentions.filter((x) => x.id !== m.id);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove() {
		try {
			await api(`/v1/notes/${id}`, { method: 'DELETE' });
			note = null; // nothing left to lose
			toast('Deleted.');
			await refreshNotes();
			await goto('/notes');
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>{note?.title ?? 'Note'} · Axon</title></svelte:head>
<svelte:window {onkeydown} />

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !note}
	<Status kind="loading" message="Loading note…" />
{:else}
	<div class="head">
		<div>
			<p class="muted path">
				{note.path}{#if note.kind !== 'note'}
					· <span class="kind">{note.kind}</span>{/if}
			</p>
			<h1>{note.title}</h1>
			{#if note.tags.length}
				<p class="tags">
					{#each note.tags as t (t)}<a href="/notes?tag={encodeURIComponent(t)}">#{t}</a>{/each}
				</p>
			{/if}
		</div>
		<div class="actions">
			{#if editing}
				<Button variant="secondary" onclick={() => setEditing(false)}>Read</Button>
			{:else}
				<Button variant="secondary" onclick={() => setEditing(true)}>Edit</Button>
			{/if}
			<Button variant="danger" onclick={() => (confirmDelete = true)}>Delete</Button>
		</div>
	</div>

	{#if conflict}
		<div class="conflict" role="alert">
			<p>
				<strong>This note was changed somewhere else</strong> (another tab, device, or your vault sync).
				Nothing has been overwritten.
			</p>
			<div class="actions">
				<Button onclick={keepMine}>Keep my version</Button>
				<Button variant="secondary" onclick={takeTheirs}>Use the saved version</Button>
			</div>
		</div>
	{/if}

	{#if editing}
		<div class="split">
			<div class="pane">
				<MarkdownEditor
					id="note-content"
					bind:value={content}
					vim={keyboard.vim}
					onsave={() => save(false)}
				/>
				<p class="status muted small" aria-live="polite">
					{saving
						? 'Saving…'
						: conflict
							? 'Not saved: see above'
							: unsaved
								? 'Unsaved changes'
								: 'All changes saved'}
					· {words} words · Ctrl+S saves · type [[ to link a note
				</p>
				<details class="move">
					<summary>Rename or move the file</summary>
					<label for="note-path">File path</label>
					<div class="row">
						<input
							id="note-path"
							bind:value={path}
							required
							maxlength="500"
							autocomplete="off"
							spellcheck="false"
						/>
						<Button variant="secondary" disabled={!moved} onclick={() => save(true)}>Move</Button>
					</div>
					<p class="muted small">Links to this note in your other notes are updated.</p>
				</details>
			</div>
			<div class="pane">
				<h2 class="visually-hidden">Preview</h2>
				<!-- render() escapes the note, so it cannot inject markup. -->
				<article class="markdown glass">{@html html}</article>
			</div>
		</div>
	{:else}
		<!-- render() escapes the note, so it cannot inject markup. -->
		<article class="markdown glass reading">{@html html}</article>
	{/if}

	{#if missing.length}
		<section class="card" aria-labelledby="missing-title">
			<h2 id="missing-title">Linked, but not written yet</h2>
			<ul class="rows">
				{#each missing as t (t)}
					<li>
						<span>{t}</span>
						<Button variant="ghost" onclick={() => create(t)}>Create note</Button>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	{#if mentions.length}
		<section class="card" aria-labelledby="mentions-title">
			<h2 id="mentions-title">Unlinked mentions</h2>
			<p class="muted small">These notes name “{note.title}” without linking to it.</p>
			<ul class="rows">
				{#each mentions as m (m.id)}
					<li>
						<a href="/notes/{m.id}">{m.title}</a>
						<Button variant="ghost" onclick={() => linkMention(m)} aria-label="Link from {m.title}"
							>Link</Button
						>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	<NoteAnalysis id={note.id} />
	<Connections id={note.id} type="note" />
{/if}

<Dialog bind:open={confirmDelete} title="Delete this note?">
	<p>
		“{note?.title}” and its links are removed. A synced vault deletes the file on its next two-way
		sync.
	</p>
	<div class="dialog-actions">
		<Button variant="secondary" onclick={() => (confirmDelete = false)}>Cancel</Button>
		<Button variant="danger" onclick={remove}>Delete</Button>
	</div>
</Dialog>

<style>
	.head {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: var(--space-4);
		flex-wrap: wrap;
		margin-bottom: var(--space-4);
	}
	h1 {
		overflow-wrap: anywhere;
		margin-bottom: var(--space-2);
	}
	.actions,
	.dialog-actions,
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.dialog-actions {
		justify-content: flex-end;
	}
	.path {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		overflow-wrap: anywhere;
		margin: 0 0 var(--space-1);
	}
	.kind {
		text-transform: capitalize;
	}
	.tags {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		margin: 0;
	}
	.tags a {
		font-size: var(--text-sm);
		padding: 0 var(--space-2);
		border-radius: 999px;
		background: var(--accent-soft);
		color: var(--fg);
		text-decoration: none;
	}
	.small {
		font-size: var(--text-sm);
	}
	.conflict {
		border: 1px solid var(--warn);
		background: var(--surface-solid);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.split {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 24rem), 1fr));
		gap: var(--space-4);
		align-items: start;
		margin-bottom: var(--space-6);
	}
	.pane {
		min-width: 0;
	}
	.split .markdown {
		position: sticky;
		top: 4.5rem;
		max-height: calc(100vh - 6rem);
		overflow-y: auto;
		margin: 0;
	}
	.status {
		margin: var(--space-2) 0;
	}
	.move summary {
		cursor: pointer;
		color: var(--accent);
		font-size: var(--text-sm);
	}
	.move label {
		display: block;
		font-weight: 600;
		margin: var(--space-2) 0 var(--space-1);
	}
	.move input {
		flex: 1;
		min-width: 0;
		font: inherit;
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.markdown {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-6);
		margin-bottom: var(--space-6);
		overflow-wrap: anywhere;
		line-height: 1.7;
	}
	.reading {
		font-size: 1.0625rem;
	}
	.markdown :global(:first-child) {
		margin-top: 0;
	}
	.markdown :global(.props) {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: var(--space-1) var(--space-4);
		font-size: var(--text-sm);
		color: var(--muted);
		padding-bottom: var(--space-4);
		border-bottom: 1px solid var(--border);
	}
	.markdown :global(dd) {
		margin: 0;
	}
	.markdown :global(pre) {
		overflow-x: auto;
		background: var(--bg);
		border-radius: var(--radius);
		padding: var(--space-3);
	}
	.markdown :global(code) {
		font-family: var(--font-mono);
		font-size: 0.9em;
	}
	.markdown :global(blockquote) {
		margin: var(--space-3) 0;
		padding-left: var(--space-4);
		border-left: 3px solid var(--border);
		color: var(--muted);
	}
	.markdown :global(.callout) {
		margin: var(--space-4) 0;
		padding: var(--space-3) var(--space-4);
		border-radius: var(--radius);
		border-left: 4px solid var(--accent);
		background: var(--accent-soft);
	}
	.markdown :global(.callout p) {
		margin: 0 0 var(--space-1);
	}
	.markdown :global(.callout-title) {
		font-weight: 650;
	}
	.markdown :global(.callout.warning),
	.markdown :global(.callout.caution),
	.markdown :global(.callout.danger) {
		border-left-color: var(--warn);
	}
	.markdown :global(.callout.tip),
	.markdown :global(.callout.success) {
		border-left-color: var(--ok);
	}
	.markdown :global(.table) {
		overflow-x: auto;
		margin: var(--space-4) 0;
	}
	.markdown :global(table) {
		border-collapse: collapse;
		width: 100%;
		font-size: var(--text-sm);
	}
	.markdown :global(th),
	.markdown :global(td) {
		border: 1px solid var(--border);
		padding: var(--space-2) var(--space-3);
		text-align: left;
	}
	.markdown :global(th) {
		background: var(--accent-soft);
	}
	.markdown :global(mark) {
		background: rgb(255 214 10 / 0.45);
		color: inherit;
		border-radius: 3px;
		padding: 0 2px;
	}
	.markdown :global(.tag) {
		color: var(--accent);
		text-decoration: none;
	}
	.markdown :global(.wikilink) {
		color: var(--accent);
	}
	.markdown :global(.wikilink.unresolved) {
		color: var(--muted);
		text-decoration: underline dotted;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	.card h2 {
		font-size: var(--text-lg);
	}
	.rows {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.rows li {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
	}
</style>
