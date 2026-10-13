<script lang="ts">
	import { onMount } from 'svelte';
	import { api, message } from '$lib/api';
	import type { operations } from '$lib/api-types';
	import { i18n, t } from '$lib/i18n.svelte';

	// What helps you (EP-92): the clear links in the last 90 days between a
	// day's habits and how it felt, worked out on the server, explained, and
	// never presented as causes.
	type Helps = operations['getWhatHelps']['responses'][200]['content']['application/json'];
	let helps = $state<Helps | null>(null);
	let error = $state('');

	onMount(async () => {
		try {
			helps = await api<Helps>('/v1/journal/helps');
		} catch (err) {
			error = message(err);
		}
	});
	const pct = (v: number) => `${(v / 5) * 100}%`;
	const num = (v: number, digits: number) =>
		v.toLocaleString(i18n.locale, { minimumFractionDigits: digits, maximumFractionDigits: digits });
</script>

<section class="card" aria-labelledby="helps-title">
	<h2 id="helps-title">{t('What helps you')}</h2>
	{#if error}
		<p class="muted">{error}</p>
	{:else if !helps}
		<p class="muted">{t('Loading…')}</p>
	{:else if helps.needed > 0}
		<p class="muted">
			{helps.needed === 1
				? t('Write in your journal on {n} more day to see what goes with your better days.', {
						n: helps.needed
					})
				: t('Write in your journal on {n} more days to see what goes with your better days.', {
						n: helps.needed
					})}
		</p>
	{:else if !helps.findings.length}
		<p class="muted">
			{t(
				'Nothing stands out yet in the last 90 days ({days} days written). That is an answer too: keep writing, and look again in a few weeks.',
				{ days: helps.days }
			)}
		</p>
	{:else}
		<ul class="findings">
			{#each helps.findings as f (f.factor + f.outcome)}
				<li>
					<p>
						<span class="chip"
							>{f.strength === 'strong' ? t('Strong link') : t('Moderate link')}</span
						>
						{f.sentence}
					</p>
					<div class="bars" aria-hidden="true">
						<span class="label small">{t('Those days')}</span>
						<span class="track"><span class="with" style:width={pct(f.mean_with)}></span></span>
						<span class="small">{num(f.mean_with, 1)}</span>
						<span class="label small">{t('Other days')}</span>
						<span class="track"
							><span class="without" style:width={pct(f.mean_without)}></span></span
						>
						<span class="small">{num(f.mean_without, 1)}</span>
					</div>
					<p class="small muted">
						{t('Rank correlation {rho} · q = {q} · {n} days', {
							rho: num(f.rho, 2),
							q: f.q < 0.001 ? `< ${num(0.001, 3)}` : num(f.q, 3),
							n: f.n
						})}
					</p>
				</li>
			{/each}
		</ul>
	{/if}
	{#if helps}
		<p class="small caveat">
			{t(
				'A link is not a cause: a good night may come with a calm day for other reasons. Use it as a hint to try, not a rule.'
			)}
		</p>
		<details>
			<summary>{t('How is this worked out?')}</summary>
			<p class="small">{helps.method}</p>
		</details>
	{/if}
</section>

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.findings {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: var(--space-4);
	}
	.findings p {
		margin: 0 0 var(--space-2);
	}
	.chip {
		display: inline-block;
		font-size: var(--text-sm);
		padding: 0 var(--space-2);
		border-radius: 999px;
		background: var(--accent-soft);
		color: var(--fg);
		margin-right: var(--space-1);
	}
	.bars {
		display: grid;
		grid-template-columns: 7rem 1fr 2.5rem;
		align-items: center;
		gap: var(--space-1) var(--space-3);
		max-width: 32rem;
	}
	.track {
		height: 0.6rem;
		border-radius: 999px;
		background: var(--border);
		overflow: hidden;
	}
	.track span {
		display: block;
		height: 100%;
	}
	.with {
		background: var(--chart-1);
	}
	.without {
		background: var(--muted);
	}
	.small {
		font-size: var(--text-sm);
	}
	.caveat {
		border-left: 4px solid var(--accent);
		padding-left: var(--space-3);
		margin-top: var(--space-4);
	}
	summary {
		cursor: pointer;
		color: var(--accent);
	}
</style>
