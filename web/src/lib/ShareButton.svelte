<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { drawCard, socialLinks, type ShareSpec } from '$lib/shareImage';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Share an element as a picture (EP-121): drawn on this device, then the
	// device's share sheet, a download, a copy, or a post with its words.
	let {
		spec,
		text,
		label = t('Share')
	}: { spec: ShareSpec; text: string; label?: string } = $props();

	const uid = $props.id(); // one dialog per button: its own ids
	let dialog: HTMLDialogElement;
	let blob = $state<Blob | null>(null);
	let preview = $state('');

	async function open() {
		blob = await drawCard(spec, t('Made with Axon, my life OS'));
		preview = await new Promise<string>((resolve) => {
			const r = new FileReader();
			r.onload = () => resolve(r.result as string);
			r.readAsDataURL(blob!);
		});
		dialog.showModal();
	}
	const file = () => new File([blob!], 'axon.png', { type: 'image/png' });
	const canShareFiles = () =>
		typeof navigator !== 'undefined' && !!navigator.canShare?.({ files: [file()] });

	async function share() {
		try {
			await navigator.share({ files: [file()], text });
		} catch {
			// cancelled
		}
	}
	function download() {
		const a = document.createElement('a');
		a.href = preview;
		a.download = 'axon.png';
		a.click();
	}
	async function copy() {
		try {
			await navigator.clipboard.write([new ClipboardItem({ 'image/png': blob! })]);
			toast(t('Picture copied.'));
		} catch {
			toast(t('This browser cannot copy pictures; download it instead.'), 'error');
		}
	}
</script>

<Button variant="ghost" onclick={open}>{label}</Button>

<dialog bind:this={dialog} class="share" aria-labelledby="{uid}-title">
	<h2 id="{uid}-title">{t('Share')}</h2>
	{#if preview}
		<img
			src={preview}
			alt={`${spec.title}: ${spec.value}. ${spec.caption}`}
			width="600"
			height="315"
		/>
	{/if}
	<div class="row">
		{#if blob && canShareFiles()}<Button onclick={share}>{t('Share…')}</Button>{/if}
		<Button variant="secondary" onclick={download}>{t('Download the picture')}</Button>
		<Button variant="secondary" onclick={copy}>{t('Copy the picture')}</Button>
	</div>
	<p class="small muted">{t('Post it, then attach the picture:')}</p>
	<div class="row">
		{#each socialLinks(text) as l (l.name)}
			<a class="post" href={l.url} target="_blank" rel="noopener noreferrer"
				>{t('Post on {network}', { network: l.name })}</a
			>
		{/each}
	</div>
	<p class="small muted">
		{t('The picture is made on this device; nothing is sent until you share it.')}
	</p>
	<form method="dialog"><Button type="submit" variant="ghost">{t('Close')}</Button></form>
</dialog>

<style>
	.share {
		width: min(40rem, calc(100vw - 2rem));
		border: 1px solid var(--border);
		border-radius: var(--radius-xl);
		background: var(--surface-solid);
		color: var(--fg);
		padding: var(--space-4);
	}
	img {
		width: 100%;
		height: auto;
		border-radius: var(--radius-lg);
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin: var(--space-3) 0;
	}
	.post {
		padding: var(--space-1) var(--space-3);
		border: 1px solid var(--border);
		border-radius: 999px;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
