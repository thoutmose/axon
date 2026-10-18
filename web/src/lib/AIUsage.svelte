<script lang="ts">
	import { appLocale } from './dates';
	import { api, message } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import { formatDate } from '$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import Status from '$lib/ui/Status.svelte';

	// AI tokens (EP-152): what every agent, meal photo, video summary, and
	// Jarvis used, this month and all along, and what compacting the tools'
	// results saved, measured.
	type Usage = paths['/v1/me/ai-usage']['get']['responses'][200]['content']['application/json'];
	let usage = $state<Usage | null>(null);
	let error = $state('');
	async function load() {
		try {
			usage = await api<Usage>('/v1/me/ai-usage');
		} catch (err) {
			error = message(err);
		}
	}
	load();

	const n = (v: number) => v.toLocaleString(appLocale());
	const dollars = (micros: number) =>
		(micros / 1_000_000).toLocaleString(appLocale(), { style: 'currency', currency: 'USD' });
	// About four characters a token: an estimate, said as one.
	const saved = (raw: number, sent: number) => Math.round((raw - sent) / 4);
	const share = (raw: number, sent: number) => (raw ? Math.round(((raw - sent) / raw) * 100) : 0);
	const peak = $derived(
		Math.max(1, ...(usage?.by_month ?? []).map((m) => m.input_tokens + m.output_tokens))
	);
	const names: Record<string, string> = $derived({
		'food-photo': t('Meal photos'),
		'video-summary': t('Video summaries'),
		jarvis: 'Jarvis'
	});
</script>

<section class="usage" aria-labelledby="usage-title">
	<h3 id="usage-title">{t('AI tokens')}</h3>
	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !usage}
		<Status kind="loading" message={t('Adding it up…')} />
	{:else}
		<dl class="tiles">
			<div>
				<dt>{t('This month')}</dt>
				<dd>{n(usage.this_month.input_tokens + usage.this_month.output_tokens)}</dd>
				<dd class="muted small">
					{t('{in} in, {out} out · {cost}', {
						in: n(usage.this_month.input_tokens),
						out: n(usage.this_month.output_tokens),
						cost: dollars(usage.this_month.cost_micros)
					})}
				</dd>
			</div>
			<div>
				<dt>{t('All along')}</dt>
				<dd>{n(usage.all_time.input_tokens + usage.all_time.output_tokens)}</dd>
				<dd class="muted small">
					{t('{runs} calls · {cost}', {
						runs: n(usage.all_time.runs),
						cost: dollars(usage.all_time.cost_micros)
					})}
				</dd>
			</div>
			<div>
				<dt>{t('Saved by compaction')}</dt>
				<dd>
					{share(usage.all_time.tool_chars_raw, usage.all_time.tool_chars_sent)} %
				</dd>
				<dd class="muted small">
					{t('of tool results: {raw} → {sent} characters, about {tokens} tokens', {
						raw: n(usage.all_time.tool_chars_raw),
						sent: n(usage.all_time.tool_chars_sent),
						tokens: n(saved(usage.all_time.tool_chars_raw, usage.all_time.tool_chars_sent))
					})}
				</dd>
			</div>
		</dl>

		<ol class="months" aria-label={t('Tokens by month')}>
			{#each usage.by_month as m (m.month)}
				<li title="{m.month}: {n(m.input_tokens + m.output_tokens)} · {dollars(m.cost_micros)}">
					<span class="bar" style:height="{((m.input_tokens + m.output_tokens) / peak) * 100}%"
					></span>
					<span class="label">{formatDate(`${m.month}-01`, { month: 'narrow' })}</span>
				</li>
			{/each}
		</ol>

		{#if usage.by_agent.length}
			<div class="table">
				<table>
					<thead>
						<tr>
							<th scope="col">{t('Agent')}</th>
							<th scope="col" class="num">{t('Calls')}</th>
							<th scope="col" class="num">{t('Tokens in')}</th>
							<th scope="col" class="num">{t('Tokens out')}</th>
							<th scope="col" class="num">{t('Cost')}</th>
							<th scope="col" class="num">{t('Compaction')}</th>
						</tr>
					</thead>
					<tbody>
						{#each usage.by_agent as a (a.agent)}
							<tr>
								<td>{names[a.agent] ?? a.agent}</td>
								<td class="num">{n(a.runs)}</td>
								<td class="num">{n(a.input_tokens)}</td>
								<td class="num">{n(a.output_tokens)}</td>
								<td class="num">{dollars(a.cost_micros)}</td>
								<td class="num"
									>{a.tool_chars_raw ? `−${share(a.tool_chars_raw, a.tool_chars_sent)} %` : '—'}</td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<p class="muted small">{t('No AI call yet.')}</p>
		{/if}
		<p class="muted small">
			{t(
				'Axon compacts what its tools hand the model, as rtk does for a coding agent’s commands: empty fields left out, times to the minute, lists of records as rows. The saving above is measured on your own calls. rtk itself compresses shell output for coding agents such as Claude Code, which can use it on the server; Axon’s agents run no commands, so it does not apply to them.'
			)}
		</p>
	{/if}
</section>

<style>
	.usage {
		margin-top: var(--space-6);
	}
	h3 {
		margin: 0 0 var(--space-2);
		font-size: var(--text-base);
	}
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
		gap: var(--space-3);
		margin: 0 0 var(--space-3);
	}
	dt {
		color: var(--muted);
		font-size: var(--text-sm);
	}
	dd {
		margin: 0;
		font-size: 1.4rem;
		font-weight: 650;
	}
	dd.small {
		font-size: var(--text-sm);
		font-weight: 400;
	}
	.months {
		list-style: none;
		display: grid;
		grid-template-columns: repeat(12, 1fr);
		gap: 4px;
		align-items: end;
		height: 5rem;
		padding: 0;
		margin: 0 0 var(--space-3);
	}
	.months li {
		display: flex;
		flex-direction: column;
		justify-content: flex-end;
		height: 100%;
	}
	.bar {
		display: block;
		background: var(--accent);
		border-radius: 3px 3px 0 0;
		min-height: 2px;
	}
	.label {
		text-align: center;
		font-size: 0.7rem;
		color: var(--muted);
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	th,
	td {
		padding: var(--space-1) var(--space-2);
		border-bottom: 1px solid var(--border);
		text-align: left;
	}
	.num {
		text-align: right;
		font-variant-numeric: tabular-nums;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
