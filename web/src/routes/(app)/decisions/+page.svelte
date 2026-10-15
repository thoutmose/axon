<script lang="ts">
	import { api, message, type Schemas } from '$lib/api';
	import { addMonths, confidence, confidences, plot, verdict, verdicts } from '$lib/decisions';
	import { i18n, t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// The decision journal (EP-114): write down a decision, what you expect
	// and how likely it is; on its day, say what happened. Axon scores the
	// forecasts, so you learn how far to trust your own confidence.
	type Decision = Schemas['Decision'];
	type Calibration = {
		count: number;
		brier: number | null;
		predicted: number;
		happened: number;
		buckets: { from: number; count: number; predicted: number; happened: number }[];
	};
	let items = $state<Decision[] | null>(null);
	let cal = $state<Calibration | null>(null);
	let today = $state('');
	let error = $state('');
	let busy = $state(false);

	const blank = () => ({
		title: '',
		context: '',
		choice: '',
		expected: '',
		chance: 70,
		months: 3,
		review_on: ''
	});
	let form = $state(blank());
	const reviewOn = $derived(form.review_on || (today ? addMonths(today, form.months) : ''));

	// The review being written, by decision.
	let reviewing = $state<Record<string, { outcome: string; lessons: string }>>({});

	async function load() {
		error = '';
		try {
			const [l, c] = await Promise.all([
				api<{ items: Decision[]; today: string }>('/v1/decisions'),
				api<Calibration>('/v1/decisions/calibration')
			]);
			items = l.items;
			today = l.today;
			cal = c;
			for (const d of l.items) if (d.due) reviewing[d.id] ??= { outcome: '', lessons: '' };
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function add(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			await api('/v1/decisions', {
				method: 'POST',
				body: {
					title: form.title,
					context: form.context,
					choice: form.choice,
					expected: form.expected,
					probability: form.chance / 100,
					review_on: reviewOn
				}
			});
			toast(t('Recorded. Axon will ask you on {day}.', { day: reviewOn }));
			form = blank();
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function review(d: Decision, happened: boolean) {
		const r = reviewing[d.id] ?? { outcome: '', lessons: '' };
		try {
			await api(`/v1/decisions/${d.id}/review`, {
				method: 'POST',
				body: { happened, outcome: r.outcome, lessons: r.lessons }
			});
			delete reviewing[d.id];
			toast(t('Reviewed.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove(d: Decision) {
		if (!confirm(t('Remove “{title}”?', { title: d.title }))) return;
		try {
			await api(`/v1/decisions/${d.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const due = $derived((items ?? []).filter((d) => d.due));
	const open = $derived((items ?? []).filter((d) => d.happened === null && !d.due));
	const reviewed = $derived((items ?? []).filter((d) => d.happened !== null));
	const pct = (p: number) => p.toLocaleString(i18n.locale, { style: 'percent' });
	const score = (b: number) => b.toLocaleString(i18n.locale, { maximumFractionDigits: 2 });

	const SIZE = 260;
	const PAD = 30;
	const dots = $derived(cal ? plot(cal.buckets, SIZE, PAD) : []);
	const leaning = $derived(cal ? confidence(cal) : null);
</script>

<svelte:head><title>{t('Decisions')} · Axon</title></svelte:head>

<h1>{t('Decisions')}</h1>
<p class="muted hint">
	{t(
		'Write a decision down when you make it: what you chose, what you expect, and how likely it is. On the review day, say what happened. The forecast cannot be changed afterwards, so your score tells you how far to trust your own confidence.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !items || !cal}
	<Status kind="loading" message={t('Loading…')} />
{:else}
	{#if due.length}
		<section class="card" aria-labelledby="due-title">
			<h2 id="due-title">{t('To review')}</h2>
			<ul class="list">
				{#each due as d (d.id)}
					<li>
						<h3>{d.title}</h3>
						<p>
							{t('You expected: {expected}', { expected: d.expected })}
							<span class="muted small"
								>· {t('{chance} sure, on {day}', {
									chance: pct(d.probability),
									day: d.decided_on
								})}</span
							>
						</p>
						{#if d.choice}<p class="small">{t('You chose: {choice}', { choice: d.choice })}</p>{/if}
						<label
							>{t('What happened')}
							<textarea rows="2" bind:value={reviewing[d.id].outcome} maxlength="5000"
							></textarea></label
						>
						<label
							>{t('What you learned')}
							<textarea rows="2" bind:value={reviewing[d.id].lessons} maxlength="5000"
							></textarea></label
						>
						<div class="row">
							<Button onclick={() => review(d, true)}>{t('It happened')}</Button>
							<Button variant="secondary" onclick={() => review(d, false)}
								>{t('It did not happen')}</Button
							>
						</div>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	<section class="card" aria-labelledby="new-title">
		<h2 id="new-title">{t('Record a decision')}</h2>
		<form class="grid" onsubmit={add}>
			<label class="wide"
				>{t('Decision')}
				<input
					bind:value={form.title}
					required
					maxlength="200"
					placeholder={t('Move the API to Go')}
				/></label
			>
			<label class="wide"
				>{t('Context: what you know, and the options')}
				<textarea rows="2" bind:value={form.context} maxlength="5000"></textarea></label
			>
			<label class="wide"
				>{t('What you chose')}
				<input bind:value={form.choice} maxlength="2000" /></label
			>
			<label class="wide"
				>{t('What you expect to happen')}
				<input
					bind:value={form.expected}
					required
					maxlength="500"
					placeholder={t('p99 latency under 100 ms by March')}
				/></label
			>
			<label
				>{t('How likely: {chance}', { chance: pct(form.chance / 100) })}
				<input type="range" min="5" max="95" step="5" bind:value={form.chance} /></label
			>
			<label
				>{t('Review in')}
				<select bind:value={form.months} onchange={() => (form.review_on = '')}>
					<option value={1}>{t('1 month')}</option>
					<option value={3}>{t('3 months')}</option>
					<option value={6}>{t('6 months')}</option>
					<option value={12}>{t('1 year')}</option>
				</select></label
			>
			<label>{t('Or on')} <input type="date" bind:value={form.review_on} min={today} /></label>
			<div class="wide">
				<Button type="submit" {busy} disabled={!form.title.trim() || !form.expected.trim()}
					>{t('Record it')}</Button
				>
			</div>
		</form>
	</section>

	<section class="card" aria-labelledby="cal-title">
		<h2 id="cal-title">{t('Your calibration')}</h2>
		{#if cal.count === 0 || cal.brier === null}
			<p class="muted">
				{t('Review your first decision to see how well your forecasts match what happens.')}
			</p>
		{:else}
			<p class="kpis">
				<span><strong>{score(cal.brier)}</strong> {t('Brier score')}</span>
				<span><strong>{pct(cal.predicted)}</strong> {t('chance given, on average')}</span>
				<span><strong>{pct(cal.happened)}</strong> {t('came true')}</span>
				<span><strong>{cal.count}</strong> {t('reviewed')}</span>
			</p>
			<p>{t(verdicts[verdict(cal.brier)])}</p>
			{#if leaning}<p>{t(confidences[leaning])}</p>{/if}
			<p class="muted small">
				{t(
					'The Brier score is the mean of (chance − outcome)²: 0 is perfect, and always saying 50% scores 0.25.'
				)}
			</p>
			<figure>
				<svg
					viewBox="0 0 {SIZE} {SIZE}"
					width={SIZE}
					height={SIZE}
					role="img"
					aria-labelledby="cal-chart-title"
				>
					<title id="cal-chart-title"
						>{t('Chance given (across) against the share that came true (up)')}</title
					>
					<rect x={PAD} y={PAD} width={SIZE - 2 * PAD} height={SIZE - 2 * PAD} class="frame" />
					<line x1={PAD} y1={SIZE - PAD} x2={SIZE - PAD} y2={PAD} class="ideal" />
					{#each dots as b (b.from)}
						<circle cx={b.x} cy={SIZE - b.y} r={b.r} class="dot">
							<title
								>{t('{from}–{to} sure: {n} decisions, {share} came true', {
									from: pct(b.from),
									to: pct(b.from + 0.1),
									n: b.count,
									share: pct(b.happened)
								})}</title
							>
						</circle>
					{/each}
					<text x={SIZE / 2} y={SIZE - 6} text-anchor="middle" class="axis"
						>{t('chance given')}</text
					>
					<text
						x="12"
						y={SIZE / 2}
						text-anchor="middle"
						class="axis"
						transform="rotate(-90 12 {SIZE / 2})">{t('came true')}</text
					>
				</svg>
				<figcaption class="small muted">
					{t('On the dashed line, things happen as often as you say. Below it, you were too sure.')}
				</figcaption>
			</figure>
			<table>
				<caption class="visually-hidden">{t('Your calibration')}</caption>
				<thead
					><tr
						><th scope="col">{t('Chance given')}</th><th scope="col">{t('Decisions')}</th><th
							scope="col">{t('Came true')}</th
						></tr
					></thead
				>
				<tbody>
					{#each cal.buckets as b (b.from)}
						<tr
							><th scope="row">{pct(b.from)}–{pct(b.from + 0.1)}</th><td>{b.count}</td><td
								>{pct(b.happened)}</td
							></tr
						>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>

	<section class="card" aria-labelledby="open-title">
		<h2 id="open-title">{t('Waiting for their day')}</h2>
		{#if open.length}
			<ul class="list">
				{#each open as d (d.id)}
					<li class="line">
						<span class="grow"
							><strong>{d.title}</strong>
							<span class="muted small"
								>· {d.expected} · {pct(d.probability)} · {t('review on {day}', {
									day: d.review_on
								})}</span
							></span
						>
						<button
							type="button"
							class="link"
							onclick={() => remove(d)}
							aria-label={t('Remove {title}', { title: d.title })}>{t('Remove')}</button
						>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="muted">{t('Nothing waiting. Record a decision above.')}</p>
		{/if}
	</section>

	<section class="card" aria-labelledby="done-title">
		<h2 id="done-title">{t('Reviewed')}</h2>
		{#if reviewed.length}
			<ul class="list">
				{#each reviewed as d (d.id)}
					<li>
						<p class="line">
							<span class="grow"
								><strong>{d.title}</strong>
								<span class="muted small"
									>· {pct(d.probability)} · {d.happened
										? t('it happened')
										: t('it did not happen')}</span
								></span
							>
							<button
								type="button"
								class="link"
								onclick={() => remove(d)}
								aria-label={t('Remove {title}', { title: d.title })}>{t('Remove')}</button
							>
						</p>
						<p class="small">{t('You expected: {expected}', { expected: d.expected })}</p>
						{#if d.outcome}<p class="small">{d.outcome}</p>{/if}
						{#if d.lessons}<p class="small lesson">{d.lessons}</p>{/if}
					</li>
				{/each}
			</ul>
		{:else}
			<p class="muted">{t('No decision reviewed yet.')}</p>
		{/if}
	</section>
{/if}

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.link {
		background: none;
		border: 0;
		color: var(--accent);
		cursor: pointer;
		font: inherit;
		padding: 0;
	}
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		gap: var(--space-3);
	}
	.grid label {
		display: grid;
		gap: var(--space-1);
	}
	.wide {
		grid-column: 1 / -1;
	}
	.list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: var(--space-4);
	}
	.list > li + li {
		border-top: 1px solid var(--border);
		padding-top: var(--space-3);
	}
	.list label {
		display: grid;
		gap: var(--space-1);
		margin-top: var(--space-2);
	}
	h3 {
		margin: 0;
		font-size: var(--text-lg);
	}
	.row,
	.line {
		display: flex;
		gap: var(--space-2);
		align-items: baseline;
		flex-wrap: wrap;
		margin-top: var(--space-2);
	}
	.grow {
		flex: 1;
	}
	.kpis {
		display: flex;
		gap: var(--space-4);
		flex-wrap: wrap;
	}
	.lesson {
		border-left: 3px solid var(--accent);
		padding-left: var(--space-2);
	}
	figure {
		margin: var(--space-3) 0;
	}
	svg {
		max-width: 100%;
		height: auto;
	}
	.frame {
		fill: none;
		stroke: var(--border);
	}
	.ideal {
		stroke: var(--muted);
		stroke-width: 2;
		stroke-dasharray: 6 5;
	}
	.dot {
		fill: var(--accent);
		fill-opacity: 0.8;
		stroke: var(--surface);
		stroke-width: 2;
	}
	.axis {
		fill: var(--muted);
		font-size: 11px;
	}
	table {
		border-collapse: collapse;
	}
	th,
	td {
		padding: var(--space-1) var(--space-3);
		text-align: left;
	}
</style>
