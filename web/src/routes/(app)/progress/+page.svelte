<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { api, message } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import Collection from '$lib/Collection.svelte';
	import { turnOnRewards } from '$lib/collection';
	import ShareButton from '$lib/ShareButton.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Progress = paths['/v1/progress']['get']['responses'][200]['content']['application/json'];
	type Rules = paths['/v1/progress/rules']['get']['responses'][200]['content']['application/json'];
	type Entry = { at: string; rule: string; points: number; entity_id: string; title: string };

	let p = $state<Progress | null>(null);
	let rules = $state<Rules | null>(null);
	let ledger = $state<Entry[]>([]);
	let error = $state('');

	async function load() {
		error = '';
		try {
			[p, rules] = await Promise.all([
				api<Progress>('/v1/progress'),
				api<Rules>('/v1/progress/rules')
			]);
			ledger = p.enabled
				? (await api<{ items: Entry[] }>('/v1/progress/ledger?days=30')).items
				: [];
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function turnOn() {
		try {
			await turnOnRewards();
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function buy() {
		try {
			await api('/v1/progress/freezes', { method: 'POST' });
			toast(t('Freeze ready. It covers a missed day automatically.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const ruleText = $derived(
		Object.fromEntries((rules?.items ?? []).map((r) => [r.id, r.description]))
	);
	const stateNames: Record<string, string> = $derived({
		active: t('Today counts'),
		at_risk: t('Still open today'),
		frozen: t('Covered by a freeze'),
		broken: t('Fresh start'),
		inactive: t('Not started')
	});
	const pct = $derived(
		p?.level_to && p.level_from !== undefined && p.xp !== undefined
			? Math.round(((p.xp - p.level_from) / (p.level_to - p.level_from)) * 100)
			: 0
	);
</script>

<svelte:head><title>{t('Progress · Axon')}</title></svelte:head>

<h1>{t('Progress')}</h1>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !p}
	<Status kind="loading" message={t('Loading progress…')} />
{:else if !p.enabled}
	<section class="card glass">
		<h2>{t('Progress rewards are off')}</h2>
		<p>
			{t(
				'When on, everything you do — tasks, habits, focus sessions, flashcards, notes, skills, goals, and projects — earns points you can trace one by one, levels, badges, and one daily streak. There are no notifications, no shame for missed days, and nothing you can buy with money.'
			)}
		</p>
		<Button onclick={turnOn}>{t('Turn on progress rewards')}</Button>
	</section>
{:else}
	<div class="top">
		<section class="card glass level" aria-labelledby="level-title">
			<h2 id="level-title">{t('Level {level}', { level: p.level })}</h2>
			<div
				class="bar"
				role="progressbar"
				aria-label={t('Progress to level {value}', { value: (p.level ?? 1) + 1 })}
				aria-valuemin="0"
				aria-valuemax="100"
				aria-valuenow={pct}
			>
				<span class="fill w{Math.min(100, Math.max(0, Math.round(pct / 5) * 5))}"></span>
			</div>
			<p class="muted small">
				{t('{xp} points · {value} to level {value2} · {today_xp} today', {
					xp: p.xp,
					value: (p.level_to ?? 0) - (p.xp ?? 0),
					value2: (p.level ?? 1) + 1,
					today_xp: p.today_xp
				})}
			</p>
			<ShareButton
				spec={{
					title: t('My progress on Axon'),
					value: t('Level {level}', { level: p.level }),
					caption: t('{xp} points, and a run of {days} days.', {
						xp: (p.xp ?? 0).toLocaleString(appLocale()),
						days: p.streak?.days ?? 0
					}),
					kind: 'accent'
				}}
				text={t('Level {level} on Axon, with a run of {days} days. #Axon', {
					level: p.level,
					days: p.streak?.days ?? 0
				})}
			/>
		</section>
		<section class="card glass" aria-labelledby="streak-title">
			<h2 id="streak-title">{t('{value}-day run', { value: p.streak?.days ?? 0 })}</h2>
			<p class="state {p.streak?.state}">{stateNames[p.streak?.state ?? 'inactive']}</p>
			<p class="muted small">{p.message}</p>
			<p class="small">
				{t('{value} of {max_freezes} freezes ready · {points} points to spend', {
					value: p.streak?.freezes ?? 0,
					max_freezes: rules?.max_freezes,
					points: p.points
				})}
			</p>
			<Button
				variant="secondary"
				onclick={buy}
				disabled={(p.points ?? 0) < (rules?.freeze_cost ?? 100) ||
					(p.streak?.freezes ?? 0) >= (rules?.max_freezes ?? 2)}
				>{t('Buy a freeze for {freeze_cost} points', { freeze_cost: rules?.freeze_cost })}</Button
			>
		</section>
	</div>

	<section class="card glass" aria-labelledby="badges-title">
		<h2 id="badges-title">{t('Badges')}</h2>
		<ul class="badges">
			{#each p.badges ?? [] as b (b.id)}
				<li class:earned={b.earned_at}>
					<strong>{b.title}</strong>
					<span class="small">{b.description}</span>
					<span class="muted small"
						>{b.earned_at
							? t('Earned {date}', { date: new Date(b.earned_at).toLocaleDateString(appLocale()) })
							: t('Not yet')}</span
					>
				</li>
			{/each}
		</ul>
	</section>

	<Collection />

	<div class="top">
		<section class="card glass" aria-labelledby="ledger-title">
			<h2 id="ledger-title">{t('How you earned your points (30 days)')}</h2>
			{#if ledger.length}
				<table>
					<thead
						><tr
							><th scope="col">{t('When')}</th><th scope="col">{t('What')}</th><th scope="col"
								>{t('Points')}</th
							></tr
						></thead
					>
					<tbody>
						{#each ledger as e, i (i)}
							<tr>
								<td
									>{new Date(e.at).toLocaleDateString(appLocale(), {
										day: 'numeric',
										month: 'short'
									})}</td
								>
								<td>{ruleText[e.rule] ?? e.rule}: <span class="muted">{e.title}</span></td>
								<td>+{e.points}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{:else}
				<p class="muted">{t('No points yet in the last 30 days.')}</p>
			{/if}
		</section>
		<section class="card glass" aria-labelledby="rules-title">
			<h2 id="rules-title">{t('Every way to earn points')}</h2>
			<table>
				<tbody>
					{#each rules?.items ?? [] as r (r.id)}
						<tr><td>{r.description}</td><td>+{r.points}</td></tr>
					{/each}
				</tbody>
			</table>
			<p class="muted small">
				{t(
					'Levels: level 2 at 100 points, 3 at 300, 4 at 600, 5 at 1,000, and so on. Turn rewards off any time in Preferences.'
				)}
			</p>
		</section>
	</div>
{/if}

<style>
	.small {
		font-size: var(--text-sm);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
		min-width: 0;
	}
	.top {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 20rem), 1fr));
		gap: var(--space-4);
	}
	.bar {
		height: 0.75rem;
		border-radius: 999px;
		background: var(--accent-soft);
		overflow: hidden;
	}
	.fill {
		display: block;
		height: 100%;
		background: var(--accent);
	}
	.fill.w0 {
		width: 0%;
	}
	.fill.w5 {
		width: 5%;
	}
	.fill.w10 {
		width: 10%;
	}
	.fill.w15 {
		width: 15%;
	}
	.fill.w20 {
		width: 20%;
	}
	.fill.w25 {
		width: 25%;
	}
	.fill.w30 {
		width: 30%;
	}
	.fill.w35 {
		width: 35%;
	}
	.fill.w40 {
		width: 40%;
	}
	.fill.w45 {
		width: 45%;
	}
	.fill.w50 {
		width: 50%;
	}
	.fill.w55 {
		width: 55%;
	}
	.fill.w60 {
		width: 60%;
	}
	.fill.w65 {
		width: 65%;
	}
	.fill.w70 {
		width: 70%;
	}
	.fill.w75 {
		width: 75%;
	}
	.fill.w80 {
		width: 80%;
	}
	.fill.w85 {
		width: 85%;
	}
	.fill.w90 {
		width: 90%;
	}
	.fill.w95 {
		width: 95%;
	}
	.fill.w100 {
		width: 100%;
	}
	.state {
		font-weight: 650;
	}
	.state.active {
		color: var(--ok);
	}
	.badges {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 13rem), 1fr));
		gap: var(--space-2);
	}
	.badges li {
		display: grid;
		gap: var(--space-1);
		padding: var(--space-3);
		border: 1px dashed var(--border);
		border-radius: var(--radius);
	}
	.badges li.earned {
		border-style: solid;
		border-color: var(--accent);
		background: var(--accent-soft);
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--text-sm);
	}
	td,
	th {
		text-align: left;
		padding: var(--space-1) var(--space-2);
		border-bottom: 1px solid var(--border);
	}
</style>
