<script lang="ts">
	import ShareButton from '$lib/ShareButton.svelte';
	import { t } from '$lib/i18n.svelte';
	import { deleted } from '$lib/undo';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, apiResponse, message } from '$lib/api';
	import { isoWeekday } from '$lib/dates';
	import Connections from '$lib/Connections.svelte';
	import { lastDays, scheduleLabel, weekdays, type Habit, type HabitCheckin } from '$lib/habits';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	const id = $derived(page.params.id ?? '');
	let habit = $state<Habit | null>(null);
	let etag = '';
	let error = $state('');
	let busy = $state(false);
	let checkins = $state<Record<string, string>>({});
	let confirmDelete = $state(false);
	let form = $state({
		title: '',
		cue: '',
		craving: '',
		reward: '',
		stack_after: '',
		two_minute: '',
		days: [] as number[],
		status: 'active' as Habit['status'],
		kind: 'build' as Habit['kind'],
		identity: '',
		intention_time: '',
		intention_place: '',
		contract_partner: '',
		contract_pledge: ''
	});

	async function load() {
		error = '';
		try {
			const res = await apiResponse<Habit>(`/v1/habits/${id}`);
			habit = res.data;
			etag = res.etag;
			const h = habit;
			form = {
				title: h.title,
				cue: h.cue,
				craving: h.craving,
				reward: h.reward,
				stack_after: h.stack_after,
				two_minute: h.two_minute,
				days: [...h.days],
				status: h.status,
				kind: h.kind,
				identity: h.identity,
				intention_time: h.intention_time,
				intention_place: h.intention_place,
				contract_partner: h.contract_partner,
				contract_pledge: h.contract_pledge
			};
			const hist = await api<{ items: HabitCheckin[] }>(`/v1/habits/${id}/checkins`);
			checkins = Object.fromEntries(hist.items.map((c) => [c.day, c.status]));
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void id;
		habit = null;
		load();
	});

	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			const res = await apiResponse<Habit>(`/v1/habits/${id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: form
			});
			habit = res.data;
			etag = res.etag;
			toast(t('Saved.'));
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function remove() {
		busy = true;
		try {
			await api(`/v1/habits/${id}`, { method: 'DELETE' });
			deleted(t('Habit deleted.'), id, () => goto(`/habits/${id}`));
			await goto('/habits');
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	const dayState: Record<string, string> = $derived({
		done: 'done',
		two_minute: t('done (two-minute version)'),
		skipped: t('rest day')
	});
	function describe(day: string): string {
		if (checkins[day]) return dayState[checkins[day]];
		return habit?.days.includes(isoWeekday(day)) ? t('not checked in') : t('not scheduled');
	}
</script>

<svelte:head><title>{habit?.title ?? t('Habit')} · Axon</title></svelte:head>

<p class="crumbs"><a href="/habits">{t('← Habits')}</a></p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !habit}
	<Status kind="loading" message={t('Loading habit…')} />
{:else}
	<h1>{habit.title}</h1>
	<p class="muted">
		{scheduleLabel(habit.days) + (habit.status !== 'active' ? ` · ${t(habit.status)}` : '')}
	</p>
	{#if habit.state.streak > 0}
		<ShareButton
			spec={{
				title: habit.title,
				value: t('{n} days', { n: habit.state.streak }),
				caption: t('Kept {n} scheduled days in a row.', { n: habit.state.streak }),
				kind: 'accent'
			}}
			text={t('{n} days in a row: {habit}. #Axon', { n: habit.state.streak, habit: habit.title })}
		/>
	{/if}

	<section class="card" aria-labelledby="history-title">
		<h2 id="history-title">{t('Last four weeks')}</h2>
		<ol class="grid">
			{#each lastDays(habit.state.today, 28) as day (day)}
				<li
					class="cell {checkins[day] ?? ''}"
					class:off={!habit.days.includes(isoWeekday(day))}
					title="{day}: {describe(day)}"
				>
					<span class="visually-hidden">{day}: {describe(day)}</span>
				</li>
			{/each}
		</ol>
		<p class="muted small legend">
			<span class="cell done"></span>
			{t('done')} <span class="cell two_minute"></span>
			{t('two-minute')}
			<span class="cell skipped"></span>
			{t('rest')} <span class="cell"></span>
			{t('open')}
		</p>
	</section>

	<Connections id={habit.id} type="habit" />

	<section class="card" aria-labelledby="design-title">
		<h2 id="design-title">{t('Design')}</h2>
		<form onsubmit={save}>
			<TextField id="h-title" label={t('Habit')} bind:value={form.title} maxlength={500} required />
			<fieldset class="kind">
				<legend>{t('Kind')}</legend>
				<label><input type="radio" value="build" bind:group={form.kind} /> {t('Build')}</label>
				<label><input type="radio" value="break" bind:group={form.kind} /> {t('Break')}</label>
			</fieldset>
			<TextField
				id="h-identity"
				label={t('Identity')}
				hint={t('Who you are becoming; each check-in is a vote.')}
				bind:value={form.identity}
				maxlength={500}
			/>
			<div class="pair">
				<div class="field">
					<label for="h-time">{t('Time')}</label>
					<input id="h-time" type="time" bind:value={form.intention_time} />
				</div>
				<TextField
					id="h-place"
					label={t('Place')}
					bind:value={form.intention_place}
					maxlength={500}
				/>
			</div>
			<TextField
				id="h-two"
				label={t('Two-minute version')}
				hint={t('The smallest step that still counts.')}
				bind:value={form.two_minute}
				maxlength={500}
			/>
			<TextField
				id="h-stack"
				label={t('After…')}
				hint={t('An existing habit it follows.')}
				bind:value={form.stack_after}
				maxlength={500}
			/>
			<TextField
				id="h-cue"
				label={t('Cue')}
				hint={t('When and where.')}
				bind:value={form.cue}
				maxlength={500}
			/>
			<TextField
				id="h-craving"
				label={t('Why it matters')}
				bind:value={form.craving}
				maxlength={500}
			/>
			<TextField
				id="h-reward"
				label={t('Reward')}
				hint={t('Something satisfying right after.')}
				bind:value={form.reward}
				maxlength={500}
			/>
			<TextField
				id="h-partner"
				label={t('Accountability partner')}
				bind:value={form.contract_partner}
				maxlength={200}
			/>
			<TextField
				id="h-pledge"
				label={t('Pledge')}
				bind:value={form.contract_pledge}
				maxlength={2000}
			/>
			<fieldset class="days">
				<legend>{t('Days')}</legend>
				{#each weekdays as d (d.id)}
					<label
						><input type="checkbox" value={d.id} bind:group={form.days} />
						<span aria-hidden="true">{t(d.short)}</span><span class="visually-hidden"
							>{t(d.long)}</span
						></label
					>
				{/each}
			</fieldset>
			<div class="field">
				<label for="h-status">{t('Status')}</label>
				<select id="h-status" bind:value={form.status}>
					<option value="active">{t('Active')}</option>
					<option value="paused">{t('Paused (no prompts)')}</option>
					<option value="archived">{t('Archived')}</option>
				</select>
			</div>
			<div class="row">
				<Button type="submit" {busy} disabled={form.days.length === 0}>{t('Save')}</Button>
				<span class="grow"></span>
				<Button variant="danger" onclick={() => (confirmDelete = true)}>{t('Delete habit')}</Button>
			</div>
		</form>
	</section>
{/if}

<Dialog bind:open={confirmDelete} title={t('Delete this habit?')}>
	<p>
		{t(
			'“{title}” and its check-ins are removed. To keep the history, pause or archive it instead.',
			{ title: habit?.title }
		)}
	</p>
	<div class="row end">
		<Button variant="secondary" onclick={() => (confirmDelete = false)}>{t('Cancel')}</Button>
		<Button variant="danger" onclick={remove} {busy}>{t('Delete')}</Button>
	</div>
</Dialog>

<style>
	.crumbs {
		margin: 0 0 var(--space-2);
	}
	.crumbs a {
		color: var(--muted);
		text-decoration: none;
	}
	h1 {
		overflow-wrap: anywhere;
	}
	.small {
		font-size: var(--text-sm);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	@media (max-width: 30rem) {
		.card {
			padding: var(--space-4);
		}
	}
	h2 {
		font-size: var(--text-lg);
	}
	.grid {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(7, 1.5rem);
		gap: var(--space-1);
	}
	.cell {
		display: inline-block;
		width: 1.5rem;
		height: 1.5rem;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border);
		background: var(--surface);
		vertical-align: middle;
	}
	.cell.done {
		background: var(--accent);
		border-color: var(--accent);
	}
	.cell.two_minute {
		background: var(--accent-soft);
		border-color: var(--accent);
	}
	.cell.skipped {
		background: var(--bg);
		border-style: dashed;
	}
	.cell.off {
		opacity: 0.4;
	}
	.legend .cell {
		width: 0.9rem;
		height: 0.9rem;
		margin-left: var(--space-2);
	}
	.rows {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-3);
	}
	.grow {
		flex: 1;
		overflow-wrap: anywhere;
	}
	.inline,
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		align-items: center;
	}
	.end {
		justify-content: flex-end;
	}
	select {
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
		max-width: 28rem;
	}
	.field label,
	.days legend {
		font-weight: 600;
	}
	.days {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		border: none;
		padding: 0;
		margin: 0 0 var(--space-4);
	}
	.days legend {
		margin-bottom: var(--space-2);
	}
	.days label {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}
	.days input {
		width: 1.1rem;
		height: 1.1rem;
		accent-color: var(--accent);
	}
	.kind {
		border: 0;
		padding: 0;
		margin: 0 0 var(--space-3);
		display: flex;
		gap: var(--space-4);
	}
	.kind legend {
		font-weight: 600;
	}
	.pair {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr));
		gap: var(--space-3);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
	}
	.field label {
		font-weight: 600;
	}
	.field input {
		font: inherit;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
</style>
