<script lang="ts">
	import { api, message, type Schemas } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import { canReadBarcodes, readBarcode, shrink } from '$lib/photo';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// The camera on Health (EP-89): a photo of a meal is estimated by the
	// person's AI provider, line by line, for them to check before logging;
	// a photo of a barcode is read in the browser and looked up as a product.
	let {
		date,
		meal,
		onlogged,
		onfood
	}: {
		date: string;
		meal: string;
		onlogged: () => void;
		onfood: (f: Schemas['Food']) => void;
	} = $props();

	type Line = {
		on: boolean;
		name: string;
		grams: string;
		kcal: string;
		protein_g: string;
		carbs_g: string;
		fat_g: string;
	};
	let busy = $state(false);
	let lines = $state<Line[] | null>(null);
	let note = $state('');
	let pending = $state<Blob | null>(null);
	let consent = $state<{ id: string; description: string }[]>([]);
	const barcodes = canReadBarcodes();
	const num = (v: string) => (v.trim() === '' ? undefined : Number(v.replace(',', '.')));

	/** The consents a photo needs that are not given yet. */
	async function missing(): Promise<{ id: string; description: string }[]> {
		const r = await api<{ items: Schemas['Consent'][] }>('/v1/me/consents');
		return r.items.filter((c) => ['ai_agents', 'food_photos'].includes(c.id) && !c.granted);
	}

	async function picked(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file) return;
		busy = true;
		try {
			pending = await shrink(file);
			consent = await missing();
			if (!consent.length) await send();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function allowAndSend() {
		busy = true;
		try {
			for (const c of consent)
				await api(`/v1/me/consents/${c.id}`, { method: 'PUT', body: { granted: true } });
			consent = [];
			await send();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function send() {
		if (!pending) return;
		const r = await api<{
			items: {
				name: string;
				grams: number;
				kcal: number;
				protein_g: number;
				carbs_g: number;
				fat_g: number;
			}[];
			note: string;
		}>('/v1/health/food/estimate', {
			method: 'POST',
			body: pending,
			headers: { 'Content-Type': 'image/jpeg' }
		});
		pending = null;
		note = r.note;
		lines = r.items.map((i) => ({
			on: true,
			name: i.name,
			grams: String(i.grams),
			kcal: String(i.kcal),
			protein_g: String(i.protein_g),
			carbs_g: String(i.carbs_g),
			fat_g: String(i.fat_g)
		}));
	}

	async function logChecked() {
		if (!lines) return;
		busy = true;
		try {
			for (const l of lines.filter((x) => x.on && x.name.trim())) {
				await api('/v1/health/food', {
					method: 'POST',
					body: {
						eaten_on: date,
						meal,
						name: l.name.trim(),
						grams: num(l.grams),
						kcal: num(l.kcal),
						protein_g: num(l.protein_g),
						carbs_g: num(l.carbs_g),
						fat_g: num(l.fat_g)
					}
				});
			}
			lines = null;
			note = '';
			onlogged();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function barcode(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file) return;
		busy = true;
		try {
			const code = await readBarcode(file);
			if (!code) {
				toast(t('No barcode found in the photo. Try closer, with the lines straight.'), 'error');
				return;
			}
			onfood(await api<Schemas['Food']>(`/v1/health/foods/barcode/${code}`));
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
</script>

<div class="camera">
	<label class="pick" aria-disabled={busy}>
		📷 {t('Photo of a meal')}
		<input
			type="file"
			accept="image/jpeg,image/png,image/heic,image/webp"
			capture="environment"
			disabled={busy}
			onchange={picked}
		/>
	</label>
	{#if barcodes}
		<label class="pick" aria-disabled={busy}>
			▥ {t('Photo of a barcode')}
			<input
				type="file"
				accept="image/*"
				capture="environment"
				disabled={busy}
				onchange={barcode}
			/>
		</label>
	{/if}
	{#if busy}<span class="small muted" role="status">{t('Looking at the photo…')}</span>{/if}
</div>

{#if consent.length}
	<div class="consent" role="group" aria-label={t('Before the photo is sent')}>
		<p>{t('To estimate the meal, Axon sends the photo to your AI provider:')}</p>
		<ul class="small">
			{#each consent as c (c.id)}<li>{c.description}</li>{/each}
		</ul>
		<div class="row">
			<Button onclick={allowAndSend} {busy}>{t('Allow and send')}</Button>
			<Button variant="ghost" onclick={() => ((consent = []), (pending = null))}
				>{t('Cancel')}</Button
			>
		</div>
	</div>
{/if}

{#if lines}
	<section class="estimate" aria-labelledby="estimate-title">
		<h3 id="estimate-title">{t('What the photo seems to hold')}</h3>
		<p class="warning" role="note">
			{t(
				'Estimates from a photo can be far off, portions above all. Check and correct each line before you log it.'
			)}
		</p>
		{#if note}<p class="small muted">{note}</p>{/if}
		{#if lines.length}
			<div class="scroll">
				<table>
					<thead>
						<tr>
							<th scope="col">{t('Log')}</th>
							<th scope="col">{t('Food')}</th>
							<th scope="col">g</th>
							<th scope="col">kcal</th>
							<th scope="col">{t('Protein')}</th>
							<th scope="col">{t('Carbohydrates')}</th>
							<th scope="col">{t('Fat')}</th>
						</tr>
					</thead>
					<tbody>
						{#each lines as l, i (i)}
							<tr>
								<td
									><input
										type="checkbox"
										bind:checked={l.on}
										aria-label={t('Log {name}', { name: l.name })}
									/></td
								>
								<td><input bind:value={l.name} aria-label={t('Food')} /></td>
								<td><input inputmode="decimal" bind:value={l.grams} aria-label="g" /></td>
								<td><input inputmode="decimal" bind:value={l.kcal} aria-label="kcal" /></td>
								<td
									><input
										inputmode="decimal"
										bind:value={l.protein_g}
										aria-label={t('Protein')}
									/></td
								>
								<td
									><input
										inputmode="decimal"
										bind:value={l.carbs_g}
										aria-label={t('Carbohydrates')}
									/></td
								>
								<td><input inputmode="decimal" bind:value={l.fat_g} aria-label={t('Fat')} /></td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			<div class="row">
				<Button onclick={logChecked} {busy}>{t('Log the checked lines')}</Button>
				<Button variant="ghost" onclick={() => (lines = null)}>{t('Discard')}</Button>
			</div>
		{:else}
			<Button variant="ghost" onclick={() => (lines = null)}>{t('Discard')}</Button>
		{/if}
	</section>
{/if}

<style>
	.camera,
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
		margin: var(--space-2) 0;
	}
	.pick {
		position: relative;
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-6);
		border-radius: 999px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		font-weight: 600;
		cursor: pointer;
	}
	.pick[aria-disabled='true'] {
		opacity: 0.6;
	}
	.camera input {
		position: absolute;
		inset: 0;
		opacity: 0;
		cursor: pointer;
	}
	.camera label:focus-within {
		box-shadow: var(--focus);
	}
	.consent,
	.estimate {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-3);
		margin: var(--space-2) 0;
	}
	.warning {
		border-left: 4px solid var(--accent);
		background: var(--accent-soft);
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		font-size: var(--text-sm);
	}
	.scroll {
		overflow-x: auto;
	}
	table {
		border-collapse: collapse;
		width: 100%;
	}
	td input:not([type='checkbox']) {
		width: 100%;
		min-width: 4rem;
		font: inherit;
		padding: var(--space-1);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--fg);
	}
	td:nth-child(2) input {
		min-width: 10rem;
	}
	th {
		text-align: left;
		font-size: var(--text-sm);
		padding: var(--space-1);
	}
	td {
		padding: var(--space-1);
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
