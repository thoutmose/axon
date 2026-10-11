<script lang="ts">
	import { onMount } from 'svelte';
	import { api, message } from './api';
	import { createFromTemplate } from './newNote';
	import type { Note } from './notes';
	import {
		builtInFiles,
		customTemplate,
		isTemplatePath,
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

	let { open = $bindable(false) }: { open: boolean } = $props();
	let title = $state('');
	let kind = $state('permanent');
	let busy = $state(false);
	let mine = $state<Template[]>([]);
	// Template notes in the vault (Templates/ and the like), loaded on open.
	let vault = $state<Note[]>([]);
	// A vault copy of a built-in template replaces it.
	const templates = $derived([
		...mine,
		...builtIn.filter((t) => !vault.some((n) => templateName(n.path) === t.label))
	]);

	onMount(() => {
		api<{ templates: { note: string } }>('/v1/me/settings')
			.then((s) => {
				if (s.templates.note.trim()) mine = [customTemplate(s.templates.note)];
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
			toast(`The templates are in ${templatesFolder}/: edit them like any note.`);
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
			if (n) t = vaultTemplate(n, (await api<Note>(`/v1/notes/${n.id}`)).content ?? '');
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

<Dialog bind:open title="New note">
	<form onsubmit={create}>
		<TextField id="new-note-title" label="Title" bind:value={title} maxlength={200} required />
		{#if vault.length}
			<fieldset>
				<legend
					>Your templates <span class="muted small">({templatesFolder}/ in your vault)</span
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
		<fieldset>
			<legend>{vault.length ? 'Built in' : 'Kind'}</legend>
			{#each templates as t (t.id)}
				<label class="choice">
					<input type="radio" name="template" value={t.id} bind:group={kind} />
					<span><strong>{t.label}</strong><span class="muted small">{t.hint}</span></span>
				</label>
			{/each}
		</fieldset>
		{#if !vault.length}
			<p class="muted small">
				Templates are notes in <code>{templatesFolder}/</code>. Put your own there (or sync them
				from your vault), or
				<button type="button" class="link" onclick={installBuiltIns} disabled={busy}
					>copy these built-in ones there</button
				> to edit them.
			</p>
		{/if}
		<div class="actions">
			<Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
			<Button type="submit" {busy}>Create</Button>
		</div>
	</form>
</Dialog>

<style>
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
