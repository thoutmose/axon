<script lang="ts">
	import { install, promptInstall } from '$lib/install.svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { setLocale, t } from '$lib/i18n.svelte';
	import { applyAppearance, type Appearance } from '$lib/preferences';
	import ProviderSettings from '$lib/ProviderSettings.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Focus = {
		work_minutes: number;
		short_break_minutes: number;
		long_break_minutes: number;
		rounds: number;
		sessions: number;
		sound: 'none' | 'white' | 'pink' | 'brown' | 'rain';
		volume: number;
		music: { activity: string; url: string }[];
	};
	type Settings = Schemas['Settings'] & {
		keyboard: { shortcuts: boolean; vim: boolean };
		focus: Focus;
		gamification: { enabled: boolean };
		vault: { mirror: boolean; folder: string; health: boolean };
		learning: { new_per_day: number; retention: number; interleave: boolean; type_answer: boolean };
	};
	const learningDefaults = {
		new_per_day: 20,
		retention: 0.9,
		interleave: true,
		type_answer: false
	};
	const focusDefaults: Focus = {
		work_minutes: 25,
		short_break_minutes: 5,
		long_break_minutes: 15,
		rounds: 4,
		sessions: 1,
		sound: 'none',
		volume: 40,
		music: []
	};
	let s = $state<Settings | null>(null);
	let error = $state('');
	let saving = $state(false);
	let budget = $state('2.00');
	let wip = $state('');

	async function load() {
		try {
			const loaded = await api<Schemas['Settings']>('/v1/me/settings');
			s = {
				...loaded,
				appearance: { ...loaded.appearance, spaces: loaded.appearance.spaces ?? 'on' },
				keyboard: loaded.keyboard ?? { shortcuts: true, vim: false },
				focus: (loaded.focus as Focus | undefined) ?? focusDefaults,
				gamification: loaded.gamification ?? { enabled: false },
				vault: loaded.vault ?? { mirror: true, folder: 'Axon', health: false },
				learning: loaded.learning ?? learningDefaults
			};
			budget = ((s.ai?.monthly_budget_cents ?? 200) / 100).toFixed(2);
			wip = s.templates.project.wip_limit?.toString() ?? '';
		} catch (err) {
			error = message(err);
		}
	}
	load();

	function apply(saved: Schemas['Settings']) {
		s = {
			...saved,
			keyboard: saved.keyboard ?? { shortcuts: true, vim: false },
			focus: (saved.focus as Focus | undefined) ?? focusDefaults,
			gamification: saved.gamification ?? { enabled: false },
			vault: saved.vault ?? { mirror: true, folder: 'Axon', health: false },
			learning: saved.learning ?? learningDefaults
		};
		setLocale(saved.language);
		applyAppearance(saved.appearance as Appearance);
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		if (!s) return;
		saving = true;
		try {
			s.templates.project.wip_limit = wip ? Number(wip) : null;
			const { language, appearance, templates, keyboard, focus, gamification, vault, learning } = s;
			apply(
				await api<Settings>('/v1/me/settings', {
					method: 'PUT',
					body: {
						language,
						appearance,
						templates,
						keyboard,
						focus,
						gamification,
						vault,
						learning
					}
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
				{#each [['theme', 'Theme', choice( [['system', 'Follow the system'], ['light', 'Light'], ['dark', 'Dark']] )], ['font', 'Font', choice( [['system', 'System sans-serif'], ['serif', 'Serif'], ['rounded', 'Rounded'], ['mono', 'Monospace']] )], ['font_size', 'Text size', choice( [['small', 'Small'], ['normal', 'Normal'], ['large', 'Large']] )], ['icons', 'Icons', choice( [['emoji', 'Emoji'], ['symbols', 'Symbols'], ['none', 'None']] )], ['motion', 'Motion', choice( [['system', 'Follow the system'], ['reduced', 'Reduced'], ['full', 'Full']] )], ['spaces', 'Themed spaces', choice( [['on', 'On: a notebook for notes, a library for learning…'], ['off', 'Off']] )]] as const as [key, label, options] (key)}
					<div class="field">
						<label for="pref-{key}">{t(label)}</label>
						<select id="pref-{key}" bind:value={s.appearance[key]}>
							{#each options as [v, l] (v)}<option value={v}>{t(l)}</option>{/each}
						</select>
					</div>
				{/each}
			</div>
		</Section>

		<Section id="keyboard" title={t('Keyboard')}>
			<label class="check"
				><input type="checkbox" bind:checked={s.keyboard.shortcuts} />
				{t('Single-key shortcuts (n, /, ?, g then a letter)')}</label
			>
			<label class="check"
				><input type="checkbox" bind:checked={s.keyboard.vim} />
				{t('Vim keybindings in the note editor')}</label
			>
			<p class="muted small">
				{t('Turn single-key shortcuts off, or Vim mode on, in Preferences.')}
			</p>
		</Section>

		<Section id="gamification" title={t('Progress rewards')}>
			<label class="check"
				><input type="checkbox" bind:checked={s.gamification.enabled} />
				{t('Earn points, levels, badges, and a daily streak from everything you do')}</label
			>
			<p class="muted small">
				{t(
					'Off hides every surface. No notifications, no shame for missed days, nothing bought with money.'
				)}
				<a href="/progress">{t('How points are earned')}</a>
			</p>
		</Section>

		<Section id="app" title={t('App')}>
			{#if install.installed}
				<p>{t('Axon is installed on this device.')}</p>
			{:else if install.offer}
				<Button variant="secondary" onclick={promptInstall}
					>{t('Install Axon on this device')}</Button
				>
			{:else}
				<p class="muted small">
					{t(
						'Install Axon to open it like an app, even offline to capture a thought: on iPhone or iPad, Safari → Share → Add to Home Screen; in Chrome or Edge, the install icon in the address bar.'
					)}
				</p>
			{/if}
		</Section>

		<Section id="vault" title={t('Markdown vault')}>
			<label class="check"
				><input type="checkbox" bind:checked={s.vault.mirror} />
				{t(
					'Keep tasks, goals, milestones, projects, areas, resources, habits, and media as notes in my vault'
				)}</label
			>
			<div class="field">
				<label for="vault-folder">{t('Folder')}</label>
				<input id="vault-folder" bind:value={s.vault.folder} maxlength="100" required />
			</div>
			<label class="check"
				><input type="checkbox" bind:checked={s.vault.health} disabled={!s.vault.mirror} />
				{t('Also write my food and training logs there (health data, one way)')}</label
			>
			<p class="muted small">
				{t(
					'Each item is a note with its fields in the front matter and its text in sections. Edit either side and the other follows; a new note with a type creates the item; deleting the file deletes it; moving it into Archive/ archives it. Health logs also need the health consent and are removed from the vault if you withdraw it.'
				)}
				<a href="/notes/templates">{t('See the templates')}</a>
			</p>
		</Section>

		<Section id="learning" title={t('Learning')}>
			<div class="grid">
				<div class="field">
					<label for="new-per-day">{t('New flashcards a day')}</label>
					<input
						id="new-per-day"
						type="number"
						min="0"
						max="500"
						bind:value={s.learning.new_per_day}
						required
					/>
				</div>
				<div class="field">
					<label for="retention"
						>{t('Retention to aim for')}: {Math.round(s.learning.retention * 100)}%</label
					>
					<input
						id="retention"
						type="range"
						min="0.7"
						max="0.97"
						step="0.01"
						bind:value={s.learning.retention}
					/>
				</div>
			</div>
			<label class="check"
				><input type="checkbox" bind:checked={s.learning.interleave} />
				{t('Mix decks when reviewing everything (interleaving)')}</label
			>
			<label class="check"
				><input type="checkbox" bind:checked={s.learning.type_answer} />
				{t('Write my answer before it is shown')}</label
			>
			<p class="muted small">
				{t(
					'A few new cards a day keep reviews steady. Higher retention means more reviews; 90% is a good balance. Mixing decks and writing answers make reviews harder, and memories stronger.'
				)}
			</p>
		</Section>

		<Section id="focus" title={t('Focus')}>
			<div class="grid">
				{#each [['work_minutes', 'Focus minutes', 180], ['short_break_minutes', 'Short break minutes', 60], ['long_break_minutes', 'Long break minutes', 120], ['rounds', 'Rounds before the long break', 12], ['sessions', 'Sessions in a row', 12]] as const as [key, label, max] (key)}
					<div class="field">
						<label for="focus-{key}">{t(label)}</label>
						<input id="focus-{key}" type="number" min="1" {max} bind:value={s.focus[key]} />
					</div>
				{/each}
				<div class="field">
					<label for="focus-sound-pref">{t('Focus sound')}</label>
					<select id="focus-sound-pref" bind:value={s.focus.sound}>
						<option value="none">None</option>
						<option value="white">White noise</option>
						<option value="pink">Pink noise</option>
						<option value="brown">Brown noise</option>
						<option value="rain">Rain</option>
					</select>
				</div>
				<div class="field">
					<label for="focus-volume">{t('Volume')}</label>
					<input id="focus-volume" type="range" min="0" max="100" bind:value={s.focus.volume} />
				</div>
			</div>
			<h3>{t('Music for focus')}</h3>
			<p class="muted small">
				{t(
					'Links to playlists or streams you like for each kind of work. They open from the Focus page; nothing plays inside Axon.'
				)}
			</p>
			{#each s.focus.music as m, i (i)}
				<div class="music">
					<input
						aria-label="Activity"
						placeholder="coding"
						bind:value={m.activity}
						maxlength="100"
					/>
					<input aria-label="Link" type="url" placeholder="https://…" bind:value={m.url} />
					<Button
						variant="ghost"
						onclick={() => s && (s.focus.music = s.focus.music.filter((_, j) => j !== i))}
						aria-label="Remove {m.activity || 'link'}">✕</Button
					>
				</div>
			{/each}
			<Button
				variant="secondary"
				onclick={() => s && (s.focus.music = [...s.focus.music, { activity: '', url: '' }])}
				>{t('Add a music link')}</Button
			>
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
	.check {
		display: flex;
		gap: var(--space-2);
		align-items: baseline;
		font-weight: 400;
		margin-bottom: var(--space-2);
	}
	.music {
		display: grid;
		grid-template-columns: minmax(6rem, 10rem) 1fr auto;
		gap: var(--space-2);
		margin-bottom: var(--space-2);
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
