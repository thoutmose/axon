<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import { createNote, paraViews, type Note } from '$lib/notes';
	import { refreshNotes } from '$lib/noteIndex.svelte';
	import Button from '$lib/ui/Button.svelte';
	import ZettelGraph from '$lib/ZettelGraph.svelte';
	import Status from '$lib/ui/Status.svelte';
	import VaultServer from '$lib/VaultServer.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type ImportResult = {
		created: string[];
		unchanged: string[];
		conflicts: string[];
		skipped: string[];
	};

	const para = $derived(page.url.searchParams.get('para') ?? 'all');
	const tag = $derived(page.url.searchParams.get('tag'));
	const kind = $derived(page.url.searchParams.get('kind'));
	let notes = $state<Note[] | null>(null);
	let cursor = $state<string | null>(null);
	let error = $state('');
	let quick = $state('');
	let importing = $state(false);
	let report = $state<ImportResult | null>(null);

	// Where the notes live (EP-143): said at the top, so the choice between
	// Axon only and a Markdown vault kept in step both ways is never hidden.
	let vault = $state<{ host: string; dir: string; status: string } | null>(null);
	api<{ source: { host: string; dir: string; status: string } | null }>('/v1/vault/source')
		.then((r) => (vault = r.source))
		.catch(() => {});
	const vaultState: Record<string, string> = {
		waiting: 'waiting for the server to pair',
		connected: 'in step both ways',
		silent: 'no news for 15 minutes',
		failing: 'the last sync failed'
	};

	async function load(more = false) {
		error = '';
		const params = new URLSearchParams({ limit: '100', mirrored: 'false' }); // item notes live in the Axon folder
		if (para !== 'all') params.set('para', para);
		if (tag) params.set('tag', tag);
		if (kind) params.set('kind', kind);
		if (more && cursor) params.set('after', cursor);
		try {
			const res = await api<{ items: Note[]; next_cursor: string | null }>(`/v1/notes?${params}`);
			notes = more && notes ? [...notes, ...res.items] : res.items;
			cursor = res.next_cursor;
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void para;
		void tag;
		void kind;
		notes = null;
		load();
	});

	async function add(e: SubmitEvent) {
		e.preventDefault();
		try {
			const n = await createNote(`# ${quick.trim()}\n\n`);
			quick = '';
			await refreshNotes();
			await goto(`/notes/${n.id}?edit=1`);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function importZip(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		importing = true;
		try {
			report = await api<ImportResult>('/v1/notes/import', {
				method: 'POST',
				body: file,
				headers: { 'Content-Type': 'application/zip' }
			});
			await Promise.all([load(), refreshNotes()]);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			importing = false;
			input.value = '';
		}
	}

	function folder(path: string): string {
		const i = path.lastIndexOf('/');
		return i < 0 ? '' : path.slice(0, i);
	}
</script>

<svelte:head><title>{t('Notes · Axon')}</title></svelte:head>

<div class="head">
	<h1>
		{tag
			? `#${tag}`
			: kind
				? t('{kind} notes', { kind: t(`${kind[0].toUpperCase()}${kind.slice(1)}`) })
				: t('Notes')}
	</h1>
	<label class="upload" aria-busy={importing}>
		<input type="file" accept=".zip,application/zip" onchange={importZip} disabled={importing} />
		{importing ? t('Importing…') : t('Import ZIP')}
	</label>
</div>

<AgentDock {...pageAgents.notes} onapplied={() => load()} />
<p class="muted hint">
	{t('Markdown, stored exactly as you write it. Link notes with')} <code>[[note name]]</code>{t(
		'; folders like'
	)}
	<code>1-Projects/</code>
	{t('sort them into PARA.')}
</p>

<details class="where" open={page.url.searchParams.has('vault')}>
	<summary>
		{t('Where your notes live')}:
		{#if vault}
			<span class="state {vault.status}"
				>{t('Axon and {host}:{dir}', { host: vault.host, dir: vault.dir })} · {t(
					vaultState[vault.status] ?? vault.status
				)}</span
			>
		{:else}
			<span class="state">{t('In Axon only')}</span>
		{/if}
	</summary>
	<p class="muted small">
		{t(
			'With a vault, a note created, edited, renamed, or deleted here or in the files is replicated on the other side within seconds. A note changed on both sides keeps both versions; nothing is overwritten.'
		)}
	</p>
	<VaultServer />
	<details class="sync">
		<summary>{t('Or keep a folder on this computer in step by hand')}</summary>
		<p>
			{t(
				'Keep a folder of Markdown files (for example an Obsidian vault in Git) and Axon in step. Create a write'
			)} <a href="/account/tokens">{t('access token')}</a>{t(
				', save it in a file on the machine with the vault, and run:'
			)}
		</p>
		<pre><code
				>axon notes-sync -dir ~/SecondBrain -url {page.url
					.origin} -token-file ~/.axon-token -watch</code
			></pre>
		<p>
			<code>-watch</code>
			{t(
				'keeps it running and replicates every change both ways within seconds. To sync once instead, use'
			)} <code>-two-way</code>{t(', and')} <code>-git</code>
			{t(
				"to pull and push the repository around the run. When a note changed on both sides, Axon's version is saved next to yours as"
			)} <code>.axon-conflict.md</code>{t('; nothing is overwritten.')}
		</p>
	</details>
</details>

<nav aria-label={t('Note folders')} class="tabs">
	{#each paraViews as v (v.id)}
		<a href="?para={v.id}" aria-current={para === v.id ? 'page' : undefined}>{t(v.label)}</a>
	{/each}
</nav>

<form class="quick" onsubmit={add}>
	<label for="new-note" class="visually-hidden">{t('New note title')}</label>
	<input
		id="new-note"
		placeholder={t('New note title…')}
		bind:value={quick}
		maxlength="200"
		required
		autocomplete="off"
	/>
	<Button type="submit" variant="secondary">{t('Create')}</Button>
</form>

<nav class="starters" aria-label={t('Start from a template')}>
	<span class="small muted">{t('Start from a template:')}</span>
	{#each [['fleeting', t('Fleeting')], ['literature', t('Literature')], ['permanent', t('Permanent')], ['project', t('Project note')], ['area', t('Area note')], ['resource', t('Resource note')], ['archive', t('Archive note')], ['item-task', t('Task')], ['item-goal', t('Goal')], ['item-milestone', t('Milestone')], ['item-habit', t('Habit')], ['item-book', t('Book')], ['item-film', t('Film')]] as [id, label] (id)}
		<a href="?new={id}">{label}</a>
	{/each}
	<a href="/notes/templates" class="all">{t('All templates and their sections →')}</a>
</nav>

{#if para === 'all' && !tag && !kind}
	<ZettelGraph />
{/if}

{#if report}
	<div class="report" role="status">
		<p>
			{t('Imported {length} new, {length2} already here.', {
				length: report.created.length,
				length2: report.unchanged.length
			})}
			{#if report.conflicts.length}
				{t('{length} differ from the note you already have and were left untouched:', {
					length: report.conflicts.length
				})}
			{/if}
		</p>
		{#if report.conflicts.length}
			<ul>
				{#each report.conflicts as c (c)}<li><code>{c}</code></li>{/each}
			</ul>
		{/if}
		{#if report.skipped.length}
			<p class="muted">
				{t('Skipped {length} file(s) that are not Markdown notes or are over 1 MB.', {
					length: report.skipped.length
				})}
			</p>
		{/if}
		<Button variant="ghost" onclick={() => (report = null)}>{t('Dismiss')}</Button>
	</div>
{/if}

{#if error}
	<Status kind="error" message={error} retry={() => load()} />
{:else if !notes}
	<Status kind="loading" message={t('Loading notes…')} />
{:else if notes.length === 0}
	<Status
		kind="empty"
		message={para === 'all'
			? t('No notes yet. Create one above, or import a ZIP of your Markdown vault.')
			: t('No notes in this folder.')}
	/>
{:else}
	<ul class="notes">
		{#each notes as n (n.id)}
			<li>
				<a href="/notes/{n.id}">{n.title}</a>
				<span class="meta">
					{#if n.kind !== 'note'}<span class="chip">{n.kind}</span>{/if}
					<span>{folder(n.path)}</span>
				</span>
			</li>
		{/each}
	</ul>
	{#if cursor}<Button variant="secondary" onclick={() => load(true)}>{t('Show more')}</Button>{/if}
{/if}

<style>
	.starters {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
		margin: var(--space-3) 0;
	}
	.starters a {
		border: 1px solid var(--border);
		border-radius: 999px;
		padding: var(--space-1) var(--space-3);
		font-size: var(--text-sm);
		text-decoration: none;
	}
	.starters a.all {
		border: 0;
		text-decoration: underline;
	}
	.small {
		font-size: var(--text-sm);
	}
	.head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-4);
		flex-wrap: wrap;
	}
	.hint {
		font-size: var(--text-sm);
	}
	code {
		font-family: var(--font-mono);
		font-size: 0.9em;
	}
	.upload {
		display: inline-flex;
		align-items: center;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-4);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		font-weight: 600;
		cursor: pointer;
	}
	.upload:hover,
	.upload:focus-within {
		border-color: var(--accent);
	}
	.upload:focus-within {
		box-shadow: var(--focus);
	}
	.upload input {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
	}
	.tabs {
		display: flex;
		gap: var(--space-1);
		overflow-x: auto;
		margin: var(--space-4) 0;
	}
	.tabs a {
		white-space: nowrap;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		color: var(--muted);
		text-decoration: none;
	}
	.tabs a[aria-current='page'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
	}
	.quick {
		display: flex;
		gap: var(--space-2);
		margin-bottom: var(--space-6);
	}
	.quick input {
		flex: 1;
		min-width: 0;
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.report {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-6);
	}
	.report p {
		margin: 0 0 var(--space-2);
	}
	.notes {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-6);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
	}
	.notes li {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-4);
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
	}
	.notes li:last-child {
		border-bottom: none;
	}
	.notes a {
		color: var(--fg);
		text-decoration: none;
		font-weight: 500;
		overflow-wrap: anywhere;
	}
	.notes a:hover {
		color: var(--accent);
	}
	.meta {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		color: var(--muted);
		font-size: var(--text-sm);
		overflow-wrap: anywhere;
	}
	.chip {
		background: var(--accent-soft);
		color: var(--fg);
		border-radius: 999px;
		padding: 0 var(--space-2);
	}
	.where {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-3) var(--space-4);
		margin: var(--space-3) 0;
	}
	.where > summary {
		cursor: pointer;
		font-weight: 600;
	}
	.where .state {
		font-weight: 400;
		color: var(--muted);
	}
	.where .state.connected {
		color: var(--ok);
	}
	.where .state.failing,
	.where .state.silent {
		color: var(--warn);
	}
	.sync {
		margin-top: var(--space-4);
		color: var(--muted);
	}
	.sync summary {
		cursor: pointer;
		color: var(--fg);
		font-weight: 600;
	}
	.sync pre {
		overflow-x: auto;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-3);
	}
</style>
