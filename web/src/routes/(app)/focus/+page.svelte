<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { i18n, t } from '$lib/i18n.svelte';
	import { typeNames } from '$lib/links';
	import {
		advance,
		clock,
		defaults,
		finished,
		newTimer,
		pause,
		phases,
		remaining,
		resume,
		rounds,
		skip,
		totalWorkMinutes,
		workedMinutes,
		type FocusConfig,
		type Timer
	} from '$lib/pomodoro';
	import { styleFor } from '$lib/music';
	import { audio, chime, play, soundOptions, type Sound } from '$lib/sounds';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Session = Schemas['FocusSession'];
	type Hit = { id: string; type: string; title: string };
	type Settings = {
		focus: FocusConfig & {
			sound: Sound;
			volume: number;
			music: { activity: string; url: string }[];
		};
	};

	let config = $state<Settings['focus']>({ ...defaults, sound: 'none', volume: 40, music: [] });
	let timer = $state<Timer | null>(null);
	let session = $state<Session | null>(null);
	let now = $state(Date.now());
	let target = $state<Hit | null>(null);
	let title = $state('');
	let query = $state('');
	let hits = $state<Hit[]>([]);
	let history = $state<{
		items: Session[];
		today_minutes: number;
		week_minutes: number;
		streak: number;
	} | null>(null);
	let error = $state('');
	let announce = $state('');
	let sound = $state<Sound>('none');
	let stopSound: () => void = () => {};
	let interval: ReturnType<typeof setInterval> | undefined;
	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	const key = 'axon-focus';

	const phaseNames: Record<string, string> = {
		work: 'Focus',
		short: 'Short break',
		long: 'Long break'
	};
	const name = (kind: string) => t(phaseNames[kind]);
	const plan = $derived(phases(config));
	const phase = $derived(timer && !finished(timer) ? timer.phases[timer.index] : null);
	const left = $derived(timer ? remaining(timer, now) : 0);
	const progress = $derived(phase ? 1 - left / (phase.minutes * 60_000) : 0);
	const soundNames: Record<string, string> = Object.fromEntries(
		soundOptions.flatMap((g) => g.items)
	);
	// What "matched" music plays: the style for the type of what the session is about.
	const playing = $derived(
		sound === 'music' ? soundNames[styleFor(session?.target?.type)] : soundNames[sound]
	);

	async function loadHistory() {
		try {
			history = await api('/v1/focus-sessions');
		} catch (err) {
			error = message(err);
		}
	}

	onMount(() => {
		api<Settings>('/v1/me/settings')
			.then((s) => {
				config = s.focus;
				sound = s.focus.sound;
			})
			.catch(() => {});
		loadHistory();
		// Resume a session that was running when the page was left.
		try {
			const saved = JSON.parse(localStorage.getItem(key) ?? 'null');
			if (saved?.timer && saved?.session) {
				timer = saved.timer;
				session = saved.session;
				tick();
				interval = setInterval(tick, 500);
			}
		} catch {
			localStorage.removeItem(key);
		}
	});
	onDestroy(() => {
		clearInterval(interval);
		stopSound();
		if (typeof document !== 'undefined') document.title = `${t('Focus')} · Axon`;
	});

	function save() {
		try {
			if (timer && session) localStorage.setItem(key, JSON.stringify({ timer, session }));
			else localStorage.removeItem(key);
		} catch {
			// Private browsing: the session lasts for this page only.
		}
	}

	function tick() {
		if (!timer) return;
		now = Date.now();
		const ended = advance(timer, now);
		if (ended.length) {
			chime(config.volume);
			const next = !finished(timer) ? timer.phases[timer.index] : null;
			announce = next
				? t('{phase} over. {next} for {minutes} minutes.', {
						phase: name(ended[ended.length - 1].kind),
						next: name(next.kind),
						minutes: next.minutes
					})
				: t('All rounds done. Well worked.');
			syncSound();
			save();
		}
		document.title = `${clock(remaining(timer, now))} · ${name(phase?.kind ?? 'work')} · Axon`;
		if (finished(timer)) finish('completed');
	}

	function syncSound() {
		stopSound();
		stopSound =
			timer && !timer.paused && phase?.kind === 'work'
				? play(sound, config.volume, session?.target?.type)
				: () => {};
	}

	function search() {
		clearTimeout(searchTimer);
		const q = query.trim();
		if (!q) {
			hits = [];
			return;
		}
		searchTimer = setTimeout(async () => {
			try {
				const res = await api<{ items: Hit[] }>(`/v1/search?q=${encodeURIComponent(q)}&limit=6`);
				hits = res.items.filter((h) => typeNames[h.type]);
			} catch {
				hits = [];
			}
		}, 200);
	}

	async function start() {
		audio(); // while the click still counts, so the sound may start after the request
		try {
			session = await api<Session>('/v1/focus-sessions', {
				method: 'POST',
				body: {
					planned_minutes: totalWorkMinutes(plan),
					...(target ? { target_id: target.id } : { title: title.trim() || 'Focus' })
				}
			});
			timer = newTimer(plan, Date.now());
			announce = t('Focus for {minutes} minutes.', { minutes: plan[0].minutes });
			interval = setInterval(tick, 500);
			syncSound();
			save();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function togglePause() {
		if (!timer) return;
		if (timer.paused) resume(timer, Date.now());
		else pause(timer, Date.now());
		timer = timer;
		syncSound();
		save();
	}

	function next() {
		if (!timer) return;
		skip(timer, Date.now());
		timer = timer;
		syncSound();
		save();
		tick();
	}

	async function finish(status: 'completed' | 'stopped') {
		if (!timer || !session) return;
		const t0 = timer;
		const s0 = session;
		clearInterval(interval);
		stopSound();
		stopSound = () => {};
		timer = null;
		session = null;
		save();
		document.title = `${t('Focus')} · Axon`;
		try {
			await api(`/v1/focus-sessions/${s0.id}/finish`, {
				method: 'POST',
				body: { worked_minutes: workedMinutes(t0, Date.now()), rounds: rounds(t0), status }
			});
			toast(t(status === 'completed' ? 'Session complete.' : 'Session saved.'));
		} catch (err) {
			toast(message(err), 'error');
		}
		await loadHistory();
	}

	const R = 90;
	const C = 2 * Math.PI * R;
</script>

{#snippet soundChoices()}
	<option value="none">{t('No sound')}</option>
	{#each soundOptions as g (g.group)}
		<optgroup label={t(g.group)}>
			{#each g.items as [value, label] (value)}
				<option {value}>{t(label)}</option>
			{/each}
		</optgroup>
	{/each}
{/snippet}

<svelte:head><title>{t('Focus')} · Axon</title></svelte:head>

<h1>{t('Focus')}</h1>
<p class="muted hint">
	{t(
		config.sessions > 1
			? 'Pomodoro sessions: {work} min of focus and {short} min breaks, a {long} min break after {rounds} rounds, {sessions} times.'
			: 'Pomodoro sessions: {work} min of focus and {short} min breaks, a {long} min break after {rounds} rounds.',
		{
			work: config.work_minutes,
			short: config.short_break_minutes,
			long: config.long_break_minutes,
			rounds: config.rounds,
			sessions: config.sessions
		}
	)}
	{t('Change it in')} <a href="/account/preferences">{t('Preferences')}</a>.
</p>

<p class="visually-hidden" aria-live="assertive">{announce}</p>

{#if timer && session}
	<section class="timer glass" aria-labelledby="phase-title" class:break={phase?.kind !== 'work'}>
		<h2 id="phase-title">{phase ? name(phase.kind) : t('Done')}</h2>
		<p class="for">
			{session.target
				? `${t(typeNames[session.target.type] ?? '')}: ${session.target.title}`
				: session.title}
		</p>
		<svg
			viewBox="0 0 220 220"
			role="img"
			aria-label={phase
				? t('{time} left in {phase}', { time: clock(left), phase: name(phase.kind) })
				: t('{time} left in the session', { time: clock(left) })}
		>
			<circle cx="110" cy="110" r={R} class="track" />
			<circle
				cx="110"
				cy="110"
				r={R}
				class="arc"
				stroke-dasharray={C}
				stroke-dashoffset={C * (1 - progress)}
				transform="rotate(-90 110 110)"
			/>
			<text x="110" y="118" text-anchor="middle" class="time">{clock(left)}</text>
		</svg>
		<ol class="phases" aria-label={t('Plan')}>
			{#each timer.phases as p, i (i)}
				<li
					class={p.kind}
					class:done={i < timer.index}
					class:now={i === timer.index}
					aria-current={i === timer.index ? 'step' : undefined}
				>
					<span class="visually-hidden">{name(p.kind)}, </span>{p.minutes}′
				</li>
			{/each}
		</ol>
		<div class="row">
			<Button onclick={togglePause}>{t(timer.paused ? 'Resume' : 'Pause')}</Button>
			<Button variant="secondary" onclick={next}>{t('Skip to next')}</Button>
			<Button variant="ghost" onclick={() => finish('stopped')}>{t('Stop and save')}</Button>
		</div>
		<div class="field sound">
			<label for="focus-sound-now">{t('Sound')}</label>
			<select id="focus-sound-now" bind:value={sound} onchange={syncSound}>
				{@render soundChoices()}
			</select>
			{#if playing && phase?.kind !== 'work'}<span class="muted small"
					>{t('Resumes with the next focus.')}</span
				>{/if}
		</div>
		<p class="muted small">
			{t('{minutes} min worked · round {round}', {
				minutes: workedMinutes(timer, now),
				round: Math.min(rounds(timer) + 1, timer.phases.filter((p) => p.kind === 'work').length)
			})}
		</p>
	</section>
{:else}
	<section class="card glass" aria-labelledby="start-title">
		<h2 id="start-title">{t('What will you focus on?')}</h2>
		{#if target}
			<p class="picked">
				<span class="chip">{t(typeNames[target.type])}</span>
				<strong>{target.title}</strong>
				<Button
					variant="ghost"
					onclick={() => (target = null)}
					aria-label={t('Choose something else')}>✕</Button
				>
			</p>
		{:else}
			<label for="focus-search">{t('A task, project, skill, note, or anything else')}</label>
			<input
				id="focus-search"
				type="search"
				placeholder={t('Search…')}
				bind:value={query}
				oninput={search}
				autocomplete="off"
			/>
			{#if hits.length}
				<ul class="hits">
					{#each hits as h (h.id)}
						<li>
							<button type="button" onclick={() => ((target = h), (hits = []), (query = ''))}
								><span class="chip">{t(typeNames[h.type])}</span> {h.title}</button
							>
						</li>
					{/each}
				</ul>
			{/if}
			<label for="focus-title">{t('…or just name it')}</label>
			<input id="focus-title" placeholder={t('Deep work')} bind:value={title} maxlength="500" />
		{/if}
		<div class="field">
			<label for="focus-sound">{t('Focus sound')}</label>
			<select id="focus-sound" bind:value={sound}>
				{@render soundChoices()}
			</select>
			{#if sound === 'music'}
				<span class="muted small"
					>{target
						? t('{style} for “{title}”.', {
								style: t(soundNames[styleFor(target.type)]),
								title: target.title
							})
						: t(
								'Lo-fi beats for tasks and projects, ambient for notes, calm piano for skills and cards.'
							)}</span
				>
			{/if}
		</div>
		{#if config.music.length}
			<p class="small">
				{t('Your music:')}
				{#each config.music as m (m.url)}
					<a href={m.url} target="_blank" rel="noopener noreferrer">{m.activity} ↗</a>{' '}
				{/each}
			</p>
		{:else}
			<p class="muted small">
				{t(
					'Add playlists you like for coding, reading, or writing in Preferences; they open here.'
				)}
			</p>
		{/if}
		<Button onclick={start}
			>{t('Start {minutes} min of focus', { minutes: totalWorkMinutes(plan) })}</Button
		>
	</section>
{/if}

{#if error}
	<Status kind="error" message={error} retry={loadHistory} />
{:else if history}
	<section class="card glass" aria-labelledby="history-title">
		<h2 id="history-title">{t('This week')}</h2>
		<ul class="stats">
			<li><strong>{history.today_minutes}</strong> {t('min today')}</li>
			<li><strong>{history.week_minutes}</strong> {t('min in 7 days')}</li>
			<li>
				<strong>{history.streak}</strong>
				{t(
					new Intl.PluralRules(i18n.locale).select(history.streak) === 'one'
						? 'day in a row'
						: 'days in a row'
				)}
			</li>
		</ul>
		{#if history.items.length}
			<table>
				<thead
					><tr
						><th scope="col">{t('When')}</th><th scope="col">{t('For')}</th><th scope="col"
							>{t('Worked')}</th
						></tr
					></thead
				>
				<tbody>
					{#each history.items as s (s.id)}
						<tr>
							<td
								>{new Date(s.started_at).toLocaleString(undefined, {
									weekday: 'short',
									hour: '2-digit',
									minute: '2-digit'
								})}</td
							>
							<td>{s.target ? s.target.title : s.title}</td>
							<td
								>{t(s.status === 'running' ? '{minutes} min (running)' : '{minutes} min', {
									minutes: s.worked_minutes
								})}</td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		{:else}
			<p class="muted">{t('No sessions yet this week.')}</p>
		{/if}
	</section>
{/if}

<style>
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.card,
	.timer {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	.timer {
		display: grid;
		justify-items: center;
		text-align: center;
	}
	.timer h2 {
		margin-bottom: 0;
	}
	.for {
		color: var(--muted);
		margin: var(--space-1) 0 var(--space-3);
	}
	svg {
		width: min(16rem, 70vw);
		height: auto;
	}
	.track {
		fill: none;
		stroke: var(--accent-soft);
		stroke-width: 12;
	}
	.arc {
		fill: none;
		stroke: var(--accent);
		stroke-width: 12;
		stroke-linecap: round;
	}
	.break .arc {
		stroke: var(--ok);
	}
	.time {
		font-size: 44px;
		font-weight: 650;
		fill: var(--fg);
		font-variant-numeric: tabular-nums;
	}
	.phases {
		display: flex;
		gap: var(--space-1);
		list-style: none;
		padding: 0;
		margin: var(--space-3) 0;
		flex-wrap: wrap;
		justify-content: center;
	}
	.phases li {
		font-size: var(--text-sm);
		padding: 0 var(--space-2);
		border-radius: 999px;
		border: 1px solid var(--border);
	}
	.phases li.work {
		border-color: var(--accent);
	}
	.phases li.done {
		opacity: 0.5;
	}
	.phases li.now {
		background: var(--accent-soft);
		font-weight: 650;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		justify-content: center;
	}
	label {
		display: block;
		font-weight: 600;
		margin: var(--space-3) 0 var(--space-1);
	}
	input,
	select {
		font: inherit;
		width: 100%;
		max-width: 32rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.field {
		margin-bottom: var(--space-4);
	}
	.hits {
		list-style: none;
		padding: 0;
		margin: var(--space-1) 0;
		max-width: 32rem;
	}
	.hits button {
		font: inherit;
		width: 100%;
		text-align: left;
		background: none;
		border: 0;
		border-bottom: 1px solid var(--border);
		padding: var(--space-2);
		color: var(--fg);
		cursor: pointer;
	}
	.hits button:hover,
	.hits button:focus-visible {
		background: var(--accent-soft);
	}
	.chip {
		font-size: var(--text-sm);
		color: var(--muted);
	}
	.picked {
		display: flex;
		gap: var(--space-2);
		align-items: center;
	}
	.stats {
		display: flex;
		gap: var(--space-6);
		list-style: none;
		padding: 0;
		flex-wrap: wrap;
	}
	.stats strong {
		font-size: var(--text-xl);
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--text-sm);
	}
	th,
	td {
		text-align: left;
		padding: var(--space-1) var(--space-2);
		border-bottom: 1px solid var(--border);
	}
</style>
