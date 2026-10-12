<script lang="ts">
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { slug } from '$lib/noteTemplates';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import { projectTemplates, type ProjectTemplate } from '$lib/entityTemplates';
	import type { Project } from '$lib/projects';
	import TemplateChips from '$lib/TemplateChips.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	const tabs = [
		{ id: 'active', label: 'Active' },
		{ id: 'on_hold', label: 'On hold' },
		{ id: 'done', label: 'Done' },
		{ id: 'archived', label: 'Archived' }
	];
	const status = $derived(page.url.searchParams.get('status') ?? 'active');
	let projects = $state<Project[] | null>(null);
	let error = $state('');
	let title = $state('');

	$effect(() => {
		const s = status;
		projects = null;
		error = '';
		api<{ items: Project[] }>(`/v1/projects?status=${s}`)
			.then((r) => (projects = r.items))
			.catch((err) => (error = message(err)));
	});

	let busy = $state(false);

	function create(e: SubmitEvent) {
		e.preventDefault();
		start(title);
	}

	// A project from a template gets its description, WIP limit, and a
	// backlog of cards; the title is the one typed above, if any.
	async function start(name: string, t?: ProjectTemplate) {
		busy = true;
		try {
			// New projects start from the defaults in Preferences.
			const tpl = await api<{
				templates: {
					project: { description: string; wip_limit: number | null; docs_folder: string };
				};
			}>('/v1/me/settings')
				.then((s) => s.templates.project)
				.catch(() => null);
			const body: Record<string, unknown> = { title: name };
			if (tpl?.description) body.description = tpl.description;
			if (tpl?.wip_limit) body.wip_limit = tpl.wip_limit;
			if (tpl?.docs_folder) body.docs_folder = tpl.docs_folder.replaceAll('{slug}', slug(name));
			if (t) {
				body.description = t.description;
				if (t.wip_limit) body.wip_limit = t.wip_limit;
			}
			const p = await api<Project>('/v1/projects', { method: 'POST', body });
			for (const c of t?.cards ?? []) {
				const card = await api<{ task_id: string }>(`/v1/projects/${p.id}/cards`, {
					method: 'POST',
					body: { title: c.title, kind: c.kind, column: 'backlog' }
				});
				for (const child of c.children ?? []) {
					await api(`/v1/projects/${p.id}/cards`, {
						method: 'POST',
						body: { title: child, kind: 'issue', column: 'backlog', parent_id: card.task_id }
					});
				}
			}
			title = '';
			await goto(`/projects/${p.id}`);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Projects · Axon</title></svelte:head>

<h1>Projects</h1>
<AgentDock {...pageAgents.projects} />
<p class="muted hint">
	Outcomes with a finish line: a board of epics, issues, and sub-tasks, with their docs.
</p>

<nav aria-label="Project status" class="tabs">
	{#each tabs as t (t.id)}
		<a href="?status={t.id}" aria-current={status === t.id ? 'page' : undefined}>{t.label}</a>
	{/each}
</nav>

{#if status === 'active'}
	<form class="quick" onsubmit={create}>
		<label for="new-project" class="visually-hidden">New project</label>
		<input
			id="new-project"
			placeholder="New project, e.g. Launch the newsletter…"
			bind:value={title}
			maxlength="500"
			required
			autocomplete="off"
		/>
		<Button type="submit" variant="secondary">Create</Button>
	</form>
	<TemplateChips
		items={projectTemplates.map((t) => ({ title: t.title, hint: t.description }))}
		onpick={(i) => start(title.trim() || projectTemplates[i].title, projectTemplates[i])}
		{busy}
	/>
{/if}

{#if error}
	<Status kind="error" message={error} />
{:else if !projects}
	<Status kind="loading" message="Loading projects…" />
{:else if projects.length === 0}
	<Status
		kind="empty"
		message={status === 'active' ? 'No active projects. Start one above.' : 'Nothing here.'}
	/>
{:else}
	<ul class="grid">
		{#each projects as p (p.id)}
			<li class="card glass">
				<a href="/projects/{p.id}">{p.title}</a>
				{#if p.description}<p class="muted small">{p.description.slice(0, 140)}</p>{/if}
				<div
					class="bar"
					role="progressbar"
					aria-label="Progress of {p.title}"
					aria-valuemin="0"
					aria-valuemax={p.total || 1}
					aria-valuenow={p.done}
				>
					<span style:width="{p.total ? (p.done / p.total) * 100 : 0}%"></span>
				</div>
				<p class="muted small">
					{p.done} of {p.total} cards done{p.wip_limit ? ` · WIP limit ${p.wip_limit}` : ''}
				</p>
			</li>
		{/each}
	</ul>
{/if}

<style>
	.hint,
	.small {
		font-size: var(--text-sm);
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
	.grid {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 16rem), 1fr));
		gap: var(--space-3);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	.card a {
		font-weight: 650;
		color: var(--fg);
		text-decoration: none;
		overflow-wrap: anywhere;
	}
	.card p {
		margin: var(--space-2) 0 0;
	}
	.bar {
		height: 0.4rem;
		border-radius: 999px;
		background: var(--accent-soft);
		overflow: hidden;
		margin-top: var(--space-3);
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--accent);
	}
</style>
