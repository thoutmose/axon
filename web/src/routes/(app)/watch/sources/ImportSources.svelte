<script lang="ts">
	import { api, message } from '$lib/api';
	import { storedLocale, t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import { csvHeader } from '$lib/watch';

	// Import sources from a CSV, OPML, or JSON file, or from FreshRSS.
	let { onchange }: { onchange: () => Promise<void> } = $props();

	type ImportResult = {
		added: number;
		existing: number;
		topics_created: number;
		failed: { line: number; url: string; reason: string }[];
	};
	let imported = $state<ImportResult | null>(null);
	let busy = $state(false);

	async function importFile(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		// A .json is YouTube Subscription Manager's export (EP-165).
		const format = /\.csv$/i.test(file.name)
			? 'csv'
			: /\.json$/i.test(file.name)
				? 'youtube'
				: 'opml';
		try {
			const res = await fetch(`/api/v1/watch/import?format=${format}`, {
				method: 'POST',
				headers: {
					'Accept-Language': storedLocale(),
					'Content-Type':
						format === 'csv'
							? 'text/csv'
							: format === 'youtube'
								? 'application/json'
								: 'text/x-opml'
				},
				body: await file.text()
			});
			const out = await res.json();
			if (!res.ok) throw new Error(out.detail ?? res.statusText);
			imported = out;
			await onchange();
		} catch (err) {
			toast(err instanceof Error ? err.message : String(err), 'error');
		}
		input.value = '';
	}
	let fresh = $state({ url: '', user: '', password: '', news: false });
	async function importFresh(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			imported = await api<ImportResult>('/v1/watch/import/freshrss', {
				method: 'POST',
				body: fresh
			});
			fresh.password = '';
			await onchange();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	const sampleCSV = `${csvHeader}\nhttps://medium.com/@someone,Someone on Medium,AI|Writing,0 7 * * *,yes\nhttps://example.substack.com,,Databases,,\n`;
</script>

<Section id="import" title={t('Import')}>
	<div class="grid">
		<div>
			<label class="file"
				>{t('A CSV, OPML, or JSON file')}
				<input
					type="file"
					accept=".csv,.opml,.xml,.json,text/csv,text/x-opml,application/json"
					onchange={importFile}
				/></label
			>
			<p class="muted small">
				{t('OPML comes from Feedly, FreshRSS, Inoreader, and most readers; folders become topics.')}
				{t(
					"YouTube subscriptions: Google Takeout's subscriptions.csv, or the JSON of YouTube Subscription Manager, whose folders become topics."
				)}
				{t('CSV columns')}: <code>{csvHeader}</code> ({t('topics separated by |')}).
				<a
					href="data:text/csv;charset=utf-8,{encodeURIComponent(sampleCSV)}"
					download="axon-sources.csv">{t('Example CSV')}</a
				>
			</p>
		</div>
		<form onsubmit={importFresh} class="fresh">
			<strong>FreshRSS</strong>
			<input
				aria-label={t('FreshRSS address')}
				placeholder="https://rss.example.org"
				bind:value={fresh.url}
				required
			/>
			<input aria-label={t('User')} placeholder={t('User')} bind:value={fresh.user} required />
			<input
				aria-label={t('API password')}
				type="password"
				placeholder={t('API password')}
				autocomplete="off"
				bind:value={fresh.password}
				required
			/>
			<label class="check small"
				><input type="checkbox" bind:checked={fresh.news} />{t(
					'These feeds are general news'
				)}</label
			>
			<Button type="submit" variant="secondary" {busy}>{t('Import its subscriptions')}</Button>
			<p class="muted small">
				{t(
					'Set an API password in FreshRSS: Profile → API management. It is used once, not stored.'
				)}
			</p>
		</form>
	</div>
	{#if imported}
		<p role="status">
			{t('{a} added, {e} already followed, {t} topics created.', {
				a: imported.added,
				e: imported.existing,
				t: imported.topics_created
			})}
		</p>
		{#if imported.failed.length}
			<ul class="small bad">
				{#each imported.failed as f (f.line)}<li>
						{t('Line')}
						{f.line}: {f.url} — {f.reason}
					</li>{/each}
			</ul>
		{/if}
	{/if}
</Section>

<style>
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 18rem), 1fr));
		gap: var(--space-4);
		margin: var(--space-3) 0;
	}
	.check {
		display: flex;
		gap: var(--space-1);
		align-items: baseline;
	}
	.fresh {
		display: grid;
		gap: var(--space-2);
	}
	.file {
		display: grid;
		gap: var(--space-1);
		font-weight: 600;
	}
	.small {
		font-size: var(--text-sm);
	}
	.bad {
		color: var(--bad);
	}
</style>
