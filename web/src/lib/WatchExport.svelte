<script lang="ts">
	import { api, message } from './api';
	import { t } from './i18n.svelte';
	import Button from './ui/Button.svelte';
	import Dialog from './ui/Dialog.svelte';
	import { toast } from './ui/toast.svelte';
	import { exportFormats, exportURL, shareQuery, type WatchFilter } from './watch';

	/** Export or share the news the reader's filter selects (EP-112). Every
	 * file and shared page says it comes from Axon Project. */
	let { open = $bindable(false), filter }: { open: boolean; filter: WatchFilter } = $props();

	let from = $state('');
	let to = $state('');
	let name = $state('');
	let days = $state(30);
	let link = $state('');
	let busy = $state(false);

	const chosen = $derived<WatchFilter>({
		...filter,
		from: from || undefined,
		to: to ? nextDay(to) : undefined
	});
	function nextDay(d: string) {
		const x = new Date(d + 'T00:00:00Z');
		x.setUTCDate(x.getUTCDate() + 1);
		return x.toISOString().slice(0, 10);
	}

	async function share(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			const res = await api<{ path: string }>('/v1/watch/shares', {
				method: 'POST',
				body: { name, query: shareQuery(chosen), days }
			});
			link = location.origin + res.path;
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	async function copy() {
		try {
			await navigator.clipboard.writeText(link);
			toast(t('Link copied.'));
		} catch {
			toast(t('Select the link and copy it.'), 'error');
		}
	}
	$effect(() => {
		if (!open) link = '';
	});
</script>

<Dialog bind:open title={t('Export or share')}>
	<p class="muted small">
		{t(
			'The news this view shows: its tab, kind, topic, label, and words. Narrow it to a period if you like.'
		)}
	</p>
	<div class="period">
		<label>{t('From')} <input type="date" bind:value={from} /></label>
		<label>{t('To')} <input type="date" bind:value={to} /></label>
	</div>

	<h3>{t('Download')}</h3>
	<ul class="formats">
		{#each exportFormats as [format, label] (format)}
			<li><a href={exportURL(format, chosen)} download>{t(label)}</a></li>
		{/each}
	</ul>
	<p class="muted small">{t('Each file says it comes from Axon Project.')}</p>

	<h3>{t('Share a read-only link')}</h3>
	{#if link}
		<p class="small">
			{t('Anyone with this link sees these news until it expires. It is shown only now.')}
		</p>
		<div class="row">
			<input
				readonly
				value={link}
				aria-label={t('Link')}
				onfocus={(e) => e.currentTarget.select()}
			/>
			<Button onclick={copy}>{t('Copy')}</Button>
		</div>
	{:else}
		<form onsubmit={share} class="share">
			<label
				>{t('Name')}
				<input bind:value={name} required maxlength="100" placeholder={t('e.g. the team')} /></label
			>
			<label
				>{t('Expires after')}
				<select bind:value={days}>
					<option value={7}>{t('7 days')}</option>
					<option value={30}>{t('30 days')}</option>
					<option value={90}>{t('90 days')}</option>
					<option value={365}>{t('a year')}</option>
				</select></label
			>
			<Button type="submit" {busy}>{t('Create the link')}</Button>
		</form>
		<p class="muted small">
			{t(
				'It shows titles, links, summaries, and labels; never what you read or wrote. From the Saved tab, it shares what you saved.'
			)}
			<a href="/watch/sources#shares">{t('Your links')}</a>
		</p>
	{/if}
	<div class="end">
		<Button variant="ghost" onclick={() => (open = false)}>{t('Close')}</Button>
	</div>
</Dialog>

<style>
	.small {
		font-size: var(--text-sm);
	}
	h3 {
		font-size: var(--text-base);
		margin: var(--space-4) 0 var(--space-2);
	}
	.period,
	.share,
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		align-items: end;
	}
	label {
		display: grid;
		gap: var(--space-1);
		font-size: var(--text-sm);
	}
	.row input {
		flex: 1;
		min-width: 0;
	}
	.formats {
		list-style: none;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.formats a {
		display: inline-block;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-1) var(--space-3);
		text-decoration: none;
		color: var(--fg);
	}
	.formats a:hover {
		background: var(--accent-soft);
	}
	.end {
		display: flex;
		justify-content: end;
		margin-top: var(--space-4);
	}
</style>
