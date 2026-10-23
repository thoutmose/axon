<script lang="ts">
	import { api, message } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import type { WatchSource, WatchTopic } from '$lib/watch';
	import { presets } from './presets';

	// Follow a source: find its feed from an address, then pick how often
	// to check it and which topics it feeds.
	let { topics, onchange }: { topics: WatchTopic[]; onchange: () => Promise<void> } = $props();

	let address = $state('');
	let preview = $state<{
		url: string;
		title: string;
		recent: string[];
		entries: number;
	} | null>(null);
	let picked = $state<string[]>([]);
	let schedule = $state('0 0 * * *');
	let custom = $state('');
	let fullText = $state(false);
	let news = $state(false); // general news, apart from the tech watch (EP-155)
	let busy = $state(false);

	async function find(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		preview = null;
		try {
			preview = await api('/v1/watch/discover', { method: 'POST', body: { url: address } });
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	async function follow() {
		busy = true;
		try {
			const r = await api<{ source: WatchSource; stats: { new: number; merged: number } }>(
				'/v1/watch/sources',
				{
					method: 'POST',
					body: {
						url: preview?.url ?? address,
						topic_ids: picked,
						schedule: schedule === 'custom' ? custom : schedule,
						full_text: fullText,
						news
					}
				}
			);
			toast(
				t('Following {name}: {n} news stored, {m} already known.', {
					name: r.source.title,
					n: r.stats.new,
					m: r.stats.merged
				})
			);
			address = '';
			preview = null;
			await onchange();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
</script>

<Section id="follow" title={t('Follow a source')}>
	<form class="row" onsubmit={find}>
		<input
			aria-label={t('Address')}
			placeholder={t('A feed, a site, medium.com/@author, name.substack.com, a YouTube channel…')}
			bind:value={address}
			required
		/>
		<Button type="submit" variant="secondary" {busy}>{t('Find its feed')}</Button>
	</form>
	{#if preview}
		<div class="preview">
			<p><strong>{preview.title}</strong> <span class="muted small">{preview.url}</span></p>
			<ul class="small">
				{#each preview.recent as r, i (i)}<li>{r}</li>{/each}
			</ul>
			<div class="grid">
				<div class="field">
					<label for="sched">{t('Check it')}</label>
					<select id="sched" bind:value={schedule}>
						{#each presets as [v, l] (v)}<option value={v}>{t(l)}</option>{/each}
						<option value="custom">{t('Custom (cron)')}</option>
					</select>
					{#if schedule === 'custom'}
						<input aria-label={t('Cron schedule')} placeholder="30 6 * * 1-5" bind:value={custom} />
					{/if}
				</div>
				{#if topics.length}
					<fieldset class="field">
						<legend>{t('Topics')}</legend>
						{#each topics as tp (tp.id)}
							<label class="check"
								><input type="checkbox" value={tp.id} bind:group={picked} />{tp.name}</label
							>
						{/each}
					</fieldset>
				{/if}
			</div>
			<label class="check"
				><input type="checkbox" bind:checked={fullText} />{t(
					'Read each article’s page when the feed has an excerpt only (Medium, Substack previews, news sites)'
				)}</label
			>
			<label class="check"
				><input type="checkbox" bind:checked={news} />{t(
					'General news, not tech: kept out of the tech watch, for the News tab and the news digest'
				)}</label
			>
			<Button {busy} onclick={follow}>{t('Follow')}</Button>
		</div>
	{/if}
</Section>

<style>
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
	}
	.row input {
		flex: 1 1 14rem;
		min-width: 0;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 18rem), 1fr));
		gap: var(--space-4);
		margin: var(--space-3) 0;
	}
	.field {
		display: grid;
		gap: var(--space-1);
		border: 0;
		padding: 0;
		margin: 0;
	}
	.check {
		display: flex;
		gap: var(--space-1);
		align-items: baseline;
	}
	.preview {
		margin-top: var(--space-3);
		padding: var(--space-3);
		border: 1px solid var(--border);
		border-radius: var(--radius);
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
