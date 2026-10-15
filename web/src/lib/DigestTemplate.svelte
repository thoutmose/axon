<script lang="ts">
	import { api, message } from './api';
	import { t } from './i18n.svelte';
	import Button from './ui/Button.svelte';
	import Dialog from './ui/Dialog.svelte';
	import { toast } from './ui/toast.svelte';

	/** The digest template (EP-113): which sections each digest shows, in
	 * which order, and whether the quote and the picture come along. */
	let { open = $bindable(false), onsaved }: { open: boolean; onsaved: () => void } = $props();

	type Kind = 'daily' | 'weekly' | 'monthly';
	type Template = Record<Kind, string[]> & { quote: boolean; picture: boolean };
	const all: Record<Kind, string[]> = {
		daily: ['tasks', 'habits', 'coming', 'learn', 'watch'],
		weekly: ['open', 'week', 'reflect', 'watch'],
		monthly: ['goals', 'month', 'watch']
	};
	const labels: Record<string, string> = {
		tasks: 'To do today',
		habits: 'Habits',
		coming: 'Coming up',
		learn: 'Learn and remember',
		open: 'Still open',
		week: 'This week',
		reflect: 'Reflect',
		goals: 'Goals',
		month: 'This month',
		watch: 'Tech watch'
	};
	const kinds: [Kind, string][] = [
		['daily', 'Daily digest'],
		['weekly', 'Weekly digest'],
		['monthly', 'Monthly digest']
	];

	let draft = $state<Record<Kind, { id: string; on: boolean }[]>>({
		daily: [],
		weekly: [],
		monthly: []
	});
	let quote = $state(true);
	let picture = $state(true);

	$effect(() => {
		if (open) load();
	});
	async function load() {
		try {
			const doc = await api<{ digest: Template }>('/v1/me/settings');
			const tpl = doc.digest;
			for (const [k] of kinds) {
				draft[k] = [
					...tpl[k].map((id) => ({ id, on: true })),
					...all[k].filter((id) => !tpl[k].includes(id)).map((id) => ({ id, on: false }))
				];
			}
			quote = tpl.quote;
			picture = tpl.picture;
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	function move(k: Kind, i: number, by: number) {
		const j = i + by;
		if (j < 0 || j >= draft[k].length) return;
		const next = [...draft[k]];
		[next[i], next[j]] = [next[j], next[i]];
		draft[k] = next;
	}
	async function save() {
		try {
			const doc = await api<Record<string, unknown>>('/v1/me/settings');
			delete doc.ai;
			const pick = (k: Kind) => draft[k].filter((x) => x.on).map((x) => x.id);
			const digest: Template = {
				daily: pick('daily'),
				weekly: pick('weekly'),
				monthly: pick('monthly'),
				quote,
				picture
			};
			await api('/v1/me/settings', { method: 'PUT', body: { ...doc, digest } });
			open = false;
			toast(t('Digest template saved.'));
			onsaved();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<Dialog bind:open title={t('Customise the digests')}>
	<p class="muted small">
		{t('Choose the sections of each digest and their order. Email and every chat app follow it.')}
	</p>
	{#each kinds as [k, title] (k)}
		<fieldset>
			<legend>{t(title)}</legend>
			<ul class="choose">
				{#each draft[k] as x, i (x.id)}
					<li>
						<label><input type="checkbox" bind:checked={x.on} /> {t(labels[x.id])}</label>
						<span class="moves">
							<Button
								variant="ghost"
								onclick={() => move(k, i, -1)}
								disabled={i === 0}
								aria-label={t('Move {name} up', { name: t(labels[x.id]) })}>↑</Button
							>
							<Button
								variant="ghost"
								onclick={() => move(k, i, 1)}
								disabled={i === draft[k].length - 1}
								aria-label={t('Move {name} down', { name: t(labels[x.id]) })}>↓</Button
							>
						</span>
					</li>
				{/each}
			</ul>
		</fieldset>
	{/each}
	<label class="check"><input type="checkbox" bind:checked={quote} /> {t('A quote to carry')}</label
	>
	<label class="check"
		><input type="checkbox" bind:checked={picture} /> {t('A picture drawn for the day')}</label
	>
	<div class="actions">
		<Button onclick={save}>{t('Save')}</Button>
		<Button variant="secondary" onclick={() => (open = false)}>{t('Cancel')}</Button>
	</div>
</Dialog>

<style>
	.small {
		font-size: var(--text-sm);
	}
	fieldset {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		margin: var(--space-3) 0;
		padding: var(--space-2) var(--space-3);
	}
	legend {
		font-weight: 600;
		padding: 0 var(--space-1);
	}
	.choose {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.choose li {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-2);
	}
	.check {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		margin: var(--space-2) 0;
	}
	.actions {
		display: flex;
		gap: var(--space-2);
		margin-top: var(--space-4);
	}
</style>
