<script lang="ts">
	import { api, message, type Schemas } from '$lib/api';
	import { setLocale, t } from '$lib/i18n.svelte';
	import { applyAppearance, type Appearance } from '$lib/preferences';
	import ProviderSettings from '$lib/ProviderSettings.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Settings = Schemas['Settings'];
	let s = $state<Settings | null>(null);
	let error = $state('');
	let saving = $state(false);
	let budget = $state('2.00');
	let wip = $state('');

	async function load() {
		try {
			s = await api<Settings>('/v1/me/settings');
			budget = ((s.ai?.monthly_budget_cents ?? 200) / 100).toFixed(2);
			wip = s.templates.project.wip_limit?.toString() ?? '';
		} catch (err) {
			error = message(err);
		}
	}
	load();

	function apply(saved: Settings) {
		s = saved;
		setLocale(saved.language);
		applyAppearance(saved.appearance as Appearance);
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		if (!s) return;
		saving = true;
		try {
			s.templates.project.wip_limit = wip ? Number(wip) : null;
			const { language, appearance, templates } = s;
			apply(
				await api<Settings>('/v1/me/settings', {
					method: 'PUT',
					body: { language, appearance, templates }
				})
			);
			await api('/v1/me/agent-settings', {
				method: 'PUT',
				body: { monthly_budget_cents: Math.round(Number(budget) * 100) }
			});
			toast(t('Preferences saved.'));
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			saving = false;
		}
	}

	async function upload(e: Event) {
		const file = (e.currentTarget as HTMLInputElement).files?.[0];
		if (!file) return;
		try {
			const res = await fetch('/api/v1/me/settings.yaml', {
				method: 'PUT',
				headers: { 'Content-Type': 'application/yaml' },
				body: await file.text()
			});
			const out = await res.json();
			if (!res.ok) throw new Error(out.detail ?? res.statusText);
			apply(out as Settings);
			await load();
			toast(t('Settings file applied.'));
		} catch (err) {
			toast(err instanceof Error ? err.message : String(err), 'error');
		}
		(e.currentTarget as HTMLInputElement).value = '';
	}

	const choice = (pairs: [string, string][]) => pairs;
</script>

<svelte:head><title>{t('Preferences')} · Axon</title></svelte:head>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !s}
	<Status kind="loading" message="…" />
{:else}
	<form onsubmit={save}>
		<Section id="language" title={t('Language')}>
			<div class="field">
				<label for="pref-language">{t('Interface language')}</label>
				<select id="pref-language" bind:value={s.language}>
					<option value="en">English</option>
					<option value="fr">Français</option>
				</select>
			</div>
		</Section>

		<Section id="appearance" title={t('Appearance')}>
			<div class="grid">
				{#each [['theme', 'Theme', choice( [['system', 'Follow the system'], ['light', 'Light'], ['dark', 'Dark']] )], ['font', 'Font', choice( [['system', 'System sans-serif'], ['serif', 'Serif'], ['rounded', 'Rounded'], ['mono', 'Monospace']] )], ['font_size', 'Text size', choice( [['small', 'Small'], ['normal', 'Normal'], ['large', 'Large']] )], ['icons', 'Icons', choice( [['emoji', 'Emoji'], ['symbols', 'Symbols'], ['none', 'None']] )], ['motion', 'Motion', choice( [['system', 'Follow the system'], ['reduced', 'Reduced'], ['full', 'Full']] )]] as const as [key, label, options] (key)}
					<div class="field">
						<label for="pref-{key}">{t(label)}</label>
						<select id="pref-{key}" bind:value={s.appearance[key]}>
							{#each options as [v, l] (v)}<option value={v}>{t(l)}</option>{/each}
						</select>
					</div>
				{/each}
			</div>
		</Section>

		<Section id="templates" title={t('Templates')}>
			<div class="field wide">
				<label for="tpl-note">{t('New notes')}</label>
				<textarea id="tpl-note" rows="6" maxlength="20000" bind:value={s.templates.note}></textarea>
				<span class="muted small">{t('{title} is replaced by the note title.')}</span>
			</div>
			<h3>{t('New projects')}</h3>
			<div class="grid">
				<div class="field wide">
					<label for="tpl-project-description">{t('Description')}</label>
					<textarea
						id="tpl-project-description"
						rows="3"
						maxlength="20000"
						bind:value={s.templates.project.description}
					></textarea>
				</div>
				<div class="field">
					<label for="tpl-project-wip">{t('WIP limit for Doing')}</label>
					<input id="tpl-project-wip" type="number" min="1" max="50" bind:value={wip} />
				</div>
				<div class="field">
					<label for="tpl-project-docs">{t('Docs folder')}</label>
					<input
						id="tpl-project-docs"
						maxlength="200"
						bind:value={s.templates.project.docs_folder}
					/>
					<span class="muted small">{t('{slug} is replaced by the project name.')}</span>
				</div>
			</div>
			<h3>{t('New resources and skills')}</h3>
			<div class="grid">
				<div class="field">
					<label for="tpl-resource-section">{t('Section')}</label>
					<input
						id="tpl-resource-section"
						maxlength="200"
						bind:value={s.templates.resource.section}
					/>
				</div>
				<div class="field">
					<label for="tpl-resource-kind">{t('Kind')}</label>
					<select id="tpl-resource-kind" bind:value={s.templates.resource.kind}>
						<option value="hard">{t('Hard')}</option>
						<option value="soft">{t('Soft')}</option>
					</select>
				</div>
			</div>
		</Section>

		<Section id="ai" title={t('AI provider and budget')}>
			<div class="field">
				<label for="pref-budget">Monthly budget (US$)</label>
				<input id="pref-budget" type="number" min="0" max="1000" step="0.5" bind:value={budget} />
			</div>
			<ProviderSettings onchange={load} />
		</Section>

		<div class="actions"><Button type="submit" busy={saving}>{t('Save preferences')}</Button></div>
	</form>

	<Section
		id="file"
		title={t('Settings file')}
		description={t(
			'Download your settings as YAML, edit them in any editor, and upload them here. Keys are write-only: a downloaded file never contains one.'
		)}
	>
		<div class="row">
			<Button href="/api/v1/me/settings.yaml" variant="secondary"
				>{t('Download settings.yaml')}</Button
			>
			<label class="upload">
				{t('Upload a settings file')}
				<input type="file" accept=".yaml,.yml,application/yaml,text/yaml" onchange={upload} />
			</label>
		</div>
	</Section>
{/if}

<style>
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 14rem), 1fr));
		gap: var(--space-3) var(--space-4);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-3);
	}
	.field.wide {
		grid-column: 1 / -1;
	}
	label {
		font-weight: 600;
	}
	select,
	input,
	textarea {
		font: inherit;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	textarea {
		font-family: var(--font-mono);
	}
	h3 {
		font-size: var(--text-base);
		margin: var(--space-4) 0 var(--space-2);
	}
	.small {
		font-size: var(--text-sm);
		font-weight: 400;
	}
	.actions {
		margin: var(--space-4) 0 var(--space-6);
	}
	.row {
		display: flex;
		gap: var(--space-3);
		flex-wrap: wrap;
		align-items: center;
	}
	.upload {
		display: inline-grid;
		gap: var(--space-1);
	}
</style>
