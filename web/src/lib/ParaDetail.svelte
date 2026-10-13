<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, apiResponse, message } from '$lib/api';
	import Connections from '$lib/Connections.svelte';
	import { templates } from '$lib/noteTemplates';
	import type { ParaItem, ParaKind } from '$lib/para';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let { kind }: { kind: ParaKind } = $props();

	const id = $derived(page.params.id ?? '');
	let item = $state<ParaItem | null>(null);
	let etag = '';
	let error = $state('');
	let busy = $state(false);
	let confirmDelete = $state(false);
	let form = $state({ title: '', description: '', url: '' });
	let links = $state(0); // bumps Connections after we add a link

	async function load() {
		error = '';
		try {
			const res = await apiResponse<ParaItem>(`/v1/${kind.path}/${id}`);
			item = res.data;
			etag = res.etag;
			form = { title: item.title, description: item.description, url: item.url ?? '' };
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void id;
		load();
	});

	async function save(body: Record<string, string>, done: string) {
		busy = true;
		try {
			const res = await apiResponse<ParaItem>(`/v1/${kind.path}/${id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body
			});
			item = res.data;
			etag = res.etag;
			toast(done);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	function submit(e: SubmitEvent) {
		e.preventDefault();
		save(form, t('Saved.'));
	}

	async function remove() {
		busy = true;
		try {
			await api(`/v1/${kind.path}/${id}`, { method: 'DELETE' });
			await goto(`/${kind.path}`);
		} catch (err) {
			toast(message(err), 'error');
			busy = false;
		}
	}

	// A note from the area or resource template, filed under this item.
	async function newNote() {
		if (!item) return;
		const t = templates.find((x) => x.id === kind.type);
		if (!t) return;
		try {
			const now = new Date();
			const n = await api<{ id: string }>('/v1/notes', {
				method: 'POST',
				body: { path: t.path(item.title, now), content: t.content(item.title, now) }
			});
			await api('/v1/links', {
				method: 'POST',
				body: {
					source_id: n.id,
					relation: kind.type === 'area' ? 'in_area' : 'relates_to',
					target_id: item.id
				}
			});
			links++;
			await goto(`/notes/${n.id}?edit=1`);
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>{item?.title ?? t(kind.title)} · Axon</title></svelte:head>

<p><a href="/{kind.path}">← {t(kind.title)}</a></p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !item}
	<Status kind="loading" message={t('Loading…')} />
{:else}
	<div class="head">
		<h1>{item.title}</h1>
		{#if item.status === 'archived'}<span class="chip">{t('Archived')}</span>{/if}
	</div>
	<div class="layout">
		<section class="card" aria-labelledby="details">
			<h2 id="details">{t('Details')}</h2>
			<form onsubmit={submit}>
				<TextField
					id="para-title"
					label={t('Title')}
					bind:value={form.title}
					maxlength={500}
					required
				/>
				<div class="field">
					<label for="para-description"
						>{kind.type === 'area' ? t('Standard to keep') : t('What it is for')}</label
					>
					<textarea id="para-description" rows="4" maxlength="20000" bind:value={form.description}
					></textarea>
				</div>
				<TextField id="para-url" label={t('Link (optional)')} type="url" bind:value={form.url} />
				<div class="row">
					<Button type="submit" {busy}>{t('Save')}</Button>
					{#if item.url}<a href={item.url} rel="noopener noreferrer" target="_blank"
							>{t('Open link ↗')}</a
						>{/if}
				</div>
			</form>
			<div class="row actions">
				<Button variant="secondary" onclick={newNote}
					>{t('New {one} note', { one: kind.one })}</Button
				>
				{#if item.status === 'active'}
					<Button
						variant="secondary"
						onclick={() => save({ status: 'archived' }, `Archived. Find it in the Archive.`)}
						>{t('Archive')}</Button
					>
				{:else}
					<Button variant="secondary" onclick={() => save({ status: 'active' }, 'Restored.')}
						>{t('Restore')}</Button
					>
				{/if}
				<Button variant="danger" onclick={() => (confirmDelete = true)}>{t('Delete')}</Button>
			</div>
			<p class="muted small">
				{t(
					kind.type === 'area'
						? 'File projects, goals, habits, tasks, notes, and resources here with “In area” in Connections.'
						: 'Connect notes, skills, projects, and areas to this resource in Connections.'
				) +
					' ' +
					t('Notes for it live in')} <code>{kind.folder}/</code>
				{t('in your vault.')}
			</p>
		</section>
		{#key links}<Connections id={item.id} type={kind.type} onchange={load} />{/key}
	</div>
{/if}

<Dialog bind:open={confirmDelete} title={t('Delete this {one}?', { one: kind.one })}>
	<p>
		{t(
			'“{title}” is removed with its connections. What was filed under it stays. To keep it, archive it instead.',
			{ title: item?.title }
		)}
	</p>
	<div class="row end">
		<Button variant="secondary" onclick={() => (confirmDelete = false)}>{t('Cancel')}</Button>
		<Button variant="danger" onclick={remove} {busy}>{t('Delete')}</Button>
	</div>
</Dialog>

<style>
	.head {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}
	.chip {
		background: var(--accent-soft);
		border-radius: 999px;
		padding: 0 var(--space-3);
	}
	.layout {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(22rem, 1fr));
		gap: var(--space-4);
		align-items: start;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
	}
	.card h2 {
		font-size: var(--text-lg);
	}
	form {
		display: grid;
		gap: var(--space-3);
	}
	.field {
		display: grid;
		gap: var(--space-1);
	}
	.field label {
		font-weight: 600;
	}
	textarea {
		font: inherit;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		align-items: center;
	}
	.actions {
		margin-top: var(--space-6);
	}
	.end {
		justify-content: flex-end;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
