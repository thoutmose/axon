<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Trash (EP-66): deletions of the last 30 days, put back with what was
	// deleted along with them.
	type Item = {
		id: string;
		entity_id: string;
		type: string;
		title: string;
		with: number;
		deleted_at: string;
		expires_at: string;
	};
	let items = $state<Item[] | null>(null);
	let error = $state('');
	const pages: Record<string, string> = {
		note: '/notes/',
		task: '/tasks?view=all#',
		goal: '/goals/',
		habit: '/habits/',
		project: '/projects/',
		palace: '/learn/palaces/'
	};
	const names: Record<string, string> = {
		note: 'Note',
		task: 'Task',
		goal: 'Goal',
		milestone: 'Milestone',
		habit: 'Habit',
		card: 'Flashcard',
		media: 'Film, series, or book',
		project: 'Project',
		palace: 'Memory palace',
		area: 'Area',
		resource: 'Resource',
		roadmap: 'Roadmap',
		skill: 'Skill',
		focus_session: 'Focus session'
	};

	async function load() {
		error = '';
		try {
			items = (await api<{ items: Item[] }>('/v1/trash')).items;
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function restore(i: Item) {
		try {
			const r = await api<{ entity_id: string; type: string }>(`/v1/trash/${i.id}/restore`, {
				method: 'POST'
			});
			toast(`“${i.title}” is back.`);
			if (pages[r.type] && r.type !== 'task') await goto(pages[r.type] + r.entity_id);
			else await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function forget(i: Item) {
		try {
			await api(`/v1/trash/${i.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function empty() {
		if (!confirm('Delete everything in the trash for good?')) return;
		try {
			await api('/v1/trash', { method: 'DELETE' });
			toast('The trash is empty.');
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const days = (iso: string) =>
		Math.max(0, Math.ceil((new Date(iso).getTime() - Date.now()) / 86_400_000));
</script>

<svelte:head><title>Trash · Axon</title></svelte:head>

<p class="muted">
	What you delete stays here for 30 days, with what was deleted along with it (a goal's milestones,
	a habit's check-ins, a card's reviews), then is deleted for good.
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !items}
	<Status kind="loading" message="Loading the trash…" />
{:else if items.length === 0}
	<Status kind="empty" message="The trash is empty." />
{:else}
	<ul class="rows">
		{#each items as i (i.id)}
			<li>
				<span class="grow">
					<strong>{i.title}</strong>
					<span class="muted small"
						>{names[i.type] ?? i.type}{i.with ? ` and ${i.with} more` : ''} · deleted {new Date(
							i.deleted_at
						).toLocaleString()} · {days(i.expires_at)} days left</span
					>
				</span>
				<Button variant="secondary" onclick={() => restore(i)} aria-label="Restore {i.title}"
					>Restore</Button
				>
				<Button variant="ghost" onclick={() => forget(i)} aria-label="Delete {i.title} for good"
					>Delete for good</Button
				>
			</li>
		{/each}
	</ul>
	<Button variant="danger" onclick={empty}>Empty the trash</Button>
{/if}

<style>
	.rows {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}
	.rows li {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		flex-wrap: wrap;
		padding: var(--space-2) var(--space-3);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
	}
	.grow {
		flex: 1;
		min-width: 12rem;
		display: grid;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
