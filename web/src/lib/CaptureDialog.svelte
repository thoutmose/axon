<script lang="ts">
	import { api, message } from './api';
	import { keep, unreachable } from './outbox.svelte';
	import { refreshNotes } from './noteIndex.svelte';
	import { templates } from './noteTemplates';
	import { action, describe, parse, type Parsed } from './quickAdd';
	import { capture } from './tasks';
	import Button from './ui/Button.svelte';
	import Dialog from './ui/Dialog.svelte';
	import TextField from './ui/TextField.svelte';
	import { toast } from './ui/toast.svelte';

	let { open = $bindable(false), oncaptured }: { open: boolean; oncaptured?: () => void } =
		$props();
	let title = $state('');
	let busy = $state(false);

	const parsed = $derived(parse(title));
	const preview = $derived(title.trim() ? describe(parsed) : []);

	/** A fleeting note in the inbox, with the tags typed as #tags. */
	function fleeting(text: string, tags: string[]) {
		const t = templates.find((x) => x.id === 'fleeting')!;
		const now = new Date();
		let content = t.content(text, now);
		if (tags.length)
			content = content.replace('type: fleeting\n', `type: fleeting\ntags: [${tags.join(', ')}]\n`);
		return { path: t.path(text, now), content };
	}

	/** Saves the thought as a fleeting note in the notes inbox instead. */
	async function asNote() {
		if (!title.trim()) return;
		await add({
			...parsed,
			kind: 'note',
			title: parsed.kind === 'note' ? parsed.title : title.trim()
		});
	}

	async function add(p: Parsed) {
		busy = true;
		const raw = title;
		const done = (msg: string) => {
			toast(msg);
			title = '';
			open = false;
		};
		try {
			switch (p.kind) {
				case 'task':
					await capture(p.title, {
						...(p.context ? { context: p.context } : {}),
						...(p.due ? { due_on: p.due } : {}),
						...(p.status ? { status: p.status } : {})
					});
					done(p.status ? 'Saved for someday.' : 'Captured to your inbox.');
					oncaptured?.();
					break;
				case 'note':
					await api('/v1/notes', { method: 'POST', body: fleeting(p.title, p.tags) });
					done('Saved as a note in your inbox.');
					refreshNotes().catch(() => {});
					break;
				case 'book':
				case 'film':
				case 'series':
					await api('/v1/media', {
						method: 'POST',
						body: { kind: p.kind, title: p.title, status: 'want' }
					});
					done(`Added to ${p.kind === 'book' ? 'books to read' : 'films and series to watch'}.`);
					break;
				case 'habit':
					await api('/v1/habits', { method: 'POST', body: { title: p.title } });
					done('Habit added. Give it a two-minute version on the Habits page.');
					break;
				case 'goal':
					await api('/v1/goals', {
						method: 'POST',
						body: { title: p.title, ...(p.due ? { target_date: p.due } : {}) }
					});
					done('Goal added.');
					break;
				case 'project':
					await api('/v1/projects', { method: 'POST', body: { title: p.title } });
					done('Project added.');
					break;
				case 'card':
					await api('/v1/flashcards', {
						method: 'POST',
						body: { front: p.title, back: p.back ?? '', deck: p.tags[0] ?? 'Default' }
					});
					done(`Card added to ${p.tags[0] ?? 'Default'}.`);
					break;
			}
		} catch (err) {
			if (unreachable(err)) {
				// Offline, it waits on the device: a note stays a note, anything else becomes a task to sort later.
				if (p.kind === 'note') keep({ kind: 'note', title: p.title, ...fleeting(p.title, p.tags) });
				else keep({ kind: 'task', title: p.kind === 'task' ? p.title : raw.trim() });
				done('No connection: kept on this device, sent when you are back online.');
			} else toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		if (!title.trim()) return;
		await add(parsed);
	}
</script>

<Dialog bind:open title="Capture">
	<form onsubmit={submit}>
		<!-- svelte-ignore a11y_autofocus -->
		<TextField
			id="capture-title"
			label="What's on your mind?"
			hint="A task by default: add @context and a day (tomorrow, friday). Or start with note:, book:, film:, habit:, goal:, project:, or card: question :: answer."
			bind:value={title}
			maxlength={500}
			autocomplete="off"
			autofocus
			required
		/>
		{#if preview.length}
			<p class="preview" aria-live="polite">
				{#each preview as chip, i (i)}<span class="chip">{chip}</span>{/each}
				<span class="what">{parsed.title}</span>
			</p>
		{/if}
		<div class="actions">
			<Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
			{#if parsed.kind === 'task'}
				<Button variant="secondary" onclick={asNote} {busy}>Save as note</Button>
			{/if}
			<Button type="submit" {busy}>{action(parsed.kind)}</Button>
		</div>
	</form>
</Dialog>

<style>
	.preview {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		align-items: center;
		font-size: var(--text-sm);
		margin: var(--space-2) 0 0;
	}
	.chip {
		padding: 0 var(--space-2);
		border-radius: 999px;
		background: var(--accent-soft);
	}
	.what {
		color: var(--muted);
		overflow-wrap: anywhere;
	}
	.actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-3);
	}
</style>
