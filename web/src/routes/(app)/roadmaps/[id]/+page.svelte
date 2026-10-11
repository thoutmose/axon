<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import Connections from '$lib/Connections.svelte';
	import SkillGraph from '$lib/SkillGraph.svelte';
	import { bySection, type Skill } from '$lib/skillGraph';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Roadmap = Schemas['Roadmap'];
	const id = $derived(page.params.id ?? '');
	const view = $derived(page.url.searchParams.get('view') ?? 'list');
	const selectedId = $derived(page.url.searchParams.get('skill'));
	let roadmap = $state<Roadmap | null>(null);
	let error = $state('');
	let form = $state({ title: '', section: '', url: '', kind: 'hard', requires: [] as string[] });

	// New skills start from the resource defaults in Preferences.
	api<{ templates: { resource: { section: string; kind: string } } }>('/v1/me/settings')
		.then((s) => {
			form.section = form.section || s.templates.resource.section;
			form.kind = s.templates.resource.kind;
		})
		.catch(() => {});
	let confirmDelete = $state(false);
	let editing = $state(false);
	let meta = $state({ title: '', description: '' });
	let skillEdit = $state<{
		id: string;
		title: string;
		section: string;
		url: string;
		description: string;
		kind: string;
	} | null>(null);
	let skillOpen = $state(false);

	const skills = $derived(roadmap?.skills ?? []);
	const edges = $derived((roadmap?.edges ?? []).map((e) => ({ from: e.from, to: e.to })));
	const titleOf = (sid: string) => skills.find((s) => s.id === sid)?.title ?? '';
	const needs = (sid: string) => edges.filter((e) => e.from === sid).map((e) => titleOf(e.to));
	const selected = $derived(skills.find((s) => s.id === selectedId) ?? null);
	const sections = $derived([...new Set(skills.map((s) => s.section).filter(Boolean))]);
	// Imports from before skills were competencies list one book or course per node.
	const ungrouped = $derived(
		(roadmap?.source === 'library' || roadmap?.source === 'ossu') &&
			skills.length > 0 &&
			skills.every((s) => s.resources.length === 0)
	);
	let regrouping = $state(false);

	async function regroup() {
		regrouping = true;
		try {
			roadmap = await api<Roadmap>(`/v1/roadmaps/${id}/regroup`, { method: 'POST' });
			toast('Grouped into skills. Your progress and connections were kept.');
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			regrouping = false;
		}
	}
	const statusNames: Record<string, string> = {
		todo: 'To learn',
		learning: 'Learning',
		done: 'Done',
		skipped: 'Skipped'
	};

	async function load() {
		error = '';
		try {
			roadmap = await api<Roadmap>(`/v1/roadmaps/${id}`);
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void id;
		roadmap = null;
		load();
	});

	function go(params: Record<string, string | null>) {
		const u = new URL(page.url);
		for (const [k, v] of Object.entries(params)) {
			if (v === null) u.searchParams.delete(k);
			else u.searchParams.set(k, v);
		}
		goto(u, { keepFocus: true, noScroll: true, replaceState: true });
	}

	async function setStatus(s: Skill, status: string) {
		try {
			await api(`/v1/skills/${s.id}`, { method: 'PATCH', body: { status } });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function add(e: SubmitEvent) {
		e.preventDefault();
		try {
			await api(`/v1/roadmaps/${id}/skills`, {
				method: 'POST',
				body: {
					title: form.title,
					section: form.section,
					...(form.url ? { url: form.url } : {}),
					kind: form.kind,
					requires: form.requires
				}
			});
			form = { title: '', section: form.section, url: '', kind: form.kind, requires: [] };
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function openEdit() {
		if (!roadmap) return;
		meta = { title: roadmap.title, description: roadmap.description };
		editing = true;
	}

	async function saveMeta(e: SubmitEvent) {
		e.preventDefault();
		try {
			await api(`/v1/roadmaps/${id}`, { method: 'PATCH', body: meta });
			editing = false;
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function editSkill(s: Skill) {
		skillEdit = {
			id: s.id,
			title: s.title,
			section: s.section,
			url: s.url ?? '',
			description: s.description,
			kind: s.kind
		};
		skillOpen = true;
	}

	async function saveSkill(e: SubmitEvent) {
		e.preventDefault();
		if (!skillEdit) return;
		const { id: sid, ...body } = skillEdit;
		try {
			await api(`/v1/skills/${sid}`, { method: 'PATCH', body });
			skillOpen = false;
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function removeSkill(s: Skill) {
		try {
			await api(`/v1/skills/${s.id}`, { method: 'DELETE' });
			go({ skill: null });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove() {
		try {
			await api(`/v1/roadmaps/${id}`, { method: 'DELETE' });
			await goto('/roadmaps');
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>{roadmap?.title ?? 'Roadmap'} · Axon</title></svelte:head>

<p class="crumbs"><a href="/roadmaps">← Roadmaps</a></p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !roadmap}
	<Status kind="loading" message="Loading roadmap…" />
{:else}
	<div class="head">
		<div>
			<h1>{roadmap.title}</h1>
			<p class="muted">
				{roadmap.done} of {roadmap.total} skills done{roadmap.description
					? ` · ${roadmap.description}`
					: ''}
			</p>
			{#if roadmap.attribution}
				<p class="attribution small">
					{roadmap.attribution}
					{#if roadmap.license_url}<a
							href={roadmap.license_url}
							rel="noopener noreferrer"
							target="_blank">Licence</a
						>{/if}
				</p>
			{/if}
		</div>
		<div class="row">
			<Button variant="secondary" onclick={openEdit}>Edit</Button>
			<Button variant="danger" onclick={() => (confirmDelete = true)}>Delete</Button>
		</div>
	</div>

	{#if ungrouped}
		<div class="upgrade card glass" role="note">
			<p>
				This roadmap was imported when each book or course was its own node. Group them into the
				skills they teach, with the resources listed under each skill. Progress and connections are
				kept.
			</p>
			<Button variant="secondary" busy={regrouping} onclick={regroup}>Group into skills</Button>
		</div>
	{/if}

	<nav class="tabs" aria-label="View">
		<a href="?view=list" aria-current={view === 'list' ? 'page' : undefined}>List</a>
		<a href="?view=graph" aria-current={view === 'graph' ? 'page' : undefined}>Skill graph</a>
	</nav>

	<div class="layout" class:with-detail={!!selected}>
		<div class="main">
			{#if skills.length === 0}
				<Status kind="empty" message="No skills yet. Add the first one below." />
			{:else if view === 'graph'}
				<SkillGraph {skills} {edges} selected={selectedId} onselect={(sid) => go({ skill: sid })} />
			{:else}
				{#each bySection(skills) as [section, items] (section)}
					<section class="card glass" aria-labelledby="sec-{section}">
						<h2 id="sec-{section}">
							{section}
							<span class="muted small"
								>{items.filter((s) => s.status === 'done').length}/{items.length}</span
							>
						</h2>
						<ul class="rows">
							{#each items as s (s.id)}
								<li class={s.status}>
									<div class="grow">
										<a href="?view=list&skill={s.id}">{s.title}</a>
										{#if needs(s.id).length}<span class="muted small"
												>needs {needs(s.id).join(', ')}</span
											>{/if}
									</div>
									<label class="visually-hidden" for="status-{s.id}">Status of {s.title}</label>
									<select
										id="status-{s.id}"
										value={s.status}
										onchange={(e) => setStatus(s, e.currentTarget.value)}
									>
										{#each Object.entries(statusNames) as [k, label] (k)}<option value={k}
												>{label}</option
											>{/each}
									</select>
								</li>
							{/each}
						</ul>
					</section>
				{/each}
			{/if}

			<section class="card glass" aria-labelledby="add-skill-title">
				<h2 id="add-skill-title">Add a skill</h2>
				<form onsubmit={add}>
					<div class="grid">
						<TextField
							id="skill-title"
							label="Skill"
							bind:value={form.title}
							maxlength={500}
							required
						/>
						<TextField
							id="skill-section"
							label="Section"
							list="sections"
							bind:value={form.section}
							maxlength={200}
						/>
						<TextField
							id="skill-url"
							label="Link (optional)"
							type="url"
							placeholder="https://…"
							bind:value={form.url}
						/>
						<div class="field">
							<label for="skill-kind">Kind</label>
							<select id="skill-kind" bind:value={form.kind}>
								<option value="hard">Hard skill</option>
								<option value="soft">Soft skill</option>
							</select>
						</div>
					</div>
					<datalist id="sections"
						>{#each sections as s (s)}<option value={s}></option>{/each}</datalist
					>
					{#if skills.length}
						<div class="field">
							<label for="skill-requires">Needs first (optional)</label>
							<select id="skill-requires" multiple size="4" bind:value={form.requires}>
								{#each skills as s (s.id)}<option value={s.id}>{s.title}</option>{/each}
							</select>
						</div>
					{/if}
					<Button type="submit" variant="secondary">Add skill</Button>
				</form>
			</section>
		</div>

		{#if selected}
			<aside class="detail card glass" aria-labelledby="detail-title">
				<div class="row">
					<h2 id="detail-title">{selected.title}</h2>
					<Button variant="ghost" onclick={() => selected && editSkill(selected)}>Edit</Button>
					<Button variant="ghost" onclick={() => go({ skill: null })} aria-label="Close details"
						>✕</Button
					>
				</div>
				{#if selected.description}<p class="muted">{selected.description}</p>{/if}
				{#if selected.resources.length}
					<h3 class="small">Learn it with</h3>
					<ul class="resources">
						{#each selected.resources as r (r.url + r.title)}
							<li>
								<a href={r.url} rel="noopener noreferrer" target="_blank">{r.title} ↗</a>
								<span class="muted small"
									>{r.kind}{r.by ? ` · ${r.by}` : ''}{r.free ? ' · free' : ''}{r.note
										? ` · ${r.note}`
										: ''}</span
								>
							</li>
						{/each}
					</ul>
				{:else if selected.url}<p>
						<a href={selected.url} rel="noopener noreferrer" target="_blank"
							>Open the course or resource ↗</a
						>
					</p>{/if}
				{#if needs(selected.id).length}<p class="small">
						Needs first: {needs(selected.id).join(', ')}
					</p>{/if}
				<div class="statuses" role="group" aria-label="Status">
					{#each Object.entries(statusNames) as [k, label] (k)}
						<Button
							variant={selected.status === k ? 'primary' : 'secondary'}
							onclick={() => selected && setStatus(selected, k)}>{label}</Button
						>
					{/each}
				</div>
				<Connections id={selected.id} type="skill" />
				<Button variant="danger" onclick={() => selected && removeSkill(selected)}
					>Delete skill</Button
				>
			</aside>
		{/if}
	</div>

	<Connections id={roadmap.id} type="roadmap" />
{/if}

<Dialog bind:open={editing} title="Edit roadmap">
	<form onsubmit={saveMeta}>
		<TextField id="roadmap-title" label="Title" bind:value={meta.title} maxlength={500} required />
		<div class="field wide">
			<label for="roadmap-description">Description</label>
			<textarea id="roadmap-description" rows="4" maxlength="20000" bind:value={meta.description}
			></textarea>
		</div>
		<div class="row end">
			<Button variant="secondary" onclick={() => (editing = false)}>Cancel</Button>
			<Button type="submit">Save</Button>
		</div>
	</form>
</Dialog>

<Dialog bind:open={skillOpen} title="Edit skill">
	{#if skillEdit}
		<form onsubmit={saveSkill}>
			<TextField
				id="edit-skill-title"
				label="Skill"
				bind:value={skillEdit.title}
				maxlength={500}
				required
			/>
			<TextField
				id="edit-skill-section"
				label="Section"
				list="sections"
				bind:value={skillEdit.section}
				maxlength={200}
			/>
			<TextField
				id="edit-skill-url"
				label="Link"
				type="url"
				placeholder="https://…"
				bind:value={skillEdit.url}
			/>
			<div class="field">
				<label for="edit-skill-kind">Kind</label>
				<select id="edit-skill-kind" bind:value={skillEdit.kind}>
					<option value="hard">Hard skill</option>
					<option value="soft">Soft skill</option>
				</select>
			</div>
			<div class="field wide">
				<label for="edit-skill-description">Notes</label>
				<textarea
					id="edit-skill-description"
					rows="3"
					maxlength="5000"
					bind:value={skillEdit.description}
				></textarea>
			</div>
			<div class="row end">
				<Button variant="secondary" onclick={() => (skillOpen = false)}>Cancel</Button>
				<Button type="submit">Save</Button>
			</div>
		</form>
	{/if}
</Dialog>

<Dialog bind:open={confirmDelete} title="Delete this roadmap?">
	<p>Its skills and their links are removed. Goals and notes stay.</p>
	<div class="row end">
		<Button variant="secondary" onclick={() => (confirmDelete = false)}>Cancel</Button>
		<Button variant="danger" onclick={remove}>Delete</Button>
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
		align-items: flex-start;
		gap: var(--space-4);
		flex-wrap: wrap;
	}
	.small {
		font-size: var(--text-sm);
	}
	.attribution {
		color: var(--muted);
	}
	.tabs {
		display: flex;
		gap: var(--space-1);
		margin: var(--space-4) 0;
	}
	.tabs a {
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
	.layout {
		display: grid;
		gap: var(--space-4);
		align-items: start;
	}
	.layout.with-detail {
		grid-template-columns: minmax(0, 1fr) minmax(16rem, 22rem);
	}
	@media (max-width: 60rem) {
		.layout.with-detail {
			grid-template-columns: minmax(0, 1fr);
		}
	}
	.main {
		min-width: 0;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-4);
	}
	@media (max-width: 30rem) {
		.card {
			padding: var(--space-4);
		}
	}
	.card h2 {
		font-size: var(--text-lg);
	}
	.detail {
		position: sticky;
		top: 4.5rem;
	}
	.rows {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.rows li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
	}
	.rows li.done a {
		color: var(--ok);
	}
	.rows li.skipped a {
		color: var(--muted);
		text-decoration: line-through;
	}
	.grow {
		flex: 1;
		min-width: 10rem;
		display: grid;
	}
	.grow a {
		color: var(--fg);
		text-decoration: none;
		font-weight: 550;
		overflow-wrap: anywhere;
	}
	select {
		font: inherit;
		font-size: var(--text-sm);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
		color: var(--fg);
		padding: var(--space-1) var(--space-2);
		max-width: 100%;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		column-gap: var(--space-4);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
		max-width: 28rem;
	}
	.field label {
		font-weight: 600;
	}
	.field.wide {
		max-width: none;
	}
	textarea {
		font: inherit;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		width: 100%;
	}
	.row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
	}
	.end {
		justify-content: flex-end;
	}
	.statuses {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}
	.resources {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-4);
		display: grid;
		gap: var(--space-2);
	}
	.resources li {
		display: grid;
	}
	.upgrade {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-4);
		flex-wrap: wrap;
		margin-bottom: var(--space-4);
	}
	.upgrade p {
		margin: 0;
		max-width: 60rem;
	}
</style>
