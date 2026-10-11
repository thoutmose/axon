<script lang="ts">
	import { api, message } from './api';
	import { refreshNotes } from './noteIndex.svelte';
	import { templates } from './noteTemplates';
	import { capture } from './tasks';
	import Button from './ui/Button.svelte';
	import Dialog from './ui/Dialog.svelte';
	import TextField from './ui/TextField.svelte';
	import { toast } from './ui/toast.svelte';

	let { open = $bindable(false), oncaptured }: { open: boolean; oncaptured?: () => void } =
		$props();
	let title = $state('');
	let busy = $state(false);

	/** Saves the thought as a fleeting note in the notes inbox instead. */
	async function asNote() {
		if (!title.trim()) return;
		busy = true;
		try {
			const t = templates.find((x) => x.id === 'fleeting')!;
			const now = new Date();
			await api('/v1/notes', {
				method: 'POST',
				body: { path: t.path(title, now), content: t.content(title, now) }
			});
			toast('Saved as a note in your inbox.');
			title = '';
			open = false;
			refreshNotes().catch(() => {});
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			await capture(title);
			toast('Captured to your inbox.');
			title = '';
			open = false;
			oncaptured?.();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
</script>

<Dialog bind:open title="Capture">
	<form onsubmit={submit}>
		<!-- svelte-ignore a11y_autofocus -->
		<TextField
			id="capture-title"
			label="What's on your mind?"
			hint="A task or a note in your inbox; sort it out later."
			bind:value={title}
			maxlength={500}
			autocomplete="off"
			autofocus
			required
		/>
		<div class="actions">
			<Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
			<Button variant="secondary" onclick={asNote} {busy}>Save as note</Button>
			<Button type="submit" {busy}>Capture task</Button>
		</div>
	</form>
</Dialog>

<style>
	.actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-3);
	}
</style>
