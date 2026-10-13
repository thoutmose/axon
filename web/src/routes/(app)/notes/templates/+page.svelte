<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';
	import { api, message } from '$lib/api';
	import {
		goalTemplates,
		habitTemplates,
		milestonePlans,
		projectTemplates
	} from '$lib/entityTemplates';
	import type { Note } from '$lib/notes';
	import {
		builtInFiles,
		frontKeys,
		isTemplatePath,
		itemTemplates,
		sections,
		templateName,
		templates,
		templatesFolder,
		type Template
	} from '$lib/noteTemplates';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let folder = $state(t('Axon'));
	let vault = $state<(Note & { content?: string })[]>([]);
	let busy = $state(false);

	async function loadVault() {
		const found: Note[] = [];
		let after = '';
		for (let i = 0; i < 10; i++) {
			const res = await api<{ items: Note[]; next_cursor: string | null }>(
				`/v1/notes?para=other&limit=100${after ? `&after=${encodeURIComponent(after)}` : ''}`
			);
			found.push(...res.items.filter((n) => isTemplatePath(n.path)));
			if (!res.next_cursor) break;
			after = res.next_cursor;
		}
		vault = await Promise.all(found.map((n) => api<Note>(`/v1/notes/${n.id}`)));
	}

	onMount(() => {
		api<{ vault?: { folder: string } }>('/v1/me/settings')
			.then((s) => (folder = s.vault?.folder ?? t('Axon')))
			.catch(() => {});
		loadVault().catch((err) => toast(message(err), 'error'));
	});

	async function install() {
		busy = true;
		try {
			for (const f of builtInFiles(new Date())) {
				await api('/v1/notes', { method: 'POST', body: f }).catch(() => {});
			}
			await loadVault();
			toast(
				t('The templates are in {templatesFolder}/ in your vault: edit them like any note.', {
					templatesFolder
				})
			);
		} finally {
			busy = false;
		}
	}

	const groups = $derived([
		{
			id: 'zettelkasten',
			title: t('Zettelkasten'),
			text: t(
				'Capture, read, and think: fleeting notes for your inbox, literature notes in your own words, permanent notes that stand alone.'
			),
			list: templates.filter((t) => t.group === 'zettelkasten')
		},
		{
			id: 'para',
			title: t('PARA notes'),
			text: t('Notes filed by Projects, Areas, Resources, and Archives.'),
			list: templates.filter((t) => t.group === 'para')
		},
		{
			id: 'items',
			title: t('Axon items in your vault'),
			text: t(
				'Under {folder}/ these notes are real tasks, goals, milestones, projects, areas, resources, habits, and films, series, or books, kept in step both ways: edit the note or the item, the other follows. Health logs are written there too when you turn them on in Preferences.',
				{ folder }
			),
			list: itemTemplates(folder)
		},
		{
			id: 'other',
			title: t('Other'),
			text: t('Meetings and a blank note. Your daily note opens from Today in the sidebar.'),
			list: templates.filter((t) => t.group === 'other')
		}
	]);
	const show = (t: Template) => t.content('Title', new Date());
</script>

<svelte:head><title>{t('Templates · Notes · Axon')}</title></svelte:head>

<h1>{t('Templates')}</h1>
<p class="muted">
	{t('Every kind of note starts from a template with its own sections. Pick one when you press')}
	<strong>{t('New note')}</strong>{t(", or use one below. Your vault's own templates (in")}
	<code>{templatesFolder}/</code>{t(
		', for example from SecondBrain) come first and replace the built-in ones with the same name.'
	)}
</p>

<section class="card" aria-labelledby="vault-title">
	<h2 id="vault-title">{t('Your templates')}</h2>
	{#if vault.length}
		<ul class="grid">
			{#each vault as n (n.id)}
				<li class="tpl">
					<h3>{templateName(n.path)}</h3>
					<p class="small muted"><code>{n.path}</code> {t('· type {kind}', { kind: n.kind })}</p>
					{#if sections(n.content ?? '').length}
						<ul class="chips" aria-label={t('Sections')}>
							{#each sections(n.content ?? '') as s (s)}<li>{s}</li>{/each}
						</ul>
					{/if}
					<div class="row">
						<Button href="/notes?new=vault:{n.id}" variant="secondary">{t('Use')}</Button>
						<a href="/notes/{n.id}?edit=1" class="small">{t('Edit')}</a>
					</div>
				</li>
			{/each}
		</ul>
	{:else}
		<p class="muted">
			{t("No templates in your vault yet. Sync your vault's")} <code>{templatesFolder}/</code>
			{t('folder, or')}
			<Button variant="secondary" {busy} onclick={install}
				>{t('copy the built-in templates there')}</Button
			>
			{t('to edit them.')}
		</p>
	{/if}
</section>

{#each groups as g (g.id)}
	<section class="card" aria-labelledby="g-{g.id}">
		<h2 id="g-{g.id}">{g.title}</h2>
		<p class="small muted">{g.text}</p>
		<ul class="grid">
			{#each g.list as tmpl (tmpl.id)}
				<li class="tpl">
					<h3>{t(tmpl.label)}</h3>
					<p class="small">{t(tmpl.hint)}</p>
					{#if sections(show(tmpl)).length}
						<ul class="chips" aria-label={t('Sections of {label}', { label: t(tmpl.label) })}>
							{#each sections(show(tmpl)) as s (s)}<li>{s}</li>{/each}
						</ul>
					{/if}
					{#if frontKeys(show(tmpl)).length}
						<p class="small muted">
							{t('Front matter: {value}', { value: frontKeys(show(tmpl)).join(', ') })}
						</p>
					{/if}
					<details>
						<summary class="small">{t('Show the Markdown')}</summary>
						<pre>{show(tmpl)}</pre>
					</details>
					<Button
						href="/notes?new={tmpl.id}"
						variant="secondary"
						aria-label={t('Use the {label} template', { label: t(tmpl.label) })}>{t('Use')}</Button
					>
				</li>
			{/each}
		</ul>
	</section>
{/each}

<section class="card" aria-labelledby="pages-title">
	<h2 id="pages-title">{t('Templates on the other pages')}</h2>
	<ul class="small">
		<li>
			<a href="/goals">{t('Goals')}</a>{t(': {goals}; milestone plans: {plans}.', {
				goals: goalTemplates.map((g) => t(g.title)).join(', '),
				plans: milestonePlans.map((m) => t(m.title)).join(', ')
			})}
		</li>
		<li>
			<a href="/habits">{t('Habits')}</a>: {habitTemplates.map((h) => t(h.title)).join(', ')}.
		</li>
		<li>
			<a href="/projects">{t('Projects')}</a>: {projectTemplates.map((p) => t(p.title)).join(', ')}.
		</li>
		<li>
			<a href="/areas">{t('Areas')}</a>
			{t('and')} <a href="/resources">{t('Resources')}</a>{t(': starting points on each page.')}
		</li>
	</ul>
</section>

<style>
	.small {
		font-size: var(--text-sm);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.grid {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 17rem), 1fr));
		gap: var(--space-3);
	}
	.tpl {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-3);
		display: grid;
		gap: var(--space-2);
		align-content: start;
	}
	.tpl h3 {
		margin: 0;
		font-size: var(--text-base);
	}
	.chips {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
	}
	.chips li {
		background: var(--accent-soft);
		border-radius: 999px;
		padding: 0 var(--space-2);
		font-size: var(--text-sm);
	}
	pre {
		white-space: pre-wrap;
		font-size: var(--text-sm);
		background: var(--surface-solid, var(--surface));
		padding: var(--space-2);
		border-radius: var(--radius);
	}
	.row {
		display: flex;
		gap: var(--space-3);
		align-items: center;
	}
</style>
