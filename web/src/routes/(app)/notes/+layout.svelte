<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, message } from '$lib/api';
	import NewNoteDialog from '$lib/NewNoteDialog.svelte';
	import NoteFolder from '$lib/NoteFolder.svelte';
	import Icon from '$lib/ui/Icon.svelte';
	import { openDaily } from '$lib/newNote';
	import { noteIndex, refreshNotes } from '$lib/noteIndex.svelte';
	import { tree } from '$lib/noteTree';
	import { toast } from '$lib/ui/toast.svelte';

	let { children } = $props();
	let creating = $state(false);
	let newKind = $state('permanent');
	let query = $state('');
	let tags = $state<{ tag: string; count: number }[]>([]);
	const groups = $derived(tree(noteIndex.notes));

	// The sidebar can be hidden (owner request), and stays as it was left on
	// this device.
	let hidden = $state(false);
	try {
		hidden = localStorage.getItem('notes.sidebar') === 'hidden';
	} catch {
		// no storage: shown
	}
	function toggleSidebar() {
		hidden = !hidden;
		try {
			localStorage.setItem('notes.sidebar', hidden ? 'hidden' : 'shown');
		} catch {
			// not remembered, still toggled
		}
	}

	// The command center opens "New note" with ?new=1; ?new=<template> picks one.
	$effect(() => {
		const wanted = page.url.searchParams.get('new');
		if (wanted) {
			if (wanted !== '1') newKind = wanted;
			creating = true;
			const u = new URL(page.url);
			u.searchParams.delete('new');
			goto(u, { replaceState: true, keepFocus: true, noScroll: true });
		}
	});

	$effect(() => {
		refreshNotes().catch((err) => toast(message(err), 'error'));
		api<{ items: { tag: string; count: number }[] }>('/v1/notes/tags')
			.then((r) => (tags = r.items))
			.catch(() => {});
	});

	function search(e: SubmitEvent) {
		e.preventDefault();
		if (query.trim()) goto(`/notes/search?q=${encodeURIComponent(query.trim())}`);
	}

	async function daily() {
		try {
			await openDaily();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<div class="workspace" class:wide={hidden}>
	<button
		type="button"
		class="toggle"
		aria-expanded={!hidden}
		aria-controls="notes-sidebar"
		title={hidden ? t('Show the notes panel') : t('Hide the notes panel')}
		onclick={toggleSidebar}
		>{hidden ? '»' : '«'}<span class="visually-hidden"
			>{hidden ? t('Show the notes panel') : t('Hide the notes panel')}</span
		></button
	>
	<aside id="notes-sidebar" class="sidebar glass" aria-label={t('Notes')} {hidden}>
		<div class="actions">
			<button class="primary" onclick={() => (creating = true)}>{t('＋ New note')}</button>
			<button onclick={daily}>{t("Today's note")}</button>
		</div>
		<form role="search" onsubmit={search}>
			<label for="note-search" class="visually-hidden">{t('Search notes')}</label>
			<input id="note-search" type="search" placeholder={t('Search notes…')} bind:value={query} />
		</form>
		<a class="explore" href="/notes/explore"
			><Icon name="graph" />{t('Explore and knowledge graph')}</a
		>
		<a class="explore" href="/notes/analysis"
			><Icon name="analysis" />{t('Analysis: topics and concepts')}</a
		>
		<a class="explore" href="/notes/templates"><Icon name="templates" />{t('Templates')}</a>
		<nav aria-label={t('Folders')}>
			{#each groups as g (g.para)}
				{#if g.root.count > 0}
					<details open>
						<summary>{t(g.label)} <span class="count">{g.root.count}</span></summary>
						<NoteFolder folder={g.root} />
					</details>
				{/if}
			{/each}
			{#if noteIndex.loaded && noteIndex.notes.length === 0}
				<p class="muted small">{t('No notes yet.')}</p>
			{/if}
		</nav>
		{#if tags.length}
			<details class="tags" open>
				<summary>{t('Tags')}</summary>
				<ul>
					{#each tags.slice(0, 40) as entry (entry.tag)}
						<li>
							<a href="/notes?tag={encodeURIComponent(entry.tag)}"
								>#{entry.tag} <span class="count">{entry.count}</span></a
							>
						</li>
					{/each}
				</ul>
			</details>
		{/if}
	</aside>
	<div class="content">
		{@render children()}
	</div>
</div>

<NewNoteDialog bind:open={creating} bind:kind={newKind} />

<style>
	.workspace {
		position: relative;
		display: grid;
		grid-template-columns: 17rem minmax(0, 1fr);
		gap: var(--space-8);
		align-items: start;
	}
	.workspace.wide {
		grid-template-columns: minmax(0, 1fr);
	}
	.toggle {
		position: absolute;
		top: var(--space-2);
		left: calc(17rem + var(--space-8) / 2 - 1rem); /* in the gap beside the sidebar */
		z-index: 2;
		width: 2rem;
		height: 2rem;
		border-radius: 50%;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		font: inherit;
		font-weight: 700;
		cursor: pointer;
	}
	.wide .toggle {
		left: 0;
	}
	.wide .content {
		padding-left: 2.75rem;
	}
	@media (max-width: 60rem) {
		.workspace {
			grid-template-columns: minmax(0, 1fr);
			gap: var(--space-4);
		}
		.sidebar {
			position: static;
			max-height: none;
		}
		.toggle {
			display: none;
		}
		.sidebar nav,
		.sidebar .tags {
			display: none;
		}
	}
	.sidebar {
		position: sticky;
		top: 4.5rem;
		max-height: calc(100vh - 6rem);
		overflow-y: auto;
		overflow-x: hidden; /* long names are cut, never scrolled sideways */
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-3);
		display: grid;
		gap: var(--space-3);
	}
	.sidebar[hidden] {
		display: none;
	}
	.sidebar > *,
	.sidebar nav > details {
		min-width: 0;
	}
	.actions {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--space-2);
	}
	.actions button {
		font: inherit;
		font-size: var(--text-sm);
		font-weight: 600;
		min-height: 2.25rem;
		border-radius: 999px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		cursor: pointer;
	}
	.actions .primary {
		background: var(--accent);
		color: var(--accent-fg);
		border-color: transparent;
	}
	input {
		width: 100%;
		font: inherit;
		font-size: var(--text-sm);
		min-height: 2.25rem;
		padding: var(--space-1) var(--space-3);
		border-radius: 999px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	nav > details > summary,
	.tags > summary {
		cursor: pointer;
		font-size: var(--text-sm);
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--muted);
		padding: var(--space-1);
		display: flex;
	}
	nav > details {
		margin-bottom: var(--space-2);
	}
	.count {
		margin-left: auto;
		font-weight: 400;
		color: var(--muted);
	}
	.tags ul {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
	}
	.tags a {
		font-size: var(--text-sm);
		display: inline-flex;
		gap: var(--space-1);
		padding: 0 var(--space-2);
		border-radius: 999px;
		background: var(--accent-soft);
		color: var(--fg);
		text-decoration: none;
	}
	.small {
		font-size: var(--text-sm);
	}
	.content {
		min-width: 0;
	}
	.explore {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin: var(--space-2) 0 var(--space-3);
		font-size: var(--text-sm);
	}
</style>
