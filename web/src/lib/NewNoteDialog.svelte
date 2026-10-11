<script lang="ts">
	import { message } from './api';
	import { createFromTemplate } from './newNote';
	import { templates } from './noteTemplates';
	import Button from './ui/Button.svelte';
	import Dialog from './ui/Dialog.svelte';
	import TextField from './ui/TextField.svelte';
	import { toast } from './ui/toast.svelte';

	let { open = $bindable(false) }: { open: boolean } = $props();
	let title = $state('');
	let kind = $state('permanent');
	let busy = $state(false);

	async function create(e: SubmitEvent) {
		e.preventDefault();
		const t = templates.find((x) => x.id === kind);
		if (!t) return;
		busy = true;
		try {
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
