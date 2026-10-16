<script lang="ts">
	import { onMount } from 'svelte';
	import { api, ApiError, message, type Schemas } from '$lib/api';
	import { formatDate } from '$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { emptyDay, hasContent, scales, type JournalFields } from '$lib/journal';
	import Scale from '$lib/Scale.svelte';
	import WhatHelps from '$lib/WhatHelps.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// The daily journal (EP-91): how the day felt, the night's sleep, three
	// things to be grateful for, and a note; the last 30 days below, by date
	// only: past moods are not shown again (EP-130).
	type Entry = Schemas['JournalEntry'];
	let entries = $state<Entry[] | null>(null);
	let today = $state('');
	let day = $state('');
	let form = $state<JournalFields>(emptyDay());
	let consent = $state(true);
	let error = $state('');
	let busy = $state(false);
	const s = $derived(scales());

	async function load() {
		try {
			const r = await api<{ items: Entry[]; today: string }>('/v1/journal');
			entries = r.items;
			today = r.today;
			if (!day) open(r.today);
		} catch (err) {
			if (err instanceof ApiError && err.status === 403) consent = false;
			else error = message(err);
		}
	}
	onMount(load);

	function open(d: string) {
		day = d;
		const e = entries?.find((x) => x.day === d);
		form = e
			? {
					mood: e.mood,
					energy: e.energy,
					stress: e.stress,
					sleep_hours: e.sleep_hours,
					gratitude: [...e.gratitude, '', '', ''].slice(0, 3),
					note: e.note
				}
			: emptyDay();
	}

	async function allow() {
		try {
			await api('/v1/me/consents/health_data', { method: 'PUT', body: { granted: true } });
			consent = true;
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		if (!hasContent(form)) return;
		busy = true;
		try {
			await api(`/v1/journal/${day}`, {
				method: 'PUT',
				body: { ...form, gratitude: form.gratitude.filter((g) => g.trim()) }
			});
			toast(t('Saved.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function remove() {
		if (!confirm(t('Delete what you wrote that day?'))) return;
		try {
			await api(`/v1/journal/${day}`, { method: 'DELETE' });
			await load();
			open(day);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const saved = $derived(entries?.some((e) => e.day === day) ?? false);
</script>

<svelte:head><title>{t('Journal')} · Axon</title></svelte:head>

<h1>{t('Journal')}</h1>
<p class="muted hint">
	{t(
		'A minute a day: how you feel, how you slept, three things you are grateful for. Over weeks it shows what helps you.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !consent}
	<section class="card" aria-labelledby="consent">
		<h2 id="consent">{t('The journal is off')}</h2>
		<p>
			{t(
				'Mood and sleep are health data, a special category under the GDPR. Axon keeps them only with your explicit consent, with your food and training logs, out of search and AI agents, and deletes them all if you turn it off again.'
			)}
		</p>
		<Button onclick={allow}>{t('Turn on health data')}</Button>
	</section>
{:else if !entries}
	<Status kind="loading" message={t('Loading…')} />
{:else}
	<form class="card entry" onsubmit={save} aria-labelledby="entry-title">
		<div class="head">
			<h2 id="entry-title">
				{day === today
					? t('Today')
					: formatDate(day, { weekday: 'long', day: 'numeric', month: 'long' })}
			</h2>
			<label
				>{t('Day')}
				<input
					type="date"
					max={today}
					value={day}
					onchange={(e) => open(e.currentTarget.value)}
				/></label
			>
		</div>
		<Scale {...s.mood} bind:value={form.mood} />
		<Scale {...s.energy} bind:value={form.energy} />
		<Scale {...s.stress} bind:value={form.stress} />
		<label class="sleep"
			>{t('Hours of sleep last night')}
			<input
				type="number"
				min="0"
				max="24"
				step="0.5"
				inputmode="decimal"
				bind:value={form.sleep_hours}
			/></label
		>
		<fieldset class="thanks">
			<legend>{t('Three things you are grateful for')}</legend>
			{#each form.gratitude as _, i (i)}
				<input
					bind:value={form.gratitude[i]}
					maxlength="300"
					aria-label={t('Grateful for, {n} of 3', { n: i + 1 })}
					placeholder={i === 0 ? t('A call with an old friend…') : ''}
				/>
			{/each}
		</fieldset>
		<label
			>{t('Note')}
			<textarea rows="4" maxlength="10000" bind:value={form.note}></textarea></label
		>
		<div class="row">
			<Button type="submit" {busy} disabled={!hasContent(form)}>{t('Save')}</Button>
			{#if saved}<Button variant="ghost" onclick={remove}>{t('Delete')}</Button>{/if}
		</div>
	</form>

	<WhatHelps />

	{#if entries.length}
		<section class="card" aria-labelledby="month-title">
			<h2 id="month-title">{t('The last 30 days')}</h2>
			<ul class="days">
				{#each entries as e (e.day)}
					<li>
						<button type="button" class="link" onclick={() => open(e.day)}
							>{formatDate(e.day, { weekday: 'short', day: 'numeric', month: 'short' })}</button
						>
						{#if e.sleep_hours != null}<span class="small"
								>{t('{hours} h of sleep', { hours: e.sleep_hours })}</span
							>{/if}
						<span class="small muted excerpt">{e.gratitude[0] ?? e.note.slice(0, 80)}</span>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
{/if}

<style>
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.entry {
		display: grid;
		gap: var(--space-4);
	}
	.head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: end;
		gap: var(--space-3);
	}
	.head h2 {
		margin: 0;
	}
	label {
		display: grid;
		gap: var(--space-1);
	}
	.sleep input {
		max-width: 8rem;
	}
	.thanks {
		border: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: var(--space-2);
	}
	.thanks legend {
		font-weight: 600;
		margin-bottom: var(--space-1);
	}
	textarea {
		font: inherit;
	}
	.row {
		display: flex;
		gap: var(--space-2);
	}
	.days {
		list-style: none;
		padding: 0;
		margin: var(--space-3) 0 0;
		display: grid;
		gap: var(--space-2);
	}
	.days li {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-3);
	}
	.link {
		background: none;
		border: none;
		padding: 0;
		color: var(--accent);
		font: inherit;
		cursor: pointer;
		min-width: 7rem;
		text-align: left;
	}
	.excerpt {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
