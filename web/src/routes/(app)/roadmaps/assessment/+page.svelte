<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Bank =
		paths['/v1/assessment/questions']['get']['responses'][200]['content']['application/json'];
	type Assessment = Schemas['Assessment'];

	let bank = $state<Bank['domains'] | null>(null);
	let history = $state<Assessment[]>([]);
	let error = $state('');
	let step = $state(1);
	let result = $state<Assessment | null>(null);
	let saving = $state(false);
	let profile = $state({
		role: '',
		experience: '',
		goals: '',
		hours_per_week: 5,
		formats: [] as string[]
	});
	let interests = $state<string[]>([]);
	let ratings = $state<Record<string, number>>({});
	let answers = $state<Record<string, number>>({});
	const formatNames: Record<string, string> = {
		book: 'Books',
		course: 'Courses',
		video: 'Videos',
		practice: 'Practice and projects',
		paper: 'Papers'
	};
	const ratingNames = [
		'Never touched it',
		'Heard of it',
		'Beginner',
		'Can work with help',
		'Confident',
		'Could teach it'
	];

	async function load() {
		try {
			const [q, h] = await Promise.all([
				api<Bank>('/v1/assessment/questions'),
				api<{ items: Assessment[] }>('/v1/assessments')
			]);
			bank = q.domains;
			history = h.items;
		} catch (err) {
			error = message(err);
		}
	}
	load();

	const chosen = $derived((bank ?? []).filter((d) => interests.includes(d.id)));

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		try {
			const r: Record<string, number> = {};
			for (const id of interests) r[id] = ratings[id] ?? 0;
			result = await api<Assessment>('/v1/assessments', {
				method: 'POST',
				body: { profile, interests, ratings: r, answers }
			});
			step = 4;
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			saving = false;
		}
	}

	async function start(domain: string) {
		try {
			const r = await api<Schemas['Roadmap']>('/v1/roadmaps/import', {
				method: 'POST',
				body: { source: domain, place: true }
			});
			await goto(`/roadmaps/${r.id}`);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const titleOf = (id: string) => bank?.find((d) => d.id === id)?.title ?? id;
	const levelOf = (a: Assessment, domain: string) =>
		a.results.find((r) => r.domain === domain)?.level;
	const allDomains = $derived([...new Set(history.flatMap((a) => a.results.map((r) => r.domain)))]);
</script>

<svelte:head><title>Skills assessment · Axon</title></svelte:head>

<p class="crumbs"><a href="/roadmaps">← Learn</a></p>
<h1>Skills assessment</h1>
<p class="muted hint">
	Ten minutes before you start: where you are, where you want to go, and what you already know. Your
	roadmaps then start at the right place, and the Tutor agent reads this instead of asking again.
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !bank}
	<Status kind="loading" message="Loading the assessment…" />
{:else}
	<ol class="steps" aria-label="Steps">
		{#each ['Your situation', 'What to learn', 'Quick checks', 'Results'] as label, i (label)}
			<li aria-current={step === i + 1 ? 'step' : undefined}>{label}</li>
		{/each}
	</ol>

	<form class="card glass" onsubmit={submit}>
		{#if step === 1}
			<h2>Your situation</h2>
			<TextField
				id="role"
				label="Your role or situation"
				placeholder="e.g. Support engineer, student, career change"
				bind:value={profile.role}
				maxlength={200}
			/>
			<TextField
				id="experience"
				label="Your experience"
				placeholder="e.g. 3 years of scripting"
				bind:value={profile.experience}
				maxlength={200}
			/>
			<div class="field">
				<label for="goals">What do you want to achieve?</label>
				<textarea id="goals" rows="3" maxlength="2000" bind:value={profile.goals}></textarea>
			</div>
			<div class="field">
				<label for="hours">Hours per week you can give</label>
				<input id="hours" type="number" min="0" max="80" bind:value={profile.hours_per_week} />
			</div>
			<fieldset>
				<legend>How do you like to learn?</legend>
				{#each Object.entries(formatNames) as [k, label] (k)}
					<label class="check"
						><input type="checkbox" value={k} bind:group={profile.formats} /> {label}</label
					>
				{/each}
			</fieldset>
			<div class="row end"><Button onclick={() => (step = 2)}>Next</Button></div>
		{:else if step === 2}
			<h2>What do you want to learn?</h2>
			<fieldset class="domains">
				<legend>Choose one or more domains</legend>
				{#each bank as d (d.id)}
					<label class="check"
						><input type="checkbox" value={d.id} bind:group={interests} /> {d.title}</label
					>
				{/each}
			</fieldset>
			<div class="row end">
				<Button variant="secondary" onclick={() => (step = 1)}>Back</Button>
				<Button
					onclick={() =>
						interests.length ? (step = 3) : toast('Choose at least one domain.', 'error')}
					>Next</Button
				>
			</div>
		{:else if step === 3}
			<h2>Quick checks</h2>
			<p class="muted">
				Rate yourself, then answer three questions per domain. Skip any you do not know: that is
				useful too.
			</p>
			{#each chosen as d (d.id)}
				<section class="domain" aria-labelledby="d-{d.id}">
					<h3 id="d-{d.id}">{d.title}</h3>
					<div class="field">
						<label for="rate-{d.id}">How well do you know it?</label>
						<select id="rate-{d.id}" bind:value={ratings[d.id]}>
							{#each ratingNames as name, i (i)}<option value={i}>{i} · {name}</option>{/each}
						</select>
					</div>
					{#each d.questions as q (q.id)}
						<fieldset>
							<legend>{q.question}</legend>
							{#each q.choices as c, i (i)}
								<label class="check"
									><input type="radio" name={q.id} value={i} bind:group={answers[q.id]} />
									{c}</label
								>
							{/each}
						</fieldset>
					{/each}
				</section>
			{/each}
			<div class="row end">
				<Button variant="secondary" onclick={() => (step = 2)}>Back</Button>
				<Button type="submit" busy={saving}>See my results</Button>
			</div>
		{:else if result}
			<h2>Your results</h2>
			<table>
				<thead
					><tr
						><th scope="col">Domain</th><th scope="col">Level</th><th scope="col">Why</th><th
							scope="col"><span class="visually-hidden">Action</span></th
						></tr
					></thead
				>
				<tbody>
					{#each result.results as r (r.domain)}
						<tr>
							<th scope="row">{r.title}</th>
							<td
								><span class="level" aria-label="Level {r.level} of 5"
									>{'●'.repeat(r.level)}{'○'.repeat(5 - r.level)}</span
								></td
							>
							<td class="small">{r.reason}</td>
							<td
								><Button
									variant={result.suggested[0] === r.domain ? 'primary' : 'secondary'}
									onclick={() => start(r.domain)}
									aria-label="Start {r.title} at my level">Start here</Button
								></td
							>
						</tr>
					{/each}
				</tbody>
			</table>
			{#if result.suggested.length}
				<p>
					Suggested first: <strong>{result.suggested.map(titleOf).join(', ')}</strong> (lowest level first).
				</p>
			{/if}
		{/if}
	</form>

	{#if history.length > 1 && allDomains.length}
		<section class="card glass" aria-labelledby="history-title">
			<h2 id="history-title">Your levels over time</h2>
			<div class="table-wrap">
				<table>
					<thead>
						<tr
							><th scope="col">Domain</th>{#each history as a (a.id)}<th scope="col"
									>{new Date(a.created_at).toLocaleDateString()}</th
								>{/each}</tr
						>
					</thead>
					<tbody>
						{#each allDomains as d (d)}
							<tr
								><th scope="row">{titleOf(d)}</th>{#each history as a (a.id)}<td
										>{levelOf(a, d) ?? '–'}</td
									>{/each}</tr
							>
						{/each}
					</tbody>
				</table>
			</div>
		</section>
	{/if}
{/if}

<style>
	.crumbs a {
		color: var(--muted);
		text-decoration: none;
	}
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.steps {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		list-style: none;
		padding: 0;
		margin: var(--space-4) 0;
		counter-reset: step;
	}
	.steps li {
		counter-increment: step;
		padding: var(--space-1) var(--space-3);
		border-radius: 999px;
		border: 1px solid var(--border);
		font-size: var(--text-sm);
		color: var(--muted);
	}
	.steps li::before {
		content: counter(step) '. ';
	}
	.steps li[aria-current='step'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
		max-width: 36rem;
	}
	.field label,
	legend {
		font-weight: 600;
	}
	textarea,
	input[type='number'],
	select {
		font: inherit;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	input[type='number'] {
		max-width: 8rem;
	}
	fieldset {
		border: 0;
		padding: 0;
		margin: 0 0 var(--space-4);
		display: grid;
		gap: var(--space-1);
	}
	.domains {
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 16rem), 1fr));
	}
	.domains legend {
		margin-bottom: var(--space-2);
	}
	.check {
		display: flex;
		gap: var(--space-2);
		align-items: baseline;
	}
	.domain {
		border-top: 1px solid var(--border);
		padding-top: var(--space-3);
	}
	.row {
		display: flex;
		gap: var(--space-2);
	}
	.end {
		justify-content: flex-end;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		margin-bottom: var(--space-4);
	}
	th,
	td {
		text-align: left;
		padding: var(--space-2);
		border-bottom: 1px solid var(--border);
		vertical-align: top;
	}
	.level {
		color: var(--accent);
		letter-spacing: 0.1em;
	}
	.table-wrap {
		overflow-x: auto;
	}
</style>
