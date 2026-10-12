<script lang="ts">
	import { api, message, type Schemas } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Imports from other apps (EP-60): preview first, then import; importing
	// the same file again creates nothing twice.
	type Result = {
		source: string;
		dry_run: boolean;
		found: number;
		created: number;
		skipped: number;
		conflicts: number;
		problems: string[];
		sample: string[];
	};
	type Account = Schemas['BudgetAccount'];

	const sources = [
		{
			id: 'todoist',
			name: 'Todoist',
			what: 'Tasks, with @labels as contexts and dates as due dates.',
			how: 'In a project: … → Export as a template → CSV. One file per project.',
			accept: '.csv,text/csv'
		},
		{
			id: 'ticktick',
			name: 'TickTick',
			what: 'Tasks, done or open, with the first tag as the context.',
			how: 'Settings → Account → Generate backup (CSV).',
			accept: '.csv,text/csv'
		},
		{
			id: 'goodreads',
			name: 'Goodreads',
			what: 'Books: read, reading, or to read, with your stars as a rating out of 10.',
			how: 'My Books → Import and export → Export library.',
			accept: '.csv,text/csv'
		},
		{
			id: 'letterboxd',
			name: 'Letterboxd',
			what: 'Films: watched, rated, and on your watchlist.',
			how: 'Settings → Data → Export your data (the ZIP, as it comes).',
			accept: '.zip,.csv,application/zip,text/csv'
		},
		{
			id: 'firefly',
			name: 'Firefly III',
			what: 'Withdrawals and deposits into the budget; transfers are left out.',
			how: 'Export data → Transactions → CSV.',
			accept: '.csv,text/csv',
			money: true
		},
		{
			id: 'ofx',
			name: 'Your bank (OFX/QFX)',
			what: 'A bank statement’s transactions into the budget.',
			how: 'In online banking: download transactions as OFX, QFX, or “Money/Quicken”.',
			accept: '.ofx,.qfx,application/x-ofx',
			money: true
		},
		{
			id: 'notion',
			name: 'Notion',
			what: 'Pages as Markdown notes under Notion/, links as [[wikilinks]], pictures kept.',
			how: 'Settings → Export all workspace content (or … → Export on a page) → Markdown & CSV.',
			accept: '.zip,application/zip'
		}
	] as const;

	let files = $state<Record<string, File | null>>({});
	let results = $state<Record<string, Result>>({});
	let busy = $state('');
	let accounts = $state<Account[]>([]);
	let account = $state('');
	api<{ items: Account[] }>('/v1/budget/accounts')
		.then((r) => (accounts = r.items))
		.catch(() => {});

	async function run(id: string, dryRun: boolean, money: boolean) {
		const f = files[id];
		if (!f) return;
		busy = id;
		try {
			const q = new URLSearchParams();
			if (dryRun) q.set('dry_run', 'true');
			if (money && account) q.set('account_id', account);
			results[id] = await api<Result>(`/v1/import/${id}?${q}`, {
				method: 'POST',
				body: f,
				headers: { 'Content-Type': 'application/octet-stream' }
			});
			if (!dryRun)
				toast(`Imported ${results[id].created} from ${sources.find((s) => s.id === id)?.name}.`);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = '';
		}
	}
</script>

<svelte:head><title>Import · Axon</title></svelte:head>

<p class="muted">
	Bring your data from other apps. Preview first: nothing is written until you import, and importing
	the same file again adds nothing twice. Vault notes come in from Notes (ZIP), and any bank CSV
	from Budget.
</p>

<ul class="sources">
	{#each sources as s (s.id)}
		{@const r = results[s.id]}
		<li class="card">
			<h2>{s.name}</h2>
			<p>{s.what}</p>
			<p class="muted small">How to export: {s.how}</p>
			<label for="file-{s.id}">{s.name} export</label>
			<input
				id="file-{s.id}"
				type="file"
				accept={s.accept}
				onchange={(e) => {
					files[s.id] = e.currentTarget.files?.[0] ?? null;
					delete results[s.id];
				}}
			/>
			{#if 'money' in s && accounts.length}
				<label for="account-{s.id}">Into the account</label>
				<select id="account-{s.id}" bind:value={account}>
					<option value="">No account</option>
					{#each accounts as a (a.id)}<option value={a.id}>{a.name}</option>{/each}
				</select>
			{/if}
			<div class="row">
				<Button
					variant="secondary"
					disabled={!files[s.id]}
					busy={busy === s.id}
					onclick={() => run(s.id, true, 'money' in s)}>Preview</Button
				>
				<Button
					disabled={!files[s.id] || !r?.dry_run}
					busy={busy === s.id}
					onclick={() => run(s.id, false, 'money' in s)}>Import</Button
				>
			</div>
			{#if r}
				<div class="result" role="status">
					{#if r.dry_run}
						<p><strong>{r.found}</strong> found{r.sample.length ? ', for example:' : '.'}</p>
						<ul class="small">
							{#each r.sample as x, i (i)}<li>{x}</li>{/each}
						</ul>
					{:else}
						<p>
							<strong>{r.created}</strong> added, {r.skipped} already in Axon{r.conflicts
								? `, ${r.conflicts} left alone (a note with other content at the same path)`
								: ''}.
						</p>
					{/if}
					{#if r.problems.length}
						<details>
							<summary>{r.problems.length} not imported</summary>
							<ul class="small">
								{#each r.problems as p, i (i)}<li>{p}</li>{/each}
							</ul>
						</details>
					{/if}
				</div>
			{/if}
		</li>
	{/each}
</ul>

<style>
	.sources {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 20rem), 1fr));
		gap: var(--space-3);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		display: grid;
		gap: var(--space-2);
		align-content: start;
	}
	.card h2 {
		margin: 0;
		font-size: var(--text-lg);
	}
	.card p {
		margin: 0;
	}
	.small {
		font-size: var(--text-sm);
	}
	.row {
		display: flex;
		gap: var(--space-2);
	}
	.result {
		border-top: 1px solid var(--border);
		padding-top: var(--space-2);
	}
</style>
