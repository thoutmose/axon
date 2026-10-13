<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { deleted } from '$lib/undo';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';
	import { ApiError, api, apiResponse, message } from '$lib/api';
	import Connections from '$lib/Connections.svelte';
	import { refreshNotes } from '$lib/noteIndex.svelte';
	import type { Note } from '$lib/notes';
	import { neighbour, reorder, wip, type Card, type Column, type Project } from '$lib/projects';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Doc = { id: string; title: string; path: string; updated_at: string };

	const id = $derived(page.params.id ?? '');
	let project = $state<Project | null>(null);
	let columns = $state<Column[]>([]);
	let docs = $state<Doc[]>([]);
	let error = $state('');
	let announce = $state('');
	let adding = $state<Record<string, string>>({});
	let addKind = $state('issue');
	let addParent = $state('');
	let dragging = $state<string | null>(null);
	let confirmDelete = $state(false);
	let etag = '';
	let form = $state({ title: '', description: '', status: 'active', wip: '', docs_folder: '' });

	const cards = $derived(columns.flatMap((c) => c.cards));
	const titleOf = (taskId: string | null | undefined) =>
		cards.find((c) => c.task_id === taskId)?.title;
	const parents = $derived(cards.filter((c) => c.kind !== 'subtask' && c.column !== 'done'));
	const labels: Record<string, string> = $derived({
		backlog: t('Backlog'),
		todo: t('To do'),
		doing: t('Doing'),
		done: t('Done')
	});

	async function load() {
		error = '';
		try {
			const [b, d, p] = await Promise.all([
				api<{ project: Project; columns: Column[] }>(`/v1/projects/${id}/board`),
				api<{ items: Doc[] }>(`/v1/projects/${id}/docs`),
				apiResponse<Project>(`/v1/projects/${id}`)
			]);
			project = b.project;
			columns = b.columns;
			docs = d.items;
			etag = p.etag;
			form = {
				title: p.data.title,
				description: p.data.description,
				status: p.data.status,
				wip: p.data.wip_limit?.toString() ?? '',
				docs_folder: p.data.docs_folder
			};
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void id;
		project = null;
		load();
	});

	/** Moves a card; over the WIP limit, asks before going ahead. */
	async function move(card: Card, body: { column?: string; before?: string }, focus = true) {
		const send = (over: boolean) =>
			api(`/v1/projects/${id}/cards/${card.task_id}`, {
				method: 'PUT',
				body: { ...body, ...(over ? { over_limit: true } : {}) }
			});
		try {
			await send(false);
		} catch (err) {
			if (!(err instanceof ApiError && err.status === 409)) {
				toast(message(err), 'error');
				return;
			}
			if (
				!confirm(
					`${err.message.replace(' Send over_limit to move it anyway.', '')}\n\nMove “${card.title}” anyway?`
				)
			) {
				announce = t('{title} stayed where it was.', { title: card.title });
				return;
			}
			await send(true);
		}
		await load();
		const col = columns.find((c) => c.cards.some((x) => x.task_id === card.task_id));
		announce = `${card.title} is in ${col?.label ?? 'the board'}${body.before ? ', reordered' : ''}.`;
		if (focus) {
			await tick();
			document.getElementById(`card-${card.task_id}`)?.focus();
		}
	}

	function onkeydown(e: KeyboardEvent, card: Card, col: Column) {
		if (!e.shiftKey) return;
		if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
			const to = neighbour(card.column, e.key === 'ArrowLeft' ? -1 : 1);
			if (!to) return;
			e.preventDefault();
			move(card, { column: to });
		} else if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
			const before = reorder(col.cards, card.task_id, e.key === 'ArrowUp' ? -1 : 1);
			if (before === null) return;
			e.preventDefault();
			move(card, before ? { before } : { column: col.id });
		}
	}

	function drop(e: DragEvent, col: Column, before?: string) {
		e.preventDefault();
		const card = cards.find((c) => c.task_id === dragging);
		dragging = null;
		if (card && before !== card.task_id)
			move(card, { column: col.id, ...(before ? { before } : {}) }, false);
	}

	async function add(e: SubmitEvent, col: Column) {
		e.preventDefault();
		const title = (adding[col.id] ?? '').trim();
		if (!title) return;
		try {
			await api(`/v1/projects/${id}/cards`, {
				method: 'POST',
				body: {
					title,
					column: col.id,
					kind: addKind,
					...(addParent ? { parent_id: addParent } : {})
				}
			});
			adding[col.id] = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function newDoc() {
		const title = prompt(t('Title of the new doc'));
		if (!title?.trim() || !project) return;
		try {
			const slug =
				title
					.toLowerCase()
					.replace(/[^\p{L}\p{N}]+/gu, '-')
					.replace(/^-|-$/g, '') || 'doc';
			const n = await api<Note>('/v1/notes', {
				method: 'POST',
				body: { path: `${project.docs_folder}/${slug}.md`, content: `# ${title.trim()}\n\n` }
			});
			await refreshNotes();
			await goto(`/notes/${n.id}?edit=1`);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		try {
			const res = await apiResponse<Project>(`/v1/projects/${id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: {
					title: form.title,
					description: form.description,
					status: form.status,
					docs_folder: form.docs_folder,
					wip_limit: form.wip === '' ? null : Number(form.wip)
				}
			});
			etag = res.etag;
			toast(t('Saved.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove() {
		try {
			await api(`/v1/projects/${id}`, { method: 'DELETE' });
			const gone = id;
			deleted(t('Project deleted; its tasks are still in your task list.'), gone, () =>
				goto(`/projects/${gone}`)
			);
			await goto('/projects');
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>{project?.title ?? t('Project')} · Axon</title></svelte:head>

<p class="crumbs"><a href="/projects">{t('← Projects')}</a></p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !project}
	<Status kind="loading" message={t('Loading board…')} />
{:else}
	<h1>{project.title}</h1>
	{#if project.description}<p class="muted">{project.description}</p>{/if}
	<p class="muted small" id="board-help">
		{t(
			'Drag cards, or focus a card and press Shift + arrow keys: ← → to change column, ↑ ↓ to reorder.'
		)}
	</p>
	<p class="visually-hidden" role="status" aria-live="polite">{announce}</p>

	<div class="board" aria-describedby="board-help">
		{#each columns as col (col.id)}
			{@const w = wip(col)}
			<section
				class="column glass"
				class:over={w?.full}
				aria-labelledby="col-{col.id}"
				ondragover={(e) => e.preventDefault()}
				ondrop={(e) => drop(e, col)}
			>
				<h2 id="col-{col.id}">
					{col.label}
					{#if w}<span class="wip" class:full={w.full} title={t('WIP limit')}>{w.label}</span
						>{:else}<span class="count">{col.cards.length}</span>{/if}
				</h2>
				<ul>
					{#each col.cards as card (card.task_id)}
						<li
							class="card {card.kind}"
							draggable="true"
							ondragstart={() => (dragging = card.task_id)}
							ondragover={(e) => e.preventDefault()}
							ondrop={(e) => {
								e.stopPropagation();
								drop(e, col, card.task_id);
							}}
						>
							<a
								id="card-{card.task_id}"
								href="/tasks/{card.task_id}"
								onkeydown={(e) => onkeydown(e, card, col)}
								aria-describedby="board-help">{card.title}</a
							>
							<div class="meta">
								{#if card.kind !== 'issue'}<span class="chip"
										>{card.kind === 'epic' ? t('Epic') : t('Sub-task')}</span
									>{/if}
								{#if card.parent_id && titleOf(card.parent_id)}<span class="muted"
										>{t('in {value}', { value: titleOf(card.parent_id) })}</span
									>{/if}
								{#if card.children}<span class="muted"
										>{t('{children_done}/{children} done', {
											children_done: card.children_done,
											children: card.children
										})}</span
									>{/if}
								{#if card.due_on}<span class="muted"
										>{t('due {due_on}', { due_on: card.due_on })}</span
									>{/if}
							</div>
							<label class="visually-hidden" for="move-{card.task_id}"
								>{t('Move {title} to', { title: card.title })}</label
							>
							<select
								id="move-{card.task_id}"
								value={card.column}
								onchange={(e) => move(card, { column: e.currentTarget.value })}
							>
								{#each Object.entries(labels) as [cid, label] (cid)}<option value={cid}
										>{label}</option
									>{/each}
							</select>
						</li>
					{/each}
				</ul>
				{#if col.id !== 'done'}
					<form onsubmit={(e) => add(e, col)}>
						<label for="add-{col.id}" class="visually-hidden"
							>{t('Add a card to {label}', { label: col.label })}</label
						>
						<input
							id="add-{col.id}"
							placeholder={t('Add a card…')}
							bind:value={adding[col.id]}
							maxlength="500"
							autocomplete="off"
						/>
					</form>
				{/if}
			</section>
		{/each}
	</div>
	<div class="add-options">
		<label for="add-kind">{t('New cards are')}</label>
		<select id="add-kind" bind:value={addKind}>
			<option value="issue">{t('issues')}</option>
			<option value="epic">{t('epics')}</option>
			<option value="subtask">{t('sub-tasks')}</option>
		</select>
		{#if addKind !== 'epic'}
			<label for="add-parent">{t('in')}</label>
			<select id="add-parent" bind:value={addParent}>
				<option value=""
					>{addKind === 'subtask' ? t('Choose an issue or epic…') : t('no epic')}</option
				>
				{#each parents.filter( (p) => (addKind === 'issue' ? p.kind === 'epic' : true) ) as p (p.task_id)}
					<option value={p.task_id}>{p.title}</option>
				{/each}
			</select>
		{/if}
	</div>

	<section class="panel" aria-labelledby="docs-title">
		<div class="row">
			<h2 id="docs-title">{t('Docs')}</h2>
			<Button variant="secondary" onclick={newDoc}>{t('New doc')}</Button>
		</div>
		{#if docs.length === 0}
			<p class="muted">
				{t('Notes in')} <code>{project.docs_folder}/</code>
				{t('appear here: specs, decisions, meeting notes.')}
			</p>
		{:else}
			<ul class="rows">
				{#each docs as d (d.id)}<li>
						<a href="/notes/{d.id}">{d.title}</a> <span class="muted small">{d.path}</span>
					</li>{/each}
			</ul>
		{/if}
	</section>

	<Connections id={project.id} type="project" />

	<section class="panel" aria-labelledby="settings-title">
		<h2 id="settings-title">{t('Settings')}</h2>
		<form onsubmit={save}>
			<TextField
				id="p-title"
				label={t('Project')}
				bind:value={form.title}
				maxlength={500}
				required
			/>
			<div class="field">
				<label for="p-desc">{t('Outcome')}</label>
				<textarea id="p-desc" rows="2" maxlength="20000" bind:value={form.description}></textarea>
			</div>
			<div class="grid">
				<TextField
					id="p-wip"
					label={t('WIP limit for Doing')}
					hint={t('Empty for no limit.')}
					type="number"
					min="1"
					max="50"
					bind:value={form.wip}
				/>
				<TextField id="p-docs" label={t('Docs folder')} bind:value={form.docs_folder} required />
				<div class="field">
					<label for="p-status">{t('Status')}</label>
					<select id="p-status" bind:value={form.status}>
						<option value="active">{t('Active')}</option>
						<option value="on_hold">{t('On hold')}</option>
						<option value="done">{t('Done')}</option>
						<option value="archived">{t('Archived')}</option>
					</select>
				</div>
			</div>
			<div class="row">
				<Button type="submit">{t('Save')}</Button>
				<span class="grow"></span>
				<Button variant="danger" onclick={() => (confirmDelete = true)}
					>{t('Delete project')}</Button
				>
			</div>
		</form>
	</section>
{/if}

<Dialog bind:open={confirmDelete} title={t('Delete this project?')}>
	<p>
		{t('The board is removed. Its tasks stay in your task list, and its docs stay in your notes.')}
	</p>
	<div class="row end">
		<Button variant="secondary" onclick={() => (confirmDelete = false)}>{t('Cancel')}</Button>
		<Button variant="danger" onclick={remove}>{t('Delete')}</Button>
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
	h1 {
		overflow-wrap: anywhere;
	}
	.small {
		font-size: var(--text-sm);
	}
	.board {
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: minmax(15rem, 1fr);
		gap: var(--space-3);
		overflow-x: auto;
		padding-bottom: var(--space-3);
		margin: var(--space-4) 0;
	}
	.column {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-3);
		min-height: 12rem;
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}
	.column.over {
		border-color: var(--warn);
	}
	.column h2 {
		font-size: var(--text-base);
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin: 0 0 var(--space-1);
	}
	.count,
	.wip {
		font-size: var(--text-sm);
		font-weight: 500;
		color: var(--muted);
	}
	.wip {
		padding: 0 var(--space-2);
		border-radius: 999px;
		background: var(--accent-soft);
	}
	.wip.full {
		color: var(--warn);
		background: rgb(251 191 36 / 0.18);
	}
	ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: var(--space-2);
	}
	.card {
		background: var(--surface-solid);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-2) var(--space-3);
		display: grid;
		gap: var(--space-1);
		cursor: grab;
	}
	.card.epic {
		border-left: 4px solid #8b5cf6;
	}
	.card.subtask {
		margin-left: var(--space-3);
	}
	.card:focus-within {
		border-color: var(--accent);
	}
	.card a {
		color: var(--fg);
		text-decoration: none;
		font-weight: 550;
		overflow-wrap: anywhere;
	}
	.meta {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		font-size: var(--text-sm);
	}
	.chip {
		border-radius: 999px;
		padding: 0 var(--space-2);
		background: var(--accent-soft);
	}
	.card select {
		font: inherit;
		font-size: var(--text-sm);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
		color: var(--fg);
		padding: 0.1rem var(--space-1);
		justify-self: start;
	}
	.column form {
		margin-top: auto;
	}
	.column input,
	.add-options select,
	textarea,
	.field select {
		width: 100%;
		font: inherit;
		font-size: var(--text-sm);
		min-height: 2.25rem;
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.add-options {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
		font-size: var(--text-sm);
		margin-bottom: var(--space-6);
	}
	.add-options select {
		width: auto;
	}
	.panel {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	.panel h2 {
		font-size: var(--text-lg);
		margin: 0;
	}
	.row {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		flex-wrap: wrap;
		justify-content: space-between;
		margin-bottom: var(--space-3);
	}
	.end {
		justify-content: flex-end;
	}
	.grow {
		flex: 1;
	}
	.rows li {
		display: flex;
		gap: var(--space-3);
		flex-wrap: wrap;
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
	}
	.field label {
		font-weight: 600;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		column-gap: var(--space-4);
	}
	@media (max-width: 40rem) {
		.board {
			grid-auto-flow: row;
			grid-auto-columns: auto;
		}
	}
</style>
