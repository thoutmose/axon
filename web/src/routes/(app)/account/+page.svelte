<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { storedLocale, t } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, apiResponse, message, type Schemas } from '$lib/api';
	import Invitations from '$lib/Invitations.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import { avatar, avatarSrc } from '$lib/avatar.svelte';

	let me = $state<Schemas['Me'] | null>(null);
	let etag = '';
	let name = $state('');
	let timezone = $state('UTC');
	const zones =
		typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : ['UTC'];
	const deviceZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
	let error = $state('');
	let busy = $state(false);

	async function load() {
		error = '';
		try {
			const res = await apiResponse<Schemas['Me']>('/v1/me');
			me = res.data;
			etag = res.etag;
			name = me.display_name;
			timezone = me.timezone;
			avatar.version = me.avatar_updated_at ?? '';
		} catch (err) {
			error = message(err);
		}
	}

	async function save(body: Record<string, string>, done: string) {
		busy = true;
		try {
			await api('/v1/me', { method: 'PATCH', body, headers: { 'If-Match': etag } });
			toast(done);
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	const rename = (e: SubmitEvent) => {
		e.preventDefault();
		return save({ display_name: name }, t('Name updated.'));
	};
	const setZone = (zone: string) =>
		save({ timezone: zone }, t('Time zone set to {zone}.', { zone }));

	async function signOut() {
		busy = true;
		try {
			await api('/v1/auth/logout', { method: 'POST' });
			await goto('/');
		} catch (err) {
			toast(message(err), 'error');
			busy = false;
		}
	}

	async function upload(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		busy = true;
		try {
			const res = await fetch('/api/v1/me/avatar', {
				method: 'PUT',
				headers: { 'Content-Type': file.type, 'Accept-Language': storedLocale() },
				body: file
			});
			if (!res.ok) throw new Error((await res.json()).detail ?? res.statusText);
			toast(t('Picture updated.'));
			await load();
		} catch (err) {
			toast(err instanceof Error ? err.message : String(err), 'error');
		} finally {
			busy = false;
			input.value = '';
		}
	}

	async function removePicture() {
		try {
			await api('/v1/me/avatar', { method: 'DELETE' });
			toast(t('Back to your default picture.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	onMount(load);
</script>

<svelte:head><title>{t('Account · Axon')}</title></svelte:head>

{#if me}
	<Section
		id="picture"
		title={t('Profile picture')}
		description={t(
			'PNG or JPEG, up to 5 MB. It is cropped square and resized; camera and location details are removed.'
		)}
	>
		<div class="picture">
			<img src={avatarSrc(avatar.version)} alt={me.display_name} width="96" height="96" />
			<div class="picture-actions">
				<label class="upload">
					<span>{t('Upload a picture')}</span>
					<input type="file" accept="image/png,image/jpeg" onchange={upload} disabled={busy} />
				</label>
				{#if me.avatar_updated_at}
					<Button variant="ghost" onclick={removePicture}>{t('Use the default picture')}</Button>
				{/if}
			</div>
		</div>
	</Section>
	<Section id="profile" title={t('Name')}>
		<form onsubmit={rename}>
			<TextField
				id="display-name"
				label={t('Your name')}
				hint={t('Shown in Axon and in your passkey manager.')}
				bind:value={name}
				maxlength={100}
				required
			/>
			<Button type="submit" variant="secondary" {busy} disabled={name === me.display_name}
				>{t('Save')}</Button
			>
		</form>
	</Section>
	<Section
		id="zone"
		title={t('Time zone')}
		description={t('Decides what “today” means for your tasks and check-ins.')}
	>
		<div class="zone">
			<label for="timezone" class="visually-hidden">{t('Time zone')}</label>
			<select
				id="timezone"
				bind:value={timezone}
				onchange={() => setZone(timezone)}
				disabled={busy}
			>
				{#each zones as z (z)}<option value={z}>{z}</option>{/each}
			</select>
			{#if deviceZone && deviceZone !== me.timezone}
				<Button variant="ghost" onclick={() => setZone(deviceZone)} {busy}
					>{t("Use this device's zone ({deviceZone})", { deviceZone })}</Button
				>
			{/if}
		</div>
	</Section>
	<Section id="summary" title={t('At a glance')}>
		<ul class="facts">
			<li>
				<strong>{me.passkeys}</strong>
				{t(me.passkeys === 1 ? 'passkey' : 'passkeys')}
			</li>
			<li><strong>{me.recovery_codes_remaining}</strong> {t('of 10 recovery codes unused')}</li>
			<li>{t('Authenticator app')} <strong>{t(me.totp_enabled ? 'on' : 'off')}</strong></li>
			<li>
				{t('Member since {date}', {
					date: new Date(me.created_at).toLocaleDateString(appLocale())
				})}
			</li>
		</ul>
		<Button variant="secondary" onclick={signOut} {busy}>{t('Sign out')}</Button>
	</Section>
	<Invitations />
{:else if error}
	<Status kind="error" message={error} retry={load} />
{:else}
	<Status kind="loading" message={t('Loading your account…')} />
{/if}

<style>
	.zone {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		align-items: center;
	}
	.zone select {
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		max-width: 100%;
	}
	.facts {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-2);
		margin: 0 0 var(--space-6);
	}
	.picture {
		display: flex;
		gap: var(--space-4);
		align-items: center;
		flex-wrap: wrap;
	}
	.picture img {
		border-radius: 50%;
		border: 1px solid var(--border);
	}
	.picture-actions {
		display: grid;
		gap: var(--space-2);
	}
	.upload {
		display: grid;
		gap: var(--space-1);
		font-weight: 600;
	}
</style>
