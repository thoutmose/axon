<script lang="ts">
	import { api, message, type Schemas } from '$lib/api';
	import { i18n, t } from '$lib/i18n.svelte';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import { fromURL } from '$lib/practice';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// The practice log (EP-103): algorithm problems, wargame levels, and labs,
	// each attempt marked solved unaided, with help, or not solved, coming
	// back to be re-solved at growing intervals.
	type Problem = Schemas['PracticeProblem'];
	type Stats = {
		problems: number;
		solved: number;
		due: number;
		by_difficulty: Record<string, number>;
		by_platform: Record<string, number>;
		topics: { topic: string; solved: number; total: number }[];
		weeks: { week: string; solved: number }[];
		minutes: number;
	};
	let items = $state<Problem[] | null>(null);
	let stats = $state<Stats | null>(null);
	let today = $state('');
	let error = $state('');
	let busy = $state(false);

	const platforms = $derived([
		['leetcode', 'LeetCode'],
		['codeforces', 'Codeforces'],
		['hackerrank', 'HackerRank'],
		['advent-of-code', 'Advent of Code'],
		['exercism', 'Exercism'],
		['project-euler', 'Project Euler'],
		['overthewire', 'OverTheWire'],
		['portswigger', 'PortSwigger'],
		['hackthebox', 'Hack The Box'],
		['tryhackme', 'TryHackMe'],
		['kaggle', 'Kaggle'],
		['other', t('Other')]
	]);
	const outcomes = $derived([
		['solved', t('Solved unaided')],
		['with_help', t('Solved with help')],
		['not_solved', t('Not solved')]
	]);
	const difficulties = $derived([
		['easy', t('Easy')],
		['medium', t('Medium')],
		['hard', t('Hard')]
	]);
	const name = (pairs: string[][], id: string) => pairs.find((p) => p[0] === id)?.[1] ?? id;

	let form = $state({
		url: '',
		title: '',
		platform: 'leetcode',
		difficulty: 'medium',
		topics: '',
		language: 'python',
		outcome: 'solved',
		minutes: ''
	});
	function pasted() {
		const r = fromURL(form.url);
		if (r.platform) form.platform = r.platform;
		if (r.title && !form.title) form.title = r.title;
	}

	async function load() {
		error = '';
		try {
			const [l, s] = await Promise.all([
				api<{ items: Problem[]; today: string }>('/v1/practice'),
				api<Stats>('/v1/practice/stats')
			]);
			items = l.items;
			today = l.today;
			stats = s;
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function add(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			await api('/v1/practice', {
				method: 'POST',
				body: {
					title: form.title,
					url: form.url.trim() || undefined,
					platform: form.platform,
					difficulty: form.difficulty,
					topics: form.topics
						.split(',')
						.map((s) => s.trim())
						.filter(Boolean),
					language: form.language,
					outcome: form.outcome,
					minutes: form.minutes ? Number(form.minutes) : null
				}
			});
			toast(t('Logged.'));
			form = { ...form, url: '', title: '', topics: '', minutes: '' };
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	async function attempt(p: Problem, outcome: string) {
		try {
			await api(`/v1/practice/${p.id}/attempts`, { method: 'POST', body: { outcome } });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	async function remove(p: Problem) {
		if (!confirm(t('Remove “{title}” and its attempts?', { title: p.title }))) return;
		try {
			await api(`/v1/practice/${p.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const due = $derived((items ?? []).filter((p) => p.due_on && p.due_on <= today));
	const weekMax = $derived(Math.max(1, ...(stats?.weeks ?? []).map((w) => w.solved)));
	const last = (p: Problem) => p.attempts[p.attempts.length - 1];
	const hours = $derived(
		stats ? (stats.minutes / 60).toLocaleString(i18n.locale, { maximumFractionDigits: 1 }) : '0'
	);
</script>

<svelte:head><title>{t('Practice')} · Axon</title></svelte:head>

<h1>{t('Practice')}</h1>
<LearnTabs />
<p class="muted hint">
	{t(
		'Problems, wargame levels, and labs. Mark each attempt honestly: a problem solved with help or not at all comes back sooner, and one solved unaided comes back at growing intervals until you know it.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !items || !stats}
	<Status kind="loading" message={t('Loading…')} />
{:else}
	<section class="card" aria-labelledby="log-title">
		<h2 id="log-title">{t('Log a problem')}</h2>
		<form class="grid" onsubmit={add}>
			<label class="wide"
				>{t('Link')}
				<input
					type="url"
					bind:value={form.url}
					oninput={pasted}
					placeholder="https://leetcode.com/problems/two-sum/"
				/></label
			>
			<label class="wide"
				>{t('Title')} <input bind:value={form.title} required maxlength="300" /></label
			>
			<label
				>{t('Platform')}
				<select bind:value={form.platform}>
					{#each platforms as [v, l] (v)}<option value={v}>{l}</option>{/each}
				</select></label
			>
			<label
				>{t('Difficulty')}
				<select bind:value={form.difficulty}>
					{#each difficulties as [v, l] (v)}<option value={v}>{l}</option>{/each}
				</select></label
			>
			<label
				>{t('Topics, separated by commas')}
				<input bind:value={form.topics} placeholder={t('arrays, hash table')} /></label
			>
			<label>{t('Language')} <input bind:value={form.language} maxlength="20" /></label>
			<label
				>{t('How it went')}
				<select bind:value={form.outcome}>
					{#each outcomes as [v, l] (v)}<option value={v}>{l}</option>{/each}
				</select></label
			>
			<label>{t('Minutes')} <input inputmode="numeric" bind:value={form.minutes} /></label>
			<div class="wide">
				<Button type="submit" {busy} disabled={!form.title.trim()}>{t('Log it')}</Button>
			</div>
		</form>
	</section>

	<section class="card" aria-labelledby="due-title">
		<h2 id="due-title">{t('Re-solve today')}</h2>
		{#if due.length}
			<ul class="due">
				{#each due as p (p.id)}
					<li>
						<span class="grow"
							>{#if p.url}<a href={p.url} rel="noopener noreferrer" target="_blank">{p.title}</a
								>{:else}{p.title}{/if}
							<span class="muted small"
								>· {name(difficulties, p.difficulty)} · {t('last: {outcome}', {
									outcome: name(outcomes, last(p).outcome)
								})}</span
							></span
						>
						{#each outcomes as [v, l] (v)}
							<Button
								variant="secondary"
								onclick={() => attempt(p, v)}
								aria-label={t('{outcome}: {title}', { outcome: l, title: p.title })}>{l}</Button
							>
						{/each}
					</li>
				{/each}
			</ul>
		{:else}
			<p class="muted">{t('Nothing to re-solve today.')}</p>
		{/if}
	</section>

	<section class="card" aria-labelledby="stats-title">
		<h2 id="stats-title">{t('Your practice')}</h2>
		<p class="kpis">
			<span><strong>{stats.solved}</strong> {t('solved of {n}', { n: stats.problems })}</span>
			<span><strong>{stats.by_difficulty.easy ?? 0}</strong> {t('easy')}</span>
			<span><strong>{stats.by_difficulty.medium ?? 0}</strong> {t('medium')}</span>
			<span><strong>{stats.by_difficulty.hard ?? 0}</strong> {t('hard')}</span>
			<span><strong>{hours}</strong> h</span>
		</p>
		<figure>
			<figcaption class="small muted">
				{t('First solved unaided, by week (the last 12 weeks)')}
			</figcaption>
			<ol class="weeks">
				{#each stats.weeks as w (w.week)}
					<li>
						<span
							class="bar"
							style:height="{Math.round((w.solved / weekMax) * 56)}px"
							aria-hidden="true"
						></span>
						<span class="small">{w.week.slice(5)}</span>
						<span class="visually-hidden"
							>{t('Week of {week}: {n} solved', { week: w.week, n: w.solved })}</span
						>
					</li>
				{/each}
			</ol>
		</figure>
		{#if stats.topics.length}
			<table>
				<caption class="visually-hidden">{t('By topic')}</caption>
				<thead
					><tr
						><th scope="col">{t('Topic')}</th><th scope="col">{t('Solved')}</th><th scope="col"
							>{t('Problems')}</th
						></tr
					></thead
				>
				<tbody>
					{#each stats.topics.slice(0, 12) as c (c.topic)}
						<tr><th scope="row">{c.topic}</th><td>{c.solved}</td><td>{c.total}</td></tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>

	<section class="card" aria-labelledby="all-title">
		<h2 id="all-title">{t('All problems')}</h2>
		{#if items.length}
			<div class="scroll">
				<table>
					<thead
						><tr
							><th scope="col">{t('Problem')}</th><th scope="col">{t('Platform')}</th><th
								scope="col">{t('Difficulty')}</th
							><th scope="col">{t('Attempts')}</th><th scope="col">{t('Next re-solve')}</th><th
								scope="col"><span class="visually-hidden">{t('Action')}</span></th
							></tr
						></thead
					>
					<tbody>
						{#each items as p (p.id)}
							<tr>
								<th scope="row"
									>{#if p.url}<a href={p.url} rel="noopener noreferrer" target="_blank">{p.title}</a
										>{:else}{p.title}{/if}
									{#if p.topics.length}<span class="small muted">· {p.topics.join(', ')}</span
										>{/if}</th
								>
								<td>{name(platforms, p.platform)}</td>
								<td>{name(difficulties, p.difficulty)}</td>
								<td>{p.attempts.length} · {name(outcomes, last(p).outcome)}</td>
								<td>{p.due_on ?? t('known')}</td>
								<td
									><button
										type="button"
										class="link"
										onclick={() => remove(p)}
										aria-label={t('Remove {title}', { title: p.title })}>{t('Remove')}</button
									></td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<Status kind="empty" message={t('No problems yet. Paste a link above to log the first.')} />
		{/if}
	</section>
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
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr));
		gap: var(--space-3);
	}
	.grid label {
		display: grid;
		gap: var(--space-1);
	}
	.wide {
		grid-column: 1 / -1;
	}
	input,
	select {
		font: inherit;
		padding: var(--space-2);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--fg);
	}
	.due {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: var(--space-2);
	}
	.due li {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
	}
	.grow {
		flex: 1;
		min-width: 12rem;
	}
	.kpis {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-4);
	}
	.weeks {
		list-style: none;
		padding: 0;
		margin: var(--space-2) 0 var(--space-4);
		display: grid;
		grid-template-columns: repeat(12, 1fr);
		gap: var(--space-1);
		align-items: end;
		max-width: 36rem;
	}
	.weeks li {
		display: grid;
		justify-items: center;
		gap: 2px;
	}
	.bar {
		width: 70%;
		min-height: 2px;
		background: var(--accent);
		border-radius: 3px 3px 0 0;
	}
	.scroll {
		overflow-x: auto;
	}
	table {
		border-collapse: collapse;
		width: 100%;
		font-size: var(--text-sm);
	}
	th,
	td {
		text-align: left;
		padding: var(--space-1) var(--space-2);
		border-bottom: 1px solid var(--border);
		vertical-align: top;
	}
	.link {
		background: none;
		border: 0;
		color: var(--accent);
		cursor: pointer;
		font: inherit;
		padding: 0;
	}
</style>
