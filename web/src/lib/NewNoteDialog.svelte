<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';
	import { api, message } from './api';
	import { createFromTemplate } from './newNote';
	import type { Note } from './notes';
	import {
		builtInFiles,
		customTemplate,
		frontKeys,
		isTemplatePath,
		itemTemplates,
		sections,
		templates as builtIn,
		templateName,
		templatesFolder,
		vaultTemplate,
		type Template
	} from './noteTemplates';
	import Button from './ui/Button.svelte';
	import Dialog from './ui/Dialog.svelte';
	import TextField from './ui/TextField.svelte';
	import { toast } from './ui/toast.svelte';

	let { open = $bindable(false), kind = $bindable('permanent') }: { open: boolean; kind?: string } =
		$props();
	let title = $state('');
	let folder = $state(t('Axon'));
	let busy = $state(false);
	let mine = $state<Template[]>([]);
	// Template notes in the vault (Templates/ and the like), loaded on open.
	let vault = $state<Note[]>([]);
	// A vault copy of a built-in template replaces it.
	const templates = $derived([
		...mine,
		// Matched by name, or by file name as in the SecondBrain vault (Templates/goal.md).
		...builtIn.filter(
			(t) =>
				!vault.some((n) =>
					[t.label.toLowerCase(), t.id].includes(templateName(n.path).toLowerCase())
				)
		),
		...itemTemplates(folder)
	]);
	const groups: { id: string; label: string }[] = $derived([
		{ id: 'zettelkasten', label: t('Zettelkasten') },
		{ id: 'para', label: t('PARA notes') },
		{ id: 'periodic', label: t('Daily, weekly, and monthly notes') },
		{ id: 'plans', label: t('Roadmaps, goals, and habits') },
		{ id: 'reviews', label: t('Retrieval tests') },
		{ id: 'items', label: t('Axon items (kept in step with tasks, goals, habits…)') },
		{ id: 'other', label: t('Other') }
	]);
	const groupOf = (t: Template) => t.group ?? 'other';
	// The selected template's front matter and sections, shown before creating.
	let preview = $state<{ keys: string[]; sections: string[] } | null>(null);
	$effect(() => {
		const t = templates.find((x) => x.id === kind);
		const n = vault.find((x) => `vault:${x.id}` === kind);
		if (t) {
			const c = t.content('Title', new Date());
			preview = { keys: frontKeys(c), sections: sections(c) };
		} else if (n) {
			api<Note>(`/v1/notes/${n.id}`)
				.then((full) => {
					const c = full.content ?? '';
					preview = { keys: frontKeys(c), sections: sections(c) };
				})
				.catch(() => (preview = null));
		} else preview = null;
	});

	onMount(() => {
		api<{ templates: { note: string }; vault?: { folder: string } }>('/v1/me/settings')
			.then((s) => {
				if (s.templates.note.trim()) mine = [customTemplate(s.templates.note)];
				if (s.vault?.folder) folder = s.vault.folder;
			})
			.catch(() => {});
	});

	async function loadVault() {
		const found: Note[] = [];
		let after = '';
		for (let page = 0; page < 10; page++) {
			const res = await api<{ items: Note[]; next_cursor: string | null }>(
				`/v1/notes?para=other&limit=100${after ? `&after=${encodeURIComponent(after)}` : ''}`
			);
			found.push(...res.items.filter((n) => isTemplatePath(n.path)));
			if (!res.next_cursor) break;
			after = res.next_cursor;
		}
		vault = found;
	}

	$effect(() => {
		if (open) loadVault().catch(() => {});
	});

	async function installBuiltIns() {
		busy = true;
		try {
			for (const f of builtInFiles(new Date())) {
				await api('/v1/notes', { method: 'POST', body: f }).catch(() => {}); // already there
			}
			await loadVault();
			toast(
				t('The templates are in {templatesFolder}/: edit them like any note.', { templatesFolder })
			);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function create(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			let t = templates.find((x) => x.id === kind);
			const n = vault.find((x) => `vault:${x.id}` === kind);
			if (n) t = vaultTemplate(n, (await api<Note>(`/v1/notes/${n.id}`)).content ?? '', folder);
			if (!t) return;
			await createFromTemplate(t, title.trim());
			open = false;
			title = '';
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
</script>

<Dialog bind:open title={t('New note')}>
	<form onsubmit={create}>
		<TextField id="new-note-title" label={t('Title')} bind:value={title} maxlength={200} required />
		{#if vault.length}
			<fieldset>
				<legend
					>{t('Your templates')}
					<span class="muted small"
						>{t('({templatesFolder}/ in your vault)', { templatesFolder })}</span
					></legend
				>
				{#each vault as n (n.id)}
					<label class="choice">
						<input type="radio" name="template" value="vault:{n.id}" bind:group={kind} />
						<span
							><strong>{templateName(n.path)}</strong><span class="muted small">{n.path}</span
							></span
						>
					</label>
				{/each}
			</fieldset>
		{/if}
		{#each groups as g (g.id)}
			{@const list = templates.filter((t) => groupOf(t) === g.id)}
			{#if list.length}
				<fieldset>
					<legend>{g.label}</legend>
					{#each list as tmpl (tmpl.id)}
						<label class="choice">
							<input type="radio" name="template" value={tmpl.id} bind:group={kind} />
							<span
								><strong>{t(tmpl.label)}</strong><span class="muted small">{t(tmpl.hint)}</span
								></span
							>
						</label>
					{/each}
				</fieldset>
			{/if}
		{/each}
		{#if preview}
			<div class="preview" aria-live="polite">
				{#if preview.sections.length}
					<p class="small">
						<strong>{t('Sections:')}</strong>
						{preview.sections.join(' · ')}
					</p>
				{/if}
				{#if preview.keys.length}
					<p class="small muted">
						{t('Front matter: {value}', { value: preview.keys.join(', ') })}
					</p>
				{/if}
			</div>
		{/if}
		{#if !vault.length}
			<p class="muted small">
				{t('Templates are notes in')} <code>{templatesFolder}/</code>{t(
					'. Put your own there (or sync them from your vault), or'
				)}
				<button type="button" class="link" onclick={installBuiltIns} disabled={busy}
					>{t('copy these built-in ones there')}</button
				>
				{t('to edit them.')}
			</p>
		{/if}
		<div class="actions">
			<Button variant="secondary" onclick={() => (open = false)}>{t('Cancel')}</Button>
			<Button type="submit" {busy}>{t('Create')}</Button>
		</div>
	</form>
</Dialog>

<style>
	.preview {
		border-left: 3px solid var(--accent);
		padding-left: var(--space-3);
		margin: var(--space-2) 0;
	}
	fieldset {
		border: none;
		padding: 0;
		margin: 0 0 var(--space-4);
		display: grid;
		gap: var(--space-1);
		max-height: 50vh;
		overflow-y: auto;
	}
	legend {
		font-weight: 600;
		margin-bottom: var(--space-2);
	}
	.choice {
		display: flex;
		gap: var(--space-3);
		align-items: flex-start;
		padding: var(--space-2);
		border-radius: var(--radius);
		cursor: pointer;
	}
	.choice:hover,
	.choice:has(input:checked) {
		background: var(--accent-soft);
	}
	.choice input {
		margin-top: 0.3rem;
		accent-color: var(--accent);
	}
	.choice span {
		display: grid;
	}
	.small {
		font-size: var(--text-sm);
	}
	.link {
		font: inherit;
		background: none;
		border: none;
		padding: 0;
		color: var(--accent);
		text-decoration: underline;
		cursor: pointer;
	}
	.actions {
		display: flex;
		gap: var(--space-2);
		justify-content: flex-end;
	}
</style>
