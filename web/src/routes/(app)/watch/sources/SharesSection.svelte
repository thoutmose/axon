<script lang="ts">
	import { appLocale } from '$lib/dates';
	import { api, message } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import { ago, type WatchShare } from '$lib/watch';

	// Share links (EP-112): read-only selections of news, revocable at once.
	let shares = $state<WatchShare[]>([]);
	async function loadShares() {
		try {
			({ items: shares } = await api<{ items: WatchShare[] }>('/v1/watch/shares'));
		} catch {
			// The page works without them.
		}
	}
	loadShares();
	async function revoke(s: WatchShare) {
		if (!confirm(t('Revoke “{name}”? The link stops working at once.', { name: s.name }))) return;
		try {
			await api(`/v1/watch/shares/${s.id}`, { method: 'DELETE' });
			shares = shares.filter((x) => x.id !== s.id);
			toast(t('Link revoked.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	const expired = (s: WatchShare) => new Date(s.expires_at) < new Date();
</script>

<Section id="shares" title={t('Shared links')}>
	<p class="muted small">
		{t(
			'Read-only links to a selection of your news, made from the reader with Export or share. Revoke one and it stops working at once.'
		)}
	</p>
	{#if shares.length}
		<div class="table">
			<table>
				<thead>
					<tr>
						<th>{t('Name')}</th>
						<th>{t('News')}</th>
						<th>{t('Expires')}</th>
						<th>{t('Views')}</th>
						<th><span class="sr-only">{t('Actions')}</span></th>
					</tr>
				</thead>
				<tbody>
					{#each shares as s (s.id)}
						<tr>
							<td><strong>{s.name}</strong></td>
							<td class="small">{s.description}</td>
							<td class="small" class:bad={expired(s)}>
								{expired(s) ? t('Expired') : new Date(s.expires_at).toLocaleDateString(appLocale())}
							</td>
							<td class="small">
								{s.views}
								{#if s.last_viewed_at}<div class="muted">{ago(s.last_viewed_at)}</div>{/if}
							</td>
							<td>
								<Button variant="ghost" onclick={() => revoke(s)}
									>{t('Revoke')}<span class="sr-only"> {s.name}</span></Button
								>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<p class="muted small">{t('No links yet.')}</p>
	{/if}
</Section>

<style>
	.table {
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	th,
	td {
		text-align: left;
		vertical-align: top;
		padding: var(--space-2);
		border-bottom: 1px solid var(--border);
	}
	td div {
		overflow-wrap: anywhere;
	}
	.small {
		font-size: var(--text-sm);
	}
	.bad {
		color: var(--bad);
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
		white-space: nowrap;
	}
</style>
