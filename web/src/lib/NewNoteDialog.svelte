<script lang="ts">
	import { onMount } from 'svelte';
	import { api, message } from './api';
	import { createFromTemplate } from './newNote';
	import {
		customTemplate,
		fill,
		templates as builtIn,
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
	let templates = $state<Template[]>(builtIn);

	onMount(async () => {
		const extra: Template[] = [];
		try {
			const s = await api<{ templates: { note: string } }>('/v1/me/settings');
			if (s.templates.note.trim()) extra.push(customTemplate(s.templates.note));
		} catch {
			// Settings are optional here.
		}
		try {
			// Notes in the vault's Templates/ folder are templates too.
			const list = await api<{ items: { id: string; title: string; path: string }[] }>(
				'/v1/notes?folder=Templates/&limit=50'
			);
			for (const n of list.items) {
				extra.push(
					vaultTemplate(
						n.id,
						n.path.split('/').pop()!.replace(/\.md$/, ''),
						n.path,
						async () => (await api<{ content: string }>(`/v1/notes/${n.id}`)).content
					)
				);
			}
		} catch {
			// No templates folder: the built-in ones are enough.
		}
		templates = [...extra, ...builtIn];
	});

	async function create(e: SubmitEvent) {
		e.preventDefault();
		let t = templates.find((x) => x.id === kind);
		if (!t) return;
		busy = true;
		try {
			if ('load' in t && typeof t.load === 'function') {
				const body = await (t.load as () => Promise<string>)();
				t = { ...t, content: (title) => fill(body, title, new Date()) };
			}
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
		<fieldset>
			<legend>Kind</legend>
			{#each templates as t (t.id)}
				<label class="choice">
					<input type="radio" name="template" value={t.id} bind:group={kind} />
					<span><strong>{t.label}</strong><span class="muted small">{t.hint}</span></span>
				</label>
			{/each}
		</fieldset>
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
	.actions {
		display: flex;
		gap: var(--space-2);
		justify-content: flex-end;
	}
</style>
