<script lang="ts">
	import { api, message } from '$lib/api';
	import { loadBasics, type Category, type Rule } from '$lib/finance';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let rules = $state<Rule[] | null>(null);
	let categories = $state<Category[]>([]);
	let error = $state('');
	let form = $state({
		name: '',
		field: 'payee',
		operator: 'contains',
		value: '',
		category_id: '',
		tag: ''
	});
	const ops: Record<string, string> = {
		contains: 'contains',
		equals: 'is',
		starts_with: 'starts with'
	};
	const catName = (id: string | null) => categories.find((c) => c.id === id)?.name ?? '';

	async function load() {
		try {
			const [r, basics] = await Promise.all([
				api<{ items: Rule[] }>('/v1/budget/rules'),
				loadBasics()
			]);
			rules = r.items;
			categories = basics.categories;
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function create(e: SubmitEvent) {
		e.preventDefault();
		try {
			await api('/v1/budget/rules', {
				method: 'POST',
				body: {
					...form,
					name: form.name || `${form.field} ${ops[form.operator]} “${form.value}”`,
					category_id: form.category_id || null
				}
			});
			form = { ...form, name: '', value: '', tag: '' };
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function apply() {
		try {
			const r = await api<{ changed: number }>('/v1/budget/rules/apply', { method: 'POST' });
			toast(`${r.changed} uncategorised transaction${r.changed === 1 ? '' : 's'} updated.`);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function patch(r: Rule, body: Record<string, unknown>) {
		try {
			await api(`/v1/budget/rules/${r.id}`, { method: 'PATCH', body });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove(r: Rule) {
		try {
			await api(`/v1/budget/rules/${r.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>Rules · Budget · Axon</title></svelte:head>

<h1>Rules</h1>
<p class="muted small">
	When a new transaction has no category, the rules are tried in order: the first matching rule with
	a category of the right kind sets it, and every matching rule adds its tag. A category you chose
	is never overridden.
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !rules}
	<Status kind="loading" message="Loading…" />
{:else}
	<section class="card" aria-labelledby="list">
		<div class="row">
			<h2 id="list">Your rules</h2>
			<Button variant="secondary" onclick={apply}>Apply to uncategorised transactions</Button>
		</div>
		{#if rules.length === 0}
			<p class="muted">
				No rules yet. Imports and new transactions stay uncategorised until you add one.
			</p>
		{:else}
			<table>
				<thead
					><tr
						><th scope="col">When</th><th scope="col">Then</th><th scope="col"
							><span class="visually-hidden">Actions</span></th
						></tr
					></thead
				>
				<tbody>
					{#each rules as r (r.id)}
						<tr class:muted={!r.active}>
							<td>{r.field} {ops[r.operator]} “{r.value}”</td>
							<td
								>{#if r.category_id}category {catName(
										r.category_id
									)}{/if}{#if r.category_id && r.tag},
								{/if}{#if r.tag}tag #{r.tag}{/if}</td
							>
							<td class="row"
								><button class="link" onclick={() => patch(r, { active: !r.active })}
									>{r.active ? 'Turn off' : 'Turn on'}</button
								><button class="link" onclick={() => remove(r)} aria-label="Delete rule {r.name}"
									>Delete</button
								></td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>

	<section class="card" aria-labelledby="new">
		<h2 id="new">Add a rule</h2>
		<form class="grid" onsubmit={create}>
			<div class="field">
				<label for="rule-field">When the</label>
				<select id="rule-field" bind:value={form.field}>
					<option value="payee">payee</option>
					<option value="note">note</option>
				</select>
			</div>
			<div class="field">
				<label for="rule-op">Test</label>
				<select id="rule-op" bind:value={form.operator}>
					{#each Object.entries(ops) as [k, v] (k)}<option value={k}>{v}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="rule-value">Text</label>
				<input
					id="rule-value"
					bind:value={form.value}
					maxlength="200"
					required
					placeholder="carrefour"
				/>
			</div>
			<div class="field">
				<label for="rule-category">Set category</label>
				<select id="rule-category" bind:value={form.category_id}>
					<option value="">—</option>
					{#each categories.filter((c) => !c.archived) as c (c.id)}<option value={c.id}
							>{c.name} ({c.kind})</option
						>{/each}
				</select>
			</div>
			<div class="field">
				<label for="rule-tag">Add tag</label>
				<input id="rule-tag" bind:value={form.tag} maxlength="41" placeholder="#food" />
			</div>
			<div><Button type="submit">Add rule</Button></div>
		</form>
	</section>
{/if}
