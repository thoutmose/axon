<script lang="ts">
	import { api, message } from '$lib/api';
	import { loadBasics, type Account } from '$lib/finance';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let accounts = $state<Account[]>([]);
	loadBasics()
		.then((b) => (accounts = b.accounts.filter((a) => !a.archived)))
		.catch(() => {});

	let csv = $state('');
	let spec = $state({
		account_id: '',
		delimiter: ',',
		date_format: 'YYYY-MM-DD',
		has_header: true,
		date_column: 0,
		amount_column: 1,
		payee_column: 2 as number | null,
		note_column: null as number | null
	});
	let busy = $state(false);
	let result = $state<{ created: number; skipped: number; problems: string[] } | null>(null);

	// The first rows, split like the server will, to pick the columns.
	const preview = $derived(
		csv
			.split(/\r?\n/)
			.filter(Boolean)
			.slice(0, 4)
			.map((l) => l.split(spec.delimiter || ','))
	);
	const columns = $derived(
		preview[0]?.map((h, i) => ({
			i,
			label: spec.has_header ? `${i + 1}: ${h}` : `Column ${i + 1}`
		})) ?? []
	);

	async function read(e: Event) {
		const file = (e.currentTarget as HTMLInputElement).files?.[0];
		if (!file) return;
		if (file.size > 3_500_000) {
			toast('Split files larger than 3.5 MB.', 'error');
			return;
		}
		csv = await file.text();
		const first = csv.split(/\r?\n/)[0] ?? '';
		spec.delimiter =
			(first.match(/;/g)?.length ?? 0) > (first.match(/,/g)?.length ?? 0) ? ';' : ',';
	}

	async function run(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			result = await api('/v1/budget/import', {
				method: 'POST',
				body: { ...spec, csv, account_id: spec.account_id || null }
			});
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>Import · Budget · Axon</title></svelte:head>

<h1>Import a bank export</h1>
<p class="muted small">
	Download a CSV from your bank and import it here. Negative amounts are expenses, positive ones
	income. Rows already imported are skipped, so importing an overlapping export twice is safe, and
	your rules categorise the new rows. The file stays on your Axon.
</p>

<section class="card" aria-labelledby="file">
	<h2 id="file">1. Choose the file</h2>
	<label for="csv-file" class="visually-hidden">CSV file</label>
	<input id="csv-file" type="file" accept=".csv,text/csv,text/plain" onchange={read} />
	<details>
		<summary class="small">Or paste the CSV</summary>
		<label for="csv-text" class="visually-hidden">CSV text</label>
		<textarea id="csv-text" rows="6" bind:value={csv} style:width="100%"></textarea>
	</details>
</section>

{#if preview.length}
	<section class="card" aria-labelledby="map">
		<h2 id="map">2. Tell Axon what the columns are</h2>
		<table>
			<tbody>
				{#each preview as row, i (i)}
					<tr class:muted={i === 0 && spec.has_header}
						>{#each row as cell, j (j)}<td>{cell}</td>{/each}</tr
					>
				{/each}
			</tbody>
		</table>
		<form class="grid" onsubmit={run}>
			<div class="field">
				<label for="imp-account">Into account</label>
				<select id="imp-account" bind:value={spec.account_id}>
					<option value="">No account</option>
					{#each accounts as a (a.id)}<option value={a.id}>{a.name}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="imp-delim">Separator</label>
				<select id="imp-delim" bind:value={spec.delimiter}>
					<option value=",">Comma ,</option>
					<option value=";">Semicolon ;</option>
					<option value="	">Tab</option>
				</select>
			</div>
			<div class="field">
				<label for="imp-format">Dates look like</label>
				<select id="imp-format" bind:value={spec.date_format}>
					{#each ['YYYY-MM-DD', 'DD/MM/YYYY', 'MM/DD/YYYY', 'DD.MM.YYYY'] as f (f)}<option value={f}
							>{f}</option
						>{/each}
				</select>
			</div>
			<div class="field">
				<label for="imp-date">Date column</label>
				<select id="imp-date" bind:value={spec.date_column}>
					{#each columns as c (c.i)}<option value={c.i}>{c.label}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="imp-amount">Amount column</label>
				<select id="imp-amount" bind:value={spec.amount_column}>
					{#each columns as c (c.i)}<option value={c.i}>{c.label}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="imp-payee">Payee column</label>
				<select id="imp-payee" bind:value={spec.payee_column}>
					<option value={null}>None</option>
					{#each columns as c (c.i)}<option value={c.i}>{c.label}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="imp-note">Note column</label>
				<select id="imp-note" bind:value={spec.note_column}>
					<option value={null}>None</option>
					{#each columns as c (c.i)}<option value={c.i}>{c.label}</option>{/each}
				</select>
			</div>
			<label class="row"
				><input type="checkbox" bind:checked={spec.has_header} /> First row is a header</label
			>
			<div class="wide"><Button type="submit" {busy}>Import</Button></div>
		</form>
	</section>
{/if}

{#if result}
	<section class="card" aria-labelledby="done" role="status">
		<h2 id="done">Imported {result.created}, skipped {result.skipped} already imported</h2>
		{#if result.problems.length}
			<p>Some rows were left out:</p>
			<ul>
				{#each result.problems as p (p)}<li class="small">{p}</li>{/each}
			</ul>
		{/if}
		<a href="/budget">See the budget →</a>
	</section>
{/if}
