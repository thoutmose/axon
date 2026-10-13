<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { localDate } from '$lib/dates';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import { refreshNotes } from '$lib/noteIndex.svelte';
	import { keep, unreachable } from '$lib/outbox.svelte';
	import { fleeting } from '$lib/quickAddApi';
	import { capture } from '$lib/tasks';
	import Button from '$lib/ui/Button.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Share to Axon (EP-67): what another app shares (the installed app's
	// share sheet) or the bookmarklet sends arrives here as title, text, url.
	type Note = Schemas['Note'];
	const q = page.url.searchParams;
	// Some apps put the link in the text; take it out so it is kept once.
	const urlIn = q.get('url') || (q.get('text') ?? '').match(/https?:\/\/\S+/)?.[0] || '';
	let title = $state((q.get('title') ?? '').trim());
	let text = $state((q.get('text') ?? '').replace(urlIn, '').trim());
	let url = $state(urlIn);
	let busy = $state('');

	const body = () => [text, url].filter(Boolean).join('\n\n');
	const heading = () => title || text.split('\n')[0].slice(0, 120) || url || t('Shared');

	async function asNote() {
		busy = 'note';
		const n = fleeting(heading(), []);
		n.content = n.content + body() + '\n';
		try {
			const made = await api<Note>('/v1/notes', { method: 'POST', body: n });
			refreshNotes().catch(() => {});
			toast(t('Saved as a fleeting note.'));
			await goto(`/notes/${made.id}`);
		} catch (err) {
			if (unreachable(err)) {
				keep({ kind: 'note', title: heading(), ...n });
				toast(t('No connection: kept on this device, sent when you are back online.'));
				await goto('/today');
			} else toast(message(err), 'error');
		} finally {
			busy = '';
		}
	}

	async function asTask() {
		busy = 'task';
		try {
			await capture(heading(), { notes: body() });
			toast(t('Captured to your inbox.'));
			await goto('/tasks?view=inbox');
		} catch (err) {
			if (unreachable(err)) {
				keep({ kind: 'task', title: heading() });
				toast(t('No connection: kept on this device, sent when you are back online.'));
				await goto('/today');
			} else toast(message(err), 'error');
		} finally {
			busy = '';
		}
	}

	/** A literature note with the page's own text, to read and annotate later. */
	async function asLiterature() {
		busy = 'page';
		try {
			const p = await api<{ title: string; site: string; markdown: string; url: string }>(
				'/v1/clip',
				{
					method: 'POST',
					body: { url }
				}
			);
			const name = (title || p.title).trim() || p.site;
			const slug =
				name
					.toLowerCase()
					.replace(/[^\p{L}\p{N}]+/gu, '-')
					.replace(/^-|-$/g, '')
					.slice(0, 60) || 'page';
			const day = localDate(new Date());
			const yaml = (s: string) => JSON.stringify(s);
			const content = `---\ntype: literature\ntitle: ${yaml(name)}\nsource: ${yaml(p.url)}\nsite: ${yaml(p.site)}\nclipped: ${day}\ntags: []\n---\n# ${name}\n\n${text ? `> ${text.replace(/\n/g, '\n> ')}\n\n` : ''}## My thoughts\n\n\n## The page\n\n${p.markdown}\n`;
			const made = await api<Note>('/v1/notes', {
				method: 'POST',
				body: { path: `Zettel/literature/${slug}-${Date.now().toString(36)}.md`, content }
			});
			refreshNotes().catch(() => {});
			toast(t('Saved the page as a literature note.'));
			await goto(`/notes/${made.id}?edit=1`);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = '';
		}
	}

	const bookmarklet = $derived(
		`javascript:(()=>{location.href=${JSON.stringify(page.url.origin + '/share?')}+new URLSearchParams({title:document.title,url:location.href,text:String(getSelection())})})()`
	);
</script>

<svelte:head><title>{t('Save to Axon')}</title></svelte:head>

<h1>{t('Save to Axon')}</h1>

<form class="card" onsubmit={(e) => e.preventDefault()}>
	<TextField id="share-title" label={t('Title')} bind:value={title} maxlength={300} />
	<div class="field">
		<label for="share-text">{t('Text')}</label>
		<textarea id="share-text" rows="4" bind:value={text}></textarea>
	</div>
	<TextField id="share-url" label={t('Link')} bind:value={url} maxlength={2000} />
	<div class="row">
		<Button onclick={asNote} busy={busy === 'note'} disabled={!title && !text && !url}
			>{t('Save as a note')}</Button
		>
		<Button
			variant="secondary"
			onclick={asTask}
			busy={busy === 'task'}
			disabled={!title && !text && !url}>{t('Save as a task')}</Button
		>
		{#if url}
			<Button variant="secondary" onclick={asLiterature} busy={busy === 'page'}
				>{t("Save the page's text as a literature note")}</Button
			>
		{/if}
	</div>
</form>

<section class="card" aria-labelledby="how-title">
	<h2 id="how-title">{t('Save from anywhere')}</h2>
	<p>
		<strong>{t('On a phone:')}</strong>
		{t("install Axon (Account → Preferences → App), then use your browser's or any app's")}
		<em>{t('Share')}</em>
		{t('button and choose Axon.')}
	</p>
	<p>
		<strong>{t('On a computer:')}</strong>
		{t(
			'drag this button to your bookmarks bar, then click it on any page (select some text first to save it too):'
		)}
		<!-- A bookmarklet runs on the page you are reading, not here. -->
		<a class="bookmarklet" href={bookmarklet} onclick={(e) => e.preventDefault()}
			>{t('Save to Axon')}</a
		>
	</p>
</section>

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
		max-width: 44rem;
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
		max-width: 28rem;
	}
	.field label {
		font-weight: 600;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.bookmarklet {
		display: inline-block;
		padding: var(--space-1) var(--space-3);
		border-radius: 999px;
		border: 1px dashed var(--accent);
		cursor: grab;
	}
</style>
