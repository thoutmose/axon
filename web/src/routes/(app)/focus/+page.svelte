<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
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
	import { chime, play, type Sound } from '$lib/sounds';
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

	const names: Record<string, string> = { work: 'Focus', short: 'Short break', long: 'Long break' };
	const plan = $derived(phases(config));
	const phase = $derived(timer && !finished(timer) ? timer.phases[timer.index] : null);
	const left = $derived(timer ? remaining(timer, now) : 0);
	const progress = $derived(phase ? 1 - left / (phase.minutes * 60_000) : 0);

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
		if (typeof document !== 'undefined') document.title = 'Focus · Axon';
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
				? `${names[ended[ended.length - 1].kind]} over. ${names[next.kind]} for ${next.minutes} minutes.`
				: 'All rounds done. Well worked.';
			syncSound();
			save();
		}
		document.title = `${clock(remaining(timer, now))} · ${phase ? names[phase.kind] : 'Focus'} · Axon`;
		if (finished(timer)) finish('completed');
	}

	function syncSound() {
		stopSound();
		stopSound =
			timer && !timer.paused && phase?.kind === 'work' ? play(sound, config.volume) : () => {};
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
		try {
			session = await api<Session>('/v1/focus-sessions', {
				method: 'POST',
				body: {
					planned_minutes: totalWorkMinutes(plan),
					...(target ? { target_id: target.id } : { title: title.trim() || 'Focus' })
				}
			});
			timer = newTimer(plan, Date.now());
			announce = `Focus for ${plan[0].minutes} minutes.`;
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
		document.title = 'Focus · Axon';
		try {
			await api(`/v1/focus-sessions/${s0.id}/finish`, {
				method: 'POST',
				body: { worked_minutes: workedMinutes(t0, Date.now()), rounds: rounds(t0), status }
			});
			toast(status === 'completed' ? 'Session complete.' : 'Session saved.');
		} catch (err) {
			toast(message(err), 'error');
		}
		await loadHistory();
	}

	const R = 90;
	const C = 2 * Math.PI * R;
</script>

<svelte:head><title>{t('Focus')} · Axon</title></svelte:head>

<h1>{t('Focus')}</h1>
<p class="muted hint">
	Pomodoro sessions: {config.work_minutes} min of focus and {config.short_break_minutes} min breaks, a
	{config.long_break_minutes} min break after {config.rounds} rounds{config.sessions > 1
		? `, ${config.sessions} times`
		: ''}. Change it in <a href="/account/preferences">Preferences</a>.
</p>

<p class="visually-hidden" aria-live="assertive">{announce}</p>

{#if timer && session}
	<section class="timer glass" aria-labelledby="phase-title" class:break={phase?.kind !== 'work'}>
		<h2 id="phase-title">{phase ? names[phase.kind] : 'Done'}</h2>
		<p class="for">
			{session.target
				? `${typeNames[session.target.type] ?? ''}: ${session.target.title}`
				: session.title}
		</p>
		<svg
			viewBox="0 0 220 220"
			role="img"
			aria-label="{clock(left)} left in {phase ? names[phase.kind] : 'the session'}"
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
		<ol class="phases" aria-label="Plan">
			{#each timer.phases as p, i (i)}
				<li
					class={p.kind}
					class:done={i < timer.index}
					class:now={i === timer.index}
					aria-current={i === timer.index ? 'step' : undefined}
				>
					<span class="visually-hidden">{names[p.kind]}, </span>{p.minutes}′
				</li>
			{/each}
		</ol>
		<div class="row">
			<Button onclick={togglePause}>{timer.paused ? 'Resume' : 'Pause'}</Button>
			<Button variant="secondary" onclick={next}>Skip to next</Button>
			<Button variant="ghost" onclick={() => finish('stopped')}>Stop and save</Button>
		</div>
		<p class="muted small">
			{workedMinutes(timer, now)} min worked · round {Math.min(
				rounds(timer) + 1,
				timer.phases.filter((p) => p.kind === 'work').length
			)}
		</p>
	</section>
{:else}
	<section class="card glass" aria-labelledby="start-title">
		<h2 id="start-title">What will you focus on?</h2>
		{#if target}
			<p class="picked">
				<span class="chip">{typeNames[target.type]}</span>
				<strong>{target.title}</strong>
				<Button variant="ghost" onclick={() => (target = null)} aria-label="Choose something else"
					>✕</Button
				>
			</p>
		{:else}
			<label for="focus-search">A task, project, skill, note, or anything else</label>
			<input
				id="focus-search"
				type="search"
				placeholder="Search…"
				bind:value={query}
				oninput={search}
				autocomplete="off"
			/>
			{#if hits.length}
				<ul class="hits">
					{#each hits as h (h.id)}
						<li>
							<button type="button" onclick={() => ((target = h), (hits = []), (query = ''))}
								><span class="chip">{typeNames[h.type]}</span> {h.title}</button
							>
						</li>
					{/each}
				</ul>
			{/if}
			<label for="focus-title">…or just name it</label>
			<input id="focus-title" placeholder="Deep work" bind:value={title} maxlength="500" />
		{/if}
		<div class="field">
			<label for="focus-sound">Focus sound</label>
			<select id="focus-sound" bind:value={sound}>
				<option value="none">None</option>
				<option value="white">White noise</option>
				<option value="pink">Pink noise</option>
				<option value="brown">Brown noise</option>
				<option value="rain">Rain</option>
			</select>
		</div>
		{#if config.music.length}
			<p class="small">
				Your music:
				{#each config.music as m (m.url)}
					<a href={m.url} target="_blank" rel="noopener noreferrer">{m.activity} ↗</a>{' '}
				{/each}
			</p>
		{:else}
			<p class="muted small">
				Add playlists you like for coding, reading, or writing in Preferences; they open here.
			</p>
		{/if}
		<Button onclick={start}>Start {totalWorkMinutes(plan)} min of focus</Button>
	</section>
{/if}

{#if error}
	<Status kind="error" message={error} retry={loadHistory} />
{:else if history}
	<section class="card glass" aria-labelledby="history-title">
		<h2 id="history-title">This week</h2>
		<ul class="stats">
			<li><strong>{history.today_minutes}</strong> min today</li>
			<li><strong>{history.week_minutes}</strong> min in 7 days</li>
			<li><strong>{history.streak}</strong> {history.streak === 1 ? 'day' : 'days'} in a row</li>
		</ul>
		{#if history.items.length}
			<table>
				<thead
					><tr><th scope="col">When</th><th scope="col">For</th><th scope="col">Worked</th></tr
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
							<td>{s.worked_minutes} min{s.status === 'running' ? ' (running)' : ''}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{:else}
			<p class="muted">No sessions yet this week.</p>
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
