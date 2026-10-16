<script lang="ts">
	import { onDestroy } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import { hostOf, pickVoices, voiceLabel, type NaturalVoice } from '$lib/podcastVoices';

	// The podcast of the news (EP-113), off until turned on: an episode made
	// from the period's best news, read aloud by the natural voices of the
	// operator's speech server (EP-132), alone or as two hosts taking turns,
	// else by this device's own voices.
	type Episode = Schemas['WatchEpisode'];
	type Podcast = {
		enabled: boolean;
		format: 'briefing' | 'digest' | 'deep';
		lang: 'en' | 'fr';
		minutes: number;
		days: number;
		voice: string;
		rate: number;
		hosts: number;
		voice2: string;
	};

	let settings = $state<Podcast | null>(null);
	let episode = $state<Episode | null>(null);
	let error = $state('');
	let loading = $state(false);
	let voices = $state<SpeechSynthesisVoice[]>([]);
	let current = $state(-1); // the part being read: 0 intro, 1… chapters, last outro
	let playing = $state(false);
	const speech =
		typeof window !== 'undefined' && 'speechSynthesis' in window ? window.speechSynthesis : null;
	let natural = $state<{ available: boolean; voices: NaturalVoice[] }>({
		available: false,
		voices: []
	});
	api<typeof natural>('/v1/watch/podcast/voices')
		.then((r) => (natural = r))
		.catch(() => {});
	const useNatural = $derived(
		natural.available && !!settings && natural.voices.some((v) => v.lang === settings!.lang)
	);
	const hosts = $derived(settings?.hosts === 2 ? 2 : 1);
	const hostKeys = $derived(hosts === 2 ? (['voice', 'voice2'] as const) : (['voice'] as const));
	// The chosen natural voices, or the server's best for the language.
	$effect(() => {
		if (!useNatural || !settings) return;
		const [a, b] = pickVoices(settings.lang, natural.voices, [settings.voice, settings.voice2]);
		if (settings.voice !== a) settings.voice = a;
		if (settings.voice2 !== b) settings.voice2 = b;
	});
	const canPlay = $derived(useNatural || !!speech);
	let audio: HTMLAudioElement | null = null;
	let preparing = $state(false);
	const spoken = new Map<string, Promise<string>>(); // a part's audio, by part, voice, and pace
	function forget() {
		for (const p of spoken.values()) p.then((u) => URL.revokeObjectURL(u)).catch(() => {});
		spoken.clear();
	}

	async function loadSettings() {
		try {
			const doc = await api<{ podcast: Podcast }>('/v1/me/settings');
			settings = doc.podcast;
			if (settings.enabled) generate();
		} catch (err) {
			error = message(err);
		}
	}
	loadSettings();

	async function save(next: Podcast, quiet = false) {
		try {
			const doc = await api<Record<string, unknown>>('/v1/me/settings');
			delete doc.ai;
			await api('/v1/me/settings', { method: 'PUT', body: { ...doc, podcast: next } });
			settings = next;
			if (!quiet) toast(t('Podcast settings saved.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	async function turnOn() {
		if (!settings) return;
		await save({ ...settings, enabled: true }, true);
		generate();
	}

	async function generate() {
		if (!settings) return;
		stop();
		forget();
		loading = true;
		error = '';
		try {
			const q = new URLSearchParams({
				format: settings.format,
				lang: settings.lang,
				minutes: String(settings.minutes),
				days: String(settings.days)
			});
			episode = await api<Episode>(`/v1/watch/podcast?${q}`);
		} catch (err) {
			error = message(err);
		} finally {
			loading = false;
		}
	}

	function loadVoices() {
		voices = speech?.getVoices() ?? [];
	}
	if (speech) {
		loadVoices();
		speech.addEventListener('voiceschanged', loadVoices);
	}
	const forLang = $derived(
		voices.filter((v) => settings && v.lang.toLowerCase().startsWith(settings.lang))
	);

	const parts = $derived(
		episode ? [episode.intro, ...episode.chapters.map((c) => c.text), episode.outro] : []
	);
	const voiceOf = (i: number) =>
		hostOf(i, parts.length, hosts) === 1 ? settings!.voice2 : settings!.voice;

	function partAudio(i: number): Promise<string> {
		const voice = voiceOf(i);
		const key = `${i}|${voice}|${settings!.rate}`;
		let p = spoken.get(key);
		if (!p) {
			p = fetch('/api/v1/watch/podcast/speech', {
				method: 'POST',
				credentials: 'same-origin',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ text: parts[i], voice, speed: settings!.rate })
			}).then(async (r) => {
				if (!r.ok)
					throw new Error((await r.json().catch(() => null))?.detail ?? `HTTP ${r.status}`);
				return URL.createObjectURL(await r.blob());
			});
			p.catch(() => spoken.delete(key));
			spoken.set(key, p);
		}
		return p;
	}

	async function speakNatural(i: number) {
		preparing = true;
		try {
			const url = await partAudio(i);
			if (current !== i || !playing) return;
			audio ??= new Audio();
			audio.src = url;
			audio.onended = () => {
				if (playing && current === i) speak(i + 1);
			};
			await audio.play();
			// The next part is spoken while this one plays.
			if (i + 1 < parts.length) partAudio(i + 1).catch(() => {});
		} catch (err) {
			playing = false;
			toast(err instanceof Error ? err.message : t('Something went wrong. Try again.'), 'error');
		} finally {
			preparing = false;
		}
	}

	function speak(i: number) {
		if (!canPlay || !settings || i >= parts.length) {
			playing = false;
			current = -1;
			return;
		}
		current = i;
		if (useNatural) {
			speech?.cancel();
			playing = true;
			speakNatural(i);
			return;
		}
		if (!speech) return;
		const u = new SpeechSynthesisUtterance(parts[i]);
		u.lang = settings.lang === 'fr' ? 'fr-FR' : 'en-US';
		u.rate = settings.rate;
		const v = voices.find((x) => x.name === voiceOf(i));
		if (v) u.voice = v;
		u.onend = () => {
			if (playing && current === i) speak(i + 1);
		};
		speech.cancel();
		speech.speak(u);
		playing = true;
	}
	function play() {
		if (useNatural) {
			if (audio?.src && audio.paused && current >= 0) {
				playing = true;
				audio.play().catch(() => (playing = false));
			} else speak(Math.max(0, current));
			return;
		}
		if (!speech) return;
		if (speech.paused) {
			speech.resume();
			playing = true;
		} else speak(Math.max(0, current));
	}
	function pause() {
		if (useNatural) audio?.pause();
		else speech?.pause();
		playing = false;
	}
	function stop() {
		playing = false;
		current = -1;
		speech?.cancel();
		audio?.pause();
	}
	onDestroy(() => {
		stop();
		forget();
	});

	function download() {
		if (!episode) return;
		const text = [episode.title, '', ...parts].join('\n\n');
		const a = document.createElement('a');
		a.href = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }));
		a.download = `axon-podcast-${episode.to.slice(0, 10)}.txt`;
		a.click();
		URL.revokeObjectURL(a.href);
	}
	const formats: [Podcast['format'], string][] = [
		['briefing', 'Briefing: a sentence a story'],
		['digest', 'Digest: a few sentences a story'],
		['deep', 'Deep dive: by topic, more of each']
	];
</script>

<svelte:head><title>{t('Podcast')} · Axon</title></svelte:head>

<p><a href="/watch">← {t('Tech watch')}</a></p>
<h1>{t('Podcast')}</h1>

{#if error && !episode}
	<Status kind="error" message={error} retry={settings?.enabled ? generate : loadSettings} />
{:else if !settings}
	<Status kind="loading" message={t('Loading…')} />
{:else if !settings.enabled}
	<section class="off">
		<p>
			{t(
				'Listen to your tech watch: an episode made from the best news of the day or the week, read aloud by this device. Nothing leaves Axon to make it.'
			)}
		</p>
		<p class="muted small">
			{t('The podcast is off. Turn it on to choose its format, language, length, and voice.')}
		</p>
		<Button onclick={turnOn}>{t('Turn on the podcast')}</Button>
	</section>
{:else}
	<div class="layout">
		<form
			class="settings"
			onsubmit={(e) => {
				e.preventDefault();
				save(settings!);
				generate();
			}}
		>
			<label
				>{t('Format')}
				<select bind:value={settings.format}>
					{#each formats as [v, label] (v)}<option value={v}>{t(label)}</option>{/each}
				</select></label
			>
			<label
				>{t('Language')}
				<select bind:value={settings.lang}>
					<option value="en">English</option>
					<option value="fr">Français</option>
				</select></label
			>
			<label
				>{t('Length: about {n} minutes', { n: settings.minutes })}
				<input type="range" min="3" max="30" step="1" bind:value={settings.minutes} /></label
			>
			<label
				>{t('Period')}
				<select bind:value={settings.days}>
					<option value={1}>{t('The last day')}</option>
					<option value={7}>{t('The last week')}</option>
				</select></label
			>
			<label
				>{t('Hosts')}
				<select bind:value={settings.hosts}>
					<option value={1}>{t('One host')}</option>
					<option value={2}>{t('A conversation: two hosts take turns')}</option>
				</select></label
			>
			{#each hostKeys as key, n (key)}
				<label
					>{hosts === 2 ? t('Voice of host {n}', { n: n + 1 }) : t('Voice')}
					{#if useNatural}
						<select bind:value={settings[key]}>
							{#each natural.voices.filter((v) => v.lang === settings!.lang) as v (v.id)}
								<option value={v.id}>{voiceLabel(v.id)}</option>
							{/each}
						</select>
					{:else}
						<select bind:value={settings[key]} disabled={!speech}>
							<option value="">{t('This device’s default')}</option>
							{#each forLang as v (v.name)}<option value={v.name}>{v.name}</option>{/each}
						</select>
					{/if}
				</label>
			{/each}
			<p class="muted small">
				{useNatural
					? t('Natural voices from your speech server.')
					: t(
							'This device’s voices. For natural voices, the operator runs a speech server such as Kokoro and sets AXON_TTS_URL.'
						)}
			</p>
			<label
				>{t('Pace: {n}×', { n: settings.rate.toFixed(1) })}
				<input type="range" min="0.5" max="2" step="0.1" bind:value={settings.rate} /></label
			>
			<div class="row">
				<Button type="submit">{t('Save and make the episode')}</Button>
				<Button variant="ghost" onclick={() => save({ ...settings!, enabled: false })}
					>{t('Turn off')}</Button
				>
			</div>
		</form>

		<section class="player" aria-labelledby="episode-title">
			{#if loading}
				<Status kind="loading" message={t('Making the episode…')} />
			{:else if episode}
				<h2 id="episode-title">{episode.title}</h2>
				<p class="muted small">
					{t(
						episode.chapters.length === 1
							? '1 story · about {m} min'
							: '{n} stories · about {m} min',
						{
							n: episode.chapters.length,
							m: Math.max(1, Math.round(episode.minutes))
						}
					)}
				</p>
				{#if preparing}<p class="small muted" aria-live="polite">
						{t('Preparing the voice…')}
					</p>{/if}
				{#if !canPlay}
					<p class="small">
						{t(
							'This browser cannot read aloud. Download the script, or open Axon in another browser.'
						)}
					</p>
				{/if}
				<div class="row controls">
					{#if playing}
						<Button onclick={pause}>{t('Pause')}</Button>
					{:else}
						<Button onclick={play} disabled={!canPlay || !parts.length}>{t('Play')}</Button>
					{/if}
					<Button
						variant="secondary"
						onclick={() => speak(Math.max(0, current - 1))}
						disabled={!canPlay || current <= 0}>{t('Previous')}</Button
					>
					<Button
						variant="secondary"
						onclick={() => speak(current + 1)}
						disabled={!canPlay || current < 0}>{t('Next')}</Button
					>
					<Button variant="ghost" onclick={stop} disabled={current < 0}>{t('Stop')}</Button>
					<Button variant="ghost" onclick={download}>{t('Download the script')}</Button>
				</div>
				<ol class="chapters" aria-live="polite">
					<li class:now={current === 0}>
						<button type="button" class="link" onclick={() => speak(0)} disabled={!canPlay}
							>{t('Introduction')}</button
						>
					</li>
					{#each episode.chapters as c, i (i)}
						<li class:now={current === i + 1}>
							<button type="button" class="link" onclick={() => speak(i + 1)} disabled={!canPlay}
								>{c.title}</button
							>
							<span class="muted small"
								>{c.source}{#if c.topic}
									· {c.topic}{/if} · <a href="/watch?item={c.item_id}">{t('Open')}</a></span
							>
							{#if current === i + 1}<p class="small">{c.text}</p>{/if}
						</li>
					{:else}
						<li class="muted">{t('No news in this period yet.')}</li>
					{/each}
					<li class:now={current === parts.length - 1 && current > 0}>
						<button
							type="button"
							class="link"
							onclick={() => speak(parts.length - 1)}
							disabled={!canPlay}>{t('Closing')}</button
						>
					</li>
				</ol>
				<p class="muted small">
					{t('Each episode ends by saying it was put together by Axon Project.')}
				</p>
			{/if}
		</section>
	</div>
{/if}

<style>
	.small {
		font-size: var(--text-sm);
	}
	.off {
		max-width: 40rem;
	}
	.layout {
		display: grid;
		grid-template-columns: minmax(min(100%, 18rem), 1fr) minmax(0, 2fr);
		gap: var(--space-6);
		align-items: start;
	}
	@media (max-width: 50rem) {
		.layout {
			grid-template-columns: 1fr;
		}
	}
	.settings {
		display: grid;
		gap: var(--space-3);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	.settings label {
		display: grid;
		gap: var(--space-1);
		font-size: var(--text-sm);
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.player h2 {
		margin-top: 0;
	}
	.chapters {
		display: grid;
		gap: var(--space-2);
		padding-left: var(--space-6);
	}
	.chapters li.now {
		font-weight: 600;
	}
	.chapters li.now p {
		font-weight: normal;
	}
	.chapters span {
		display: block;
	}
	button.link {
		background: none;
		border: 0;
		padding: 0;
		font: inherit;
		color: var(--fg);
		cursor: pointer;
		text-align: left;
	}
	button.link:disabled {
		cursor: default;
	}
</style>
