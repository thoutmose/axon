<script lang="ts">
	import { t } from '$lib/i18n.svelte';
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
	const ops: Record<string, string> = $derived({
		contains: 'contains',
		equals: 'is',
		starts_with: t('starts with')
	});
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

<svelte:head><title>{t('Rules · Budget · Axon')}</title></svelte:head>

<h1>{t('Rules')}</h1>
<p class="muted small">
	{t(
		'When a new transaction has no category, the rules are tried in order: the first matching rule with a category of the right kind sets it, and every matching rule adds its tag. A category you chose is never overridden.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !rules}
	<Status kind="loading" message={t('Loading…')} />
{:else}
	<section class="card" aria-labelledby="list">
		<div class="row">
			<h2 id="list">{t('Your rules')}</h2>
			<Button variant="secondary" onclick={apply}>{t('Apply to uncategorised transactions')}</Button
			>
		</div>
		{#if rules.length === 0}
			<p class="muted">
				{t('No rules yet. Imports and new transactions stay uncategorised until you add one.')}
			</p>
		{:else}
			<table>
				<thead
					><tr
						><th scope="col">{t('When')}</th><th scope="col">{t('Then')}</th><th scope="col"
							><span class="visually-hidden">{t('Actions')}</span></th
						></tr
					></thead
				>
				<tbody>
					{#each rules as r (r.id)}
						<tr class:muted={!r.active}>
							<td>{r.field} {ops[r.operator]} “{r.value}”</td>
							<td
								>{#if r.category_id}{t('category {value}', {
										value: catName(r.category_id)
									})}{/if}{#if r.category_id && r.tag},
								{/if}{#if r.tag}{t('tag #{tag}', { tag: r.tag })}{/if}</td
							>
							<td class="row"
								><button class="link" onclick={() => patch(r, { active: !r.active })}
									>{r.active ? t('Turn off') : t('Turn on')}</button
								><button
									class="link"
									onclick={() => remove(r)}
									aria-label={t('Delete rule {name}', { name: r.name })}>{t('Delete')}</button
								></td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>

	<section class="card" aria-labelledby="new">
		<h2 id="new">{t('Add a rule')}</h2>
		<form class="grid" onsubmit={create}>
			<div class="field">
				<label for="rule-field">{t('When the')}</label>
				<select id="rule-field" bind:value={form.field}>
					<option value="payee">{t('payee')}</option>
					<option value="note">{t('note')}</option>
				</select>
			</div>
			<div class="field">
				<label for="rule-op">{t('Test')}</label>
				<select id="rule-op" bind:value={form.operator}>
					{#each Object.entries(ops) as [k, v] (k)}<option value={k}>{v}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="rule-value">{t('Text')}</label>
				<input
					id="rule-value"
					bind:value={form.value}
					maxlength="200"
					required
					placeholder={t('carrefour')}
				/>
			</div>
			<div class="field">
				<label for="rule-category">{t('Set category')}</label>
				<select id="rule-category" bind:value={form.category_id}>
					<option value="">—</option>
					{#each categories.filter((c) => !c.archived) as c (c.id)}<option value={c.id}
							>{c.name} ({c.kind})</option
						>{/each}
				</select>
			</div>
			<div class="field">
				<label for="rule-tag">{t('Add tag')}</label>
				<input id="rule-tag" bind:value={form.tag} maxlength="41" placeholder={t('#food')} />
			</div>
			<div><Button type="submit">{t('Add rule')}</Button></div>
		</form>
	</section>
{/if}
