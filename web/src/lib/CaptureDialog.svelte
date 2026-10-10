<script lang="ts">
	import { message } from './api';
	import { capture } from './tasks';
	import Button from './ui/Button.svelte';
	import Dialog from './ui/Dialog.svelte';
	import TextField from './ui/TextField.svelte';
	import { toast } from './ui/toast.svelte';

	let { open = $bindable(false), oncaptured }: { open: boolean; oncaptured?: () => void } =
		$props();
	let title = $state('');
	let busy = $state(false);

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
			hint="It goes to your inbox; sort it out later."
			bind:value={title}
			maxlength={500}
			autocomplete="off"
			autofocus
			required
		/>
		<div class="actions">
			<Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
			<Button type="submit" {busy}>Capture</Button>
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
