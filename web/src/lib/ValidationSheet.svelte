<script lang="ts">
	// A domain is proven, not ticked (EP-181, after École 42): the skill's
	// evaluation sheet, each item of its proof checked with where the
	// evidence is, the explanation in the person's own words, and the mark.
	import { t } from '$lib/i18n.svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { formatDate } from '$lib/dates';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Sheet = Schemas['SkillValidationSheet'];
	type Item = Schemas['SkillValidationItem'];
	type Attempt = Schemas['SkillValidation'];

	let {
		skillId,
		title,
		open = $bindable(false),
		onvalidated
	}: {
		skillId: string;
		title: string;
		open: boolean;
		onvalidated?: (attempt: Attempt) => void;
	} = $props();

	let sheet = $state<Sheet | null>(null);
	let items = $state<Item[]>([]);
	let explanation = $state('');
	let error = $state('');
	let busy = $state(false);
	let result = $state<Attempt | null>(null);

	async function load() {
		error = '';
		result = null;
		sheet = null;
		try {
			sheet = await api<Sheet>(`/v1/skills/${skillId}/validation`);
			items = sheet.items.map((i) => ({ ...i }));
			explanation = sheet.attempts[0]?.explanation ?? '';
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void skillId;
		if (open) load();
	});

	const addItem = (kind: 'mandatory' | 'bonus') => {
		items = [...items, { text: '', kind, ok: false, evidence: '' }];
	};
	const remove = (i: number) => {
		items = items.filter((_, n) => n !== i);
	};
	const missing = (v: Attempt) => v.items.filter((i) => i.kind === 'mandatory' && !i.ok).length;
	const flagLabel: Record<string, string> = $derived({
		ok: t('Proven'),
		outstanding: t('Outstanding'),
		incomplete: t('Not yet')
	});

	async function validate(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			result = await api<Attempt>(`/v1/skills/${skillId}/validations`, {
				method: 'POST',
				body: { items, explanation }
			});
			if (sheet) sheet = { ...sheet, attempts: [result, ...sheet.attempts] };
			if (result.flag === 'incomplete') {
				toast(
					t('Not yet: {n} mandatory items are missing. Try again when they hold.', {
						n: missing(result)
					})
				);
			} else {
				toast(t('{title} proven: {mark}/100.', { title, mark: result.mark }));
			}
			onvalidated?.(result);
		} catch (err) {
			error = message(err);
		} finally {
			busy = false;
		}
	}
</script>

<Dialog bind:open title={t('Prove {title}', { title })}>
	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !sheet}
		<Status kind="loading" message={t('Loading the evaluation sheet…')} />
	{:else}
		<p class="muted small">
			{t(
				'As at École 42: check each item of the proof with where its evidence is, then explain it in your own words. Every mandatory item makes 100, the bonus items add up to 25, and one missing means another try.'
			)}
		</p>
		<form onsubmit={validate}>
			<ol class="items">
				{#each items as it, i (i)}
					<li class:bonus={it.kind === 'bonus'}>
						<input
							type="checkbox"
							bind:checked={it.ok}
							aria-label={t('Item {n} holds', { n: i + 1 })}
						/>
						<div class="grow">
							<input
								type="text"
								bind:value={it.text}
								maxlength="300"
								required
								placeholder={t('What must hold')}
								aria-label={t('Item {n}', { n: i + 1 })}
							/>
							<input
								type="text"
								bind:value={it.evidence}
								maxlength="500"
								placeholder={t('Where the evidence is: a link, a commit, a note')}
								aria-label={t('Evidence for item {n}', { n: i + 1 })}
							/>
						</div>
						<select bind:value={it.kind} aria-label={t('Kind of item {n}', { n: i + 1 })}>
							<option value="mandatory">{t('Mandatory')}</option>
							<option value="bonus">{t('Bonus')}</option>
						</select>
						<button
							type="button"
							class="link"
							onclick={() => remove(i)}
							aria-label={t('Remove item {n}', { n: i + 1 })}>×</button
						>
					</li>
				{/each}
			</ol>
			<div class="row">
				<Button variant="secondary" type="button" onclick={() => addItem('mandatory')}
					>{t('Add an item')}</Button
				>
				<Button variant="secondary" type="button" onclick={() => addItem('bonus')}
					>{t('Add a bonus item')}</Button
				>
			</div>
			<label for="explanation-{skillId}"
				>{t('Explain it in your own words, as if to a beginner: what it is, why it works')}</label
			>
			<textarea id="explanation-{skillId}" bind:value={explanation} rows="5" maxlength="5000"
			></textarea>
			<div class="row">
				<Button type="submit" disabled={busy}>{t('Validate')}</Button>
				<Button variant="secondary" type="button" onclick={() => (open = false)}
					>{t('Close')}</Button
				>
			</div>
		</form>
		{#if result}
			<p class="result {result.flag}" role="status">
				{flagLabel[result.flag]} · {result.mark}/100{result.flag === 'incomplete'
					? ' · ' + t('{n} mandatory items missing', { n: missing(result) })
					: ''}
			</p>
		{/if}
		{#if sheet.attempts.length}
			<h3>{t('Attempts')}</h3>
			<ul class="attempts">
				{#each sheet.attempts as a (a.id)}
					<li>
						<span class={a.flag}>{flagLabel[a.flag]}</span> · {a.mark}/100 · {formatDate(
							a.created_at.slice(0, 10)
						)}
					</li>
				{/each}
			</ul>
		{/if}
	{/if}
</Dialog>

<style>
	.small {
		font-size: var(--text-sm);
	}
	.items {
		list-style: decimal;
		padding-left: 1.5rem;
		display: grid;
		gap: var(--space-2);
		margin: 0 0 var(--space-3);
	}
	.items li {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
	}
	.items li.bonus {
		color: var(--muted);
	}
	.items input[type='checkbox'] {
		margin-top: 0.6rem;
	}
	.grow {
		flex: 1;
		display: grid;
		gap: var(--space-1);
		min-width: 0;
	}
	.grow input {
		width: 100%;
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin: var(--space-2) 0;
	}
	textarea {
		width: 100%;
		margin-top: var(--space-1);
	}
	.result {
		font-weight: 600;
	}
	.result.incomplete,
	.attempts .incomplete {
		color: var(--bad);
	}
	.result.ok,
	.result.outstanding,
	.attempts .ok,
	.attempts .outstanding {
		color: var(--ok);
	}
	.attempts {
		margin: 0;
		padding-left: 1.2rem;
	}
	h3 {
		font-size: var(--text-sm);
		margin: var(--space-3) 0 var(--space-1);
	}
</style>
