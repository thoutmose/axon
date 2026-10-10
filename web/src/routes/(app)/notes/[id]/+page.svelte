<script lang="ts">
	import { page } from '$app/state';
	import { beforeNavigate, goto } from '$app/navigation';
	import { ApiError, api, apiResponse, message, type Schemas } from '$lib/api';
	import { createNote, render, stem, wikilinks, type Note } from '$lib/notes';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Link = Schemas['Link'];

	const id = $derived(page.params.id ?? '');
	const editing = $derived(page.url.searchParams.get('edit') === '1');
	let note = $state<Note | null>(null);
	let etag = '';
	let error = $state('');
	let path = $state('');
	let content = $state('');
	let busy = $state(false);
	let confirmDelete = $state(false);
	let outgoing = $state<Link[]>([]);
	let backlinks = $state<Link[]>([]);
	// lower-cased stem or title → note id, for the notes this one links to
	let names = $state(new Map<string, string>());

	const dirty = $derived(!!note && (path !== note.path || content !== (note.content ?? '')));
	const html = $derived(render(editing ? content : (note?.content ?? ''), (t) => names.get(t)));
	const missing = $derived(
		wikilinks(editing ? content : (note?.content ?? '')).filter((t) => !names.has(t.toLowerCase()))
	);

	async function load() {
		error = '';
		try {
			const res = await apiResponse<Note>(`/v1/notes/${id}`);
			note = res.data;
			etag = res.etag;
			path = note.path;
			content = note.content ?? '';
			await loadLinks();
		} catch (err) {
			error = message(err);
		}
	}

	async function loadLinks() {
		const [out, back] = await Promise.all([
			api<{ items: Link[] }>(`/v1/entities/${id}/links?limit=100`),
			api<{ items: Link[] }>(`/v1/entities/${id}/links?direction=incoming&limit=100`)
		]);
		outgoing = out.items;
		backlinks = back.items;
		const linked = await Promise.all(
			out.items
				.filter((l) => l.entity.type === 'note')
				.map((l) => api<Note>(`/v1/notes/${l.entity.id}`).catch(() => null))
		);
		const m = new Map<string, string>();
		for (const n of linked) {
			if (!n) continue;
			m.set(n.title.toLowerCase(), n.id);
			m.set(stem(n.path).toLowerCase(), n.id); // the stem wins, as on the server
		}
		names = m;
	}

	$effect(() => {
		void id;
		note = null;
		load();
	});

	beforeNavigate((nav) => {
		if (dirty && !confirm('Leave without saving your changes?')) nav.cancel();
	});

	function setEditing(on: boolean) {
		const u = new URL(page.url);
		if (on) u.searchParams.set('edit', '1');
		else u.searchParams.delete('edit');
		goto(u, { keepFocus: true, noScroll: true, replaceState: true });
	}

	async function save(e?: SubmitEvent) {
		e?.preventDefault();
		busy = true;
		try {
			const res = await apiResponse<Note>(`/v1/notes/${id}`, {
				method: 'PUT',
				headers: { 'If-Match': etag },
				body: { path, content }
			});
			note = res.data;
			etag = res.etag;
			toast('Saved.');
			await loadLinks();
		} catch (err) {
			toast(
				err instanceof ApiError && err.status === 412
					? 'This note changed elsewhere since you opened it. Copy your edit, reload, and merge.'
					: message(err),
				'error'
			);
		} finally {
			busy = false;
		}
	}

	function onkeydown(e: KeyboardEvent) {
		if (editing && (e.ctrlKey || e.metaKey) && e.key === 's') {
			e.preventDefault();
			if (!busy) save();
		}
	}

	async function create(target: string) {
		try {
			await createNote(`# ${target}\n\n`);
			toast(`Created “${target}”.`);
			await loadLinks();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove() {
		busy = true;
		try {
			await api(`/v1/notes/${id}`, { method: 'DELETE' });
			note = null; // nothing left to lose
			toast('Deleted.');
			await goto('/notes');
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	function href(l: Link): string | null {
		return l.entity.type === 'note' ? `/notes/${l.entity.id}` : null;
	}
</script>

<svelte:head><title>{note?.title ?? 'Note'} · Axon</title></svelte:head>
<svelte:window {onkeydown} />

<p class="crumbs"><a href="/notes">← Notes</a></p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !note}
	<Status kind="loading" message="Loading note…" />
{:else}
	<div class="head">
		<h1>{note.title}</h1>
		<div class="actions">
			{#if editing}
				<Button variant="secondary" onclick={() => setEditing(false)}>Read</Button>
			{:else}
				<Button variant="secondary" onclick={() => setEditing(true)}>Edit</Button>
			{/if}
			<Button variant="danger" onclick={() => (confirmDelete = true)}>Delete</Button>
		</div>
	</div>
	<p class="muted path">
		{note.path}{#if note.kind !== 'note'}
			· {note.kind}{/if}
	</p>

	{#if editing}
		<form class="editor" onsubmit={save}>
			<label for="note-path">File</label>
			<input
				id="note-path"
				bind:value={path}
				required
				maxlength="500"
				autocomplete="off"
				spellcheck="false"
				aria-describedby="note-path-hint"
			/>
			<p class="muted small" id="note-path-hint">
				Moving or renaming the file updates <code>[[links]]</code> to it in your other notes.
			</p>
			<label for="note-content">Markdown</label>
			<textarea id="note-content" bind:value={content} rows="18" spellcheck="true"></textarea>
			<div class="save">
				<Button type="submit" {busy} disabled={!dirty}>Save</Button>
				<span class="muted small"
					>{dirty ? 'Unsaved changes · Ctrl+S saves' : 'All changes saved'}</span
				>
			</div>
		</form>
		<h2 class="preview-title">Preview</h2>
	{/if}

	<!-- render() escapes the note, so it cannot inject markup. -->
	<article class="markdown">{@html html}</article>

	{#if missing.length}
		<section class="links" aria-labelledby="missing-title">
			<h2 id="missing-title">Linked, but not written yet</h2>
			<ul>
				{#each missing as t (t)}
					<li>
						<span>{t}</span>
						<Button variant="ghost" onclick={() => create(t)}>Create note</Button>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	<section class="links" aria-labelledby="backlinks-title">
		<h2 id="backlinks-title">Backlinks</h2>
		{#if backlinks.length === 0}
			<p class="muted">No other note links here yet.</p>
		{:else}
			<ul>
				{#each backlinks as l (l.id)}
					<li>
						{#if href(l)}<a href={href(l)}>{l.entity.title}</a>{:else}<span>{l.entity.title}</span
							>{/if}
						<span class="muted small">{l.entity.type}</span>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	{#if outgoing.length}
		<section class="links" aria-labelledby="outgoing-title">
			<h2 id="outgoing-title">Links</h2>
			<ul>
				{#each outgoing as l (l.id)}
					<li>
						{#if href(l)}<a href={href(l)}>{l.entity.title}</a>{:else}<span>{l.entity.title}</span
							>{/if}
						<span class="muted small">{l.entity.type}</span>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
{/if}

<Dialog bind:open={confirmDelete} title="Delete this note?">
	<p>
		“{note?.title}” and its links are removed. A synced vault deletes the file on its next two-way
		sync.
	</p>
	<div class="dialog-actions">
		<Button variant="secondary" onclick={() => (confirmDelete = false)}>Cancel</Button>
		<Button variant="danger" onclick={remove} {busy}>Delete</Button>
	</div>
</Dialog>

<style>
	.crumbs {
		margin: 0 0 var(--space-2);
	}
	.crumbs a {
		color: var(--muted);
		text-decoration: none;
	}
	.head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-4);
		flex-wrap: wrap;
	}
	h1 {
		overflow-wrap: anywhere;
	}
	.actions,
	.dialog-actions {
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
	}
	.small {
		font-size: var(--text-sm);
	}
	.editor {
		display: grid;
		gap: var(--space-2);
		margin: var(--space-4) 0;
	}
	.editor label {
		font-weight: 600;
	}
	.editor p {
		margin: 0 0 var(--space-2);
	}
	.editor input,
	.editor textarea {
		font: inherit;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		min-height: 2.75rem;
		width: 100%;
	}
	.editor textarea {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		line-height: 1.6;
		resize: vertical;
	}
	.save {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	.preview-title {
		font-size: var(--text-base);
		color: var(--muted);
		margin-top: var(--space-6);
	}
	.markdown {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-6);
		margin-bottom: var(--space-6);
		overflow-wrap: anywhere;
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
	.markdown :global(.wikilink) {
		color: var(--accent);
	}
	.markdown :global(.wikilink.unresolved) {
		color: var(--muted);
		text-decoration: underline dotted;
	}
	.links {
		margin-bottom: var(--space-6);
	}
	.links h2 {
		font-size: var(--text-lg);
	}
	.links ul {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.links li {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
	}
	.links a {
		overflow-wrap: anywhere;
	}
</style>
