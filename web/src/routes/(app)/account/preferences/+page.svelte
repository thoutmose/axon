<script lang="ts">
	import { install, promptInstall } from '$lib/install.svelte';
	import { disablePush, enablePush } from '$lib/reminders';
	import EmbeddingSettings from '$lib/EmbeddingSettings.svelte';
	import VaultServer from '$lib/VaultServer.svelte';
	import AudiobookshelfSettings from '$lib/AudiobookshelfSettings.svelte';
	import ChannelSettings from '$lib/ChannelSettings.svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { setLocale, t } from '$lib/i18n.svelte';
	import { applyAppearance, type Appearance } from '$lib/preferences';
	import { setHidden } from '$lib/menu.svelte';
	import { PAGES, pageLabels } from '$lib/uses';
	import { soundOptions, type Sound } from '$lib/sounds';
	import ProviderSettings from '$lib/ProviderSettings.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Scenery from '$lib/ui/Scenery.svelte';
	import { skyState } from '$lib/skyState.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Focus = {
		work_minutes: number;
		short_break_minutes: number;
		long_break_minutes: number;
		rounds: number;
		sessions: number;
		sound: Sound;
		volume: number;
		music: { activity: string; url: string }[];
	};
	type Settings = Schemas['Settings'] & {
		keyboard: { shortcuts: boolean; vim: boolean };
		caffeine: { bedtime: string; metabolism: 'fast' | 'average' | 'slow' };
		focus: Focus;
		gamification: { enabled: boolean };
		vault: { mirror: boolean; folder: string; health: boolean };
		learning: { new_per_day: number; retention: number; interleave: boolean; type_answer: boolean };
		reminders: {
			enabled: boolean;
			morning: string;
			evening: string;
			email: string;
			weekly_day:
				'sunday' | 'monday' | 'tuesday' | 'wednesday' | 'thursday' | 'friday' | 'saturday';
			weekly_at: string;
			monthly_day: number;
			monthly_at: string;
		};
		today: { sections: string[] };
		menu: { hidden: string[]; chosen: boolean };
	};
	const remindersDefaults: Settings['reminders'] = {
		enabled: false,
		morning: '08:00',
		evening: '20:00',
		email: '',
		weekly_day: 'sunday',
		weekly_at: '18:00',
		monthly_day: 1,
		monthly_at: '09:00'
	};
	type ReminderStatus = {
		public_key: string;
		email: boolean;
		devices: { id: string; device: string; created_at: string; last_sent_at: string | null }[];
	};
	let reminderStatus = $state<ReminderStatus | null>(null);
	async function loadReminders() {
		reminderStatus = await api<ReminderStatus>('/v1/me/reminders').catch(() => null);
	}
	loadReminders();
	async function pushOn() {
		try {
			await enablePush(reminderStatus!.public_key);
			if (s && !s.reminders.enabled) s.reminders.enabled = true;
			toast(t('Notifications are on for this device. Save to keep the times.'));
			await loadReminders();
		} catch (err) {
			toast(err instanceof Error ? err.message : message(err), 'error');
		}
	}
	async function pushOff(id: string) {
		try {
			await disablePush(id);
			await loadReminders();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	async function testReminder() {
		try {
			const r = await api<{ devices: number; email: boolean; channels: number }>(
				'/v1/me/reminders/test',
				{ method: 'POST' }
			);
			toast(
				t('Sent to {devices} device(s){email}.', {
					devices: String(r.devices),
					email: r.email ? t(' and by email') : ''
				}) + (r.channels ? ' ' + t('Also to {n} channel(s).', { n: r.channels }) : '')
			);
		} catch (err) {
			toast(message(err), 'error');
		}
	}
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
	// A look at the sky later today, to choose a landscape (EP-96).
	let previewHours = $state(0);
	const previewAt = $derived(new Date(skyState.now.getTime() + previewHours * 3_600_000));
	let error = $state('');
	let saving = $state(false);
	let budget = $state('2.00');
	let wip = $state('');

	async function load() {
		try {
			const loaded = await api<Schemas['Settings']>('/v1/me/settings');
			s = {
				...loaded,
				appearance: {
					...loaded.appearance,
					spaces: loaded.appearance.spaces ?? 'on',
					scenery: loaded.appearance.scenery ?? 'mountains'
				},
				keyboard: loaded.keyboard ?? { shortcuts: true, vim: false },
				focus: (loaded.focus as Focus | undefined) ?? focusDefaults,
				gamification: loaded.gamification ?? { enabled: false },
				vault: loaded.vault ?? { mirror: true, folder: t('Axon'), health: false },
				learning: loaded.learning ?? learningDefaults,
				reminders: { ...remindersDefaults, ...loaded.reminders },
				today: loaded.today ?? {
					sections: ['kpis', 'tasks', 'milestones', 'goals', 'notes', 'overview']
				},
				menu: loaded.menu ?? { hidden: [], chosen: false },
				caffeine: { bedtime: '23:00', metabolism: 'average', ...loaded.caffeine }
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
			vault: saved.vault ?? { mirror: true, folder: t('Axon'), health: false },
			learning: saved.learning ?? learningDefaults,
			reminders: { ...remindersDefaults, ...saved.reminders },
			today: saved.today ?? s!.today,
			menu: saved.menu ?? s!.menu,
			caffeine: { bedtime: '23:00', metabolism: 'average', ...saved.caffeine }
		};
		setHidden(s.menu.hidden);
		setLocale(saved.language);
		applyAppearance(saved.appearance as Appearance);
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		if (!s) return;
		saving = true;
		try {
			s.templates.project.wip_limit = wip ? Number(wip) : null;
			const {
				language,
				appearance,
				templates,
				keyboard,
				focus,
				gamification,
				vault,
				learning,
				reminders,
				today,
				menu
			} = s;
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
						learning,
						reminders,
						// Sent back as they are: a part left out would be reset to its default.
						today,
						menu
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
				{#each [['theme', t('Theme'), choice( [['system', t('Follow the system')], ['sun', t('Follow the sun and the moon')], ['light', t('Light')], ['dark', t('Dark')]] )], ['font', t('Font'), choice( [['system', t('System sans-serif')], ['serif', t('Serif')], ['rounded', t('Rounded')], ['mono', t('Monospace')]] )], ['font_size', t('Text size'), choice( [['small', t('Small')], ['normal', t('Normal')], ['large', t('Large')]] )], ['icons', t('Icons'), choice( [['emoji', t('Emoji')], ['symbols', t('Symbols')], ['none', t('None')]] )], ['motion', t('Motion'), choice( [['system', t('Follow the system')], ['reduced', t('Reduced')], ['full', t('Full')]] )], ['spaces', t('Themed spaces'), choice( [['on', t('On: a notebook for notes, a library for learning…')], ['off', t('Off')]] )], ['scenery', t('Landscape under the sky'), choice( [['mountains', t('Mountains')], ['forest', t('Forest')], ['ocean', t('Ocean')], ['desert', t('Desert')], ['earth', t('The Earth from orbit')], ['eiffel-tower', t('Eiffel Tower, Paris (Europe)')], ['pyramids', t('Pyramids of Giza (Africa)')], ['taj-mahal', t('Taj Mahal, Agra (Asia)')], ['statue-of-liberty', t('Statue of Liberty, New York (North America)')], ['christ-the-redeemer', t('Christ the Redeemer, Rio (South America)')], ['opera-house', t('Sydney Opera House (Oceania)')]] )]] as const as [key, label, options] (key)}
					<div class="field">
						<label for="pref-{key}">{t(label)}</label>
						<select id="pref-{key}" bind:value={s.appearance[key]}>
							{#each options as [v, l] (v)}<option value={v}>{t(l)}</option>{/each}
						</select>
					</div>
				{/each}
			</div>
			<figure class="sky-preview">
				<div class="frame"><Scenery landscape={s.appearance.scenery} at={previewAt} /></div>
				<figcaption>
					<label
						>{t('Preview the sky in {hours} h', { hours: previewHours })}
						<input type="range" min="0" max="23" bind:value={previewHours} /></label
					>
					<span class="muted small"
						>{t('The sun and the moon where they are in your time zone ({zone}).', {
							zone: skyState.zone
						})}</span
					>
				</figcaption>
			</figure>
		</Section>

		<Section id="menu" title={t('Pages in the menu')}>
			<fieldset class="pages">
				<legend class="visually-hidden">{t('Pages in the menu')}</legend>
				{#each PAGES as p (p)}
					<label class="check"
						><input
							type="checkbox"
							checked={!s.menu.hidden.includes(p)}
							onchange={(e) => {
								const on = e.currentTarget.checked;
								s!.menu.hidden = PAGES.filter((x) => (x === p ? !on : s!.menu.hidden.includes(x)));
							}}
						/>
						{t(pageLabels[p])}</label
					>
				{/each}
			</fieldset>
			<p class="muted small">
				{t(
					'Show only what you use. A hidden page still opens from a link or a search, and nothing in it is deleted.'
				)}
			</p>
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
			<p class="small">
				<a href="/share">{t('Save to Axon from other apps and any web page')}</a>
			</p>
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

		<Section id="reminders" title={t('Reminders and digests')}>
			<label class="check"
				><input type="checkbox" bind:checked={s.reminders.enabled} />
				{t('Send me digests and reminders')}</label
			>
			<div class="grid">
				<div class="field">
					<label for="rem-morning">{t('Morning digest at')}</label>
					<input id="rem-morning" type="time" bind:value={s.reminders.morning} />
				</div>
				<div class="field">
					<label for="rem-evening">{t('Evening nudge at')}</label>
					<input id="rem-evening" type="time" bind:value={s.reminders.evening} />
				</div>
				<div class="field">
					<label for="rem-weekly-day">{t('Weekly digest on')}</label>
					<select id="rem-weekly-day" bind:value={s.reminders.weekly_day}>
						{#each ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'] as d (d)}
							<option value={d}>{t(d[0].toUpperCase() + d.slice(1))}</option>
						{/each}
					</select>
				</div>
				<div class="field">
					<label for="rem-weekly-at">{t('at')}</label>
					<input id="rem-weekly-at" type="time" bind:value={s.reminders.weekly_at} />
				</div>
				<div class="field">
					<label for="rem-monthly-day">{t('Monthly digest on day')}</label>
					<input
						id="rem-monthly-day"
						type="number"
						min="1"
						max="28"
						bind:value={s.reminders.monthly_day}
					/>
				</div>
				<div class="field">
					<label for="rem-monthly-at">{t('at')}</label>
					<input id="rem-monthly-at" type="time" bind:value={s.reminders.monthly_at} />
				</div>
			</div>
			<p class="muted small">
				{t(
					'The daily digest says what today holds and what comes next: tasks, milestones, habits, cards to review, notes to recall, with a line to carry and a picture of the day. The weekly and monthly ones look back and ahead. The nudge names the habits still open. Nothing is sent on days with nothing to say. Clear a time to skip it.'
				)}
				<a href="/digest">{t('See your digest')}</a>
			</p>
			{#if reminderStatus}
				<h3 class="sub">{t('Devices')}</h3>
				{#if reminderStatus.devices.length}
					<ul class="devices">
						{#each reminderStatus.devices as d (d.id)}
							<li>
								{d.device || t('A device')}
								<Button
									variant="ghost"
									onclick={() => pushOff(d.id)}
									aria-label="{t('Stop on')} {d.device}">{t('Stop')}</Button
								>
							</li>
						{/each}
					</ul>
				{/if}
				<div class="row">
					<Button variant="secondary" onclick={pushOn}>{t('Notify me on this device')}</Button>
					<Button variant="ghost" onclick={testReminder}>{t('Send a test')}</Button>
				</div>
				<p class="muted small">
					{t(
						'Notifications are encrypted for your device; your browser’s push service (Google, Mozilla, Apple, or Microsoft) passes them on without being able to read them. On iPhone and iPad, install Axon first.'
					)}
				</p>
				{#if reminderStatus.email}
					<div class="field">
						<label for="rem-email">{t('Also by email to')}</label>
						<input
							id="rem-email"
							type="email"
							maxlength="254"
							autocomplete="email"
							bind:value={s.reminders.email}
						/>
					</div>
				{/if}
			{/if}
		</Section>

		<Section id="channels" title={t('Chat apps')}>
			<ChannelSettings />
		</Section>

		<Section id="audiobookshelf" title={t('Podcasts from Audiobookshelf')}>
			<AudiobookshelfSettings />
		</Section>
		<Section id="meaning" title={t('Search by meaning')}>
			<EmbeddingSettings />
		</Section>

		<Section id="vault" title={t('Markdown vault')}>
			<VaultServer />
			<h3 class="sub">{t('Items as notes')}</h3>
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
				{t(
					'Also write my food and training logs and my journal there (health data, one way)'
				)}</label
			>
			<p class="muted small">
				{t(
					'Each item is a note with its fields in the front matter and its text in sections. Edit either side and the other follows; a new note with a type creates the item; deleting the file deletes it; moving it into Archive/ archives it. Health logs also need the health consent and are removed from the vault if you withdraw it.'
				)}
				<a href="/notes/templates">{t('See the templates')}</a>
			</p>
		</Section>

		<Section id="caffeine" title={t('Coffee and tea')}>
			<div class="grid">
				<div class="field">
					<label for="bedtime">{t('Usual bedtime')}</label>
					<input id="bedtime" type="time" bind:value={s.caffeine.bedtime} required />
				</div>
				<div class="field">
					<label for="metabolism">{t('How fast you clear caffeine')}</label>
					<select id="metabolism" bind:value={s.caffeine.metabolism}>
						<option value="fast">{t('Fast (half-life about 3 h, for example smokers)')}</option>
						<option value="average">{t('Average (about 5 h)')}</option>
						<option value="slow"
							>{t('Slow (about 7 h, for example in pregnancy or with the pill)')}</option
						>
					</select>
				</div>
			</div>
			<p class="muted small">
				{t(
					'Used to tell how much caffeine is left at bedtime, and the last coffee that keeps your sleep intact.'
				)}
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
				{#each [['work_minutes', t('Focus minutes'), 180], ['short_break_minutes', t('Short break minutes'), 60], ['long_break_minutes', t('Long break minutes'), 120], ['rounds', t('Rounds before the long break'), 12], ['sessions', t('Sessions in a row'), 12]] as const as [key, label, max] (key)}
					<div class="field">
						<label for="focus-{key}">{t(label)}</label>
						<input id="focus-{key}" type="number" min="1" {max} bind:value={s.focus[key]} />
					</div>
				{/each}
				<div class="field">
					<label for="focus-sound-pref">{t('Focus sound')}</label>
					<select id="focus-sound-pref" bind:value={s.focus.sound}>
						<option value="none">{t('No sound')}</option>
						{#each soundOptions as g (g.group)}
							<optgroup label={t(g.group)}>
								{#each g.items as [value, label] (value)}
									<option {value}>{t(label)}</option>
								{/each}
							</optgroup>
						{/each}
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
					'Axon composes its own music as you work, with nothing streamed. You can also keep links to playlists you like for each kind of work; they open from the Focus page.'
				)}
			</p>
			{#each s.focus.music as m, i (i)}
				<div class="music">
					<input
						aria-label={t('Activity')}
						placeholder={t('coding')}
						bind:value={m.activity}
						maxlength="100"
					/>
					<input aria-label={t('Link')} type="url" placeholder="https://…" bind:value={m.url} />
					<Button
						variant="ghost"
						onclick={() => s && (s.focus.music = s.focus.music.filter((_, j) => j !== i))}
						aria-label={t('Remove {value}', { value: m.activity || 'link' })}>✕</Button
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
				<label for="pref-budget">{t('Monthly budget (US$)')}</label>
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
	.sub {
		font-size: var(--text-base);
		margin: var(--space-3) 0 var(--space-1);
	}
	.devices {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.devices li {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
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
	.pages {
		border: 0;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
		gap: var(--space-2);
	}
	.sky-preview {
		margin: var(--space-4) 0 0;
	}
	.sky-preview .frame {
		position: relative;
		height: 10rem;
		border-radius: var(--radius-lg);
		overflow: hidden;
		border: 1px solid var(--border);
	}
	.sky-preview figcaption {
		display: grid;
		gap: var(--space-1);
		margin-top: var(--space-2);
	}
</style>
