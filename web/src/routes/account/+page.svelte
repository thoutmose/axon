<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { startRegistration } from '@simplewebauthn/browser';
	import { api, apiResponse, ApiError, message, type AuthStatus, type Schemas } from '$lib/api';

	type Me = Schemas['Me'];
	type Session = Schemas['Session'];
	type Passkey = Schemas['Passkey'];
	type Consent = Schemas['Consent'];
	type Event = Schemas['AuditEvent'];
	type Token = Schemas['Token'];

	let me = $state<Me | null>(null);
	let status = $state<AuthStatus | null>(null);
	let sessions = $state<Session[]>([]);
	let passkeys = $state<Passkey[]>([]);
	let consents = $state<Consent[]>([]);
	let events = $state<Event[]>([]);
	let tokens = $state<Token[]>([]);
	let etag = '';
	let newName = $state('');
	let tokenName = $state('');
	let tokenScope = $state<'read' | 'write'>('read');
	let tokenDays = $state(90);
	let newToken = $state('');
	let error = $state('');
	let notice = $state('');
	let busy = $state(false);

	let newPasskey = $state('');
	let codes = $state<string[]>([]);
	let totpSetup = $state<{ secret: string; otpauth_url: string } | null>(null);
	let totpCode = $state('');
	let confirmDelete = $state('');

	const when = (iso: string | null) => (iso ? new Date(iso).toLocaleString() : 'never');

	async function load() {
		try {
			const [current, auth] = await Promise.all([
				apiResponse<Me>('/v1/me'),
				api<AuthStatus>('/v1/auth/status')
			]);
			me = current.data;
			etag = current.etag;
			status = auth;
			newName ||= me.display_name;
			const [s, p, c, a, t] = await Promise.all([
				api<{ items: Session[] }>('/v1/me/sessions'),
				api<{ items: Passkey[] }>('/v1/me/passkeys'),
				api<{ items: Consent[] }>('/v1/me/consents'),
				api<{ items: Event[] }>('/v1/me/audit'),
				api<{ items: Token[] }>('/v1/me/tokens')
			]);
			sessions = s.items;
			passkeys = p.items;
			consents = c.items;
			events = a.items;
			tokens = t.items;
		} catch (err) {
			if (err instanceof ApiError && err.status === 401) return goto('/signin');
			error = message(err);
		}
	}

	async function run(action: () => Promise<unknown>, done = '') {
		busy = true;
		error = '';
		notice = '';
		try {
			await action();
			notice = done;
			await load();
		} catch (err) {
			error = message(err);
		} finally {
			busy = false;
		}
	}

	const rename = (e: SubmitEvent) => {
		e.preventDefault();
		return run(
			() =>
				api('/v1/me', {
					method: 'PATCH',
					body: { display_name: newName },
					headers: { 'If-Match': etag }
				}),
			'Name updated.'
		);
	};

	const createToken = (e: SubmitEvent) => {
		e.preventDefault();
		return run(async () => {
			const res = await api<{ token: string }>('/v1/me/tokens', {
				method: 'POST',
				body: { name: tokenName, scope: tokenScope, expires_in_days: tokenDays }
			});
			newToken = res.token;
			tokenName = '';
		}, 'Token created. Copy it now; it will not be shown again.');
	};

	const revokeToken = (t: Token) =>
		run(() => api(`/v1/me/tokens/${t.id}`, { method: 'DELETE' }), `Revoked “${t.name}”.`);

	const addPasskey = (e: SubmitEvent) => {
		e.preventDefault();
		return run(async () => {
			const options = await api<{ publicKey: never }>('/v1/me/passkeys/begin', {
				method: 'POST',
				body: { name: newPasskey }
			});
			const response = await startRegistration({ optionsJSON: options.publicKey });
			await api('/v1/me/passkeys/finish', { method: 'POST', body: response });
			newPasskey = '';
		}, 'Passkey added.');
	};

	const removePasskey = (p: Passkey) =>
		run(() => api(`/v1/me/passkeys/${p.id}`, { method: 'DELETE' }), `Removed “${p.name}”.`);

	const revoke = (s: Session) =>
		run(async () => {
			await api(`/v1/me/sessions/${s.id}`, { method: 'DELETE' });
			if (s.current) await goto('/signin');
		}, 'Session signed out.');

	const signOutEverywhere = () =>
		run(async () => {
			await api('/v1/me/logout-all', { method: 'POST' });
			await goto('/signin');
		});

	const signOut = () =>
		run(async () => {
			await api('/v1/auth/logout', { method: 'POST' });
			await goto('/');
		});

	const newCodes = () =>
		run(async () => {
			codes = (await api<{ recovery_codes: string[] }>('/v1/me/recovery-codes', { method: 'POST' }))
				.recovery_codes;
		}, 'New recovery codes issued; the old ones no longer work.');

	const totpBegin = () =>
		run(async () => {
			totpSetup = await api('/v1/me/totp/begin', { method: 'POST' });
		});

	const totpSubmit = (e: SubmitEvent, on: boolean) => {
		e.preventDefault();
		return run(
			async () => {
				await api(on ? '/v1/me/totp/confirm' : '/v1/me/totp/disable', {
					method: 'POST',
					body: { code: totpCode }
				});
				totpSetup = null;
				totpCode = '';
			},
			on ? 'Authenticator codes are on.' : 'Authenticator codes are off.'
		);
	};

	const setConsent = (c: Consent, granted: boolean) =>
		run(() => api(`/v1/me/consents/${c.id}`, { method: 'PUT', body: { granted } }));

	const deleteAccount = (e: SubmitEvent) => {
		e.preventDefault();
		return run(async () => {
			await api('/v1/me', { method: 'DELETE', body: { confirm: confirmDelete } });
			await goto('/');
		});
	};

	onMount(load);
</script>

<svelte:head>
	<title>Account · Axon</title>
</svelte:head>

{#if me}
	<h1>{me.display_name}</h1>
	<p><button class="secondary" onclick={signOut} disabled={busy}>Sign out</button></p>

	<div aria-live="polite">
		{#if notice}<p class="ok">{notice}</p>{/if}
		{#if error}<p class="error" role="alert">{error}</p>{/if}
	</div>

	<section aria-labelledby="profile">
		<h2 id="profile">Name</h2>
		<form onsubmit={rename}>
			<label for="display-name">Shown in Axon and your authenticator</label>
			<input id="display-name" bind:value={newName} maxlength="100" required />
			<p><button class="secondary" disabled={busy || newName === me.display_name}>Save</button></p>
		</form>
	</section>

	<section aria-labelledby="passkeys">
		<h2 id="passkeys">Passkeys</h2>
		<ul class="list">
			{#each passkeys as p (p.id)}
				<li>
					<span>{p.name} <span class="muted">· last used {when(p.last_used_at)}</span></span>
					<button
						class="danger"
						onclick={() => removePasskey(p)}
						disabled={busy || passkeys.length < 2}
						aria-label={`Remove passkey ${p.name}`}>Remove</button
					>
				</li>
			{/each}
		</ul>
		<form onsubmit={addPasskey}>
			<label for="pk-name">Add a passkey on another device</label>
			<input
				id="pk-name"
				bind:value={newPasskey}
				placeholder="e.g. Phone"
				maxlength="100"
				required
			/>
			<p><button disabled={busy}>Add passkey</button></p>
		</form>
	</section>

	<section aria-labelledby="recovery">
		<h2 id="recovery">Recovery codes</h2>
		<p>{me.recovery_codes_remaining} of 10 unused.</p>
		{#if codes.length}
			<p>Save these now; they will not be shown again.</p>
			<ul class="codes">
				{#each codes as c (c)}<li>{c}</li>{/each}
			</ul>
		{/if}
		<button class="secondary" onclick={newCodes} disabled={busy}>Issue new codes</button>
	</section>

	{#if status?.totp_available}
		<section aria-labelledby="totp">
			<h2 id="totp">Authenticator app</h2>
			{#if me.totp_enabled}
				<p>On. Recovery codes also need a code from your authenticator app.</p>
				<form onsubmit={(e) => totpSubmit(e, false)}>
					<label for="totp-off">Current code, to turn it off</label>
					<input id="totp-off" bind:value={totpCode} inputmode="numeric" required />
					<p><button class="danger" disabled={busy}>Turn off</button></p>
				</form>
			{:else if totpSetup}
				<p>Add this key to your authenticator app, then enter the code it shows.</p>
				<p><code>{totpSetup.secret}</code></p>
				<p class="muted"><a href={totpSetup.otpauth_url}>Open in authenticator app</a></p>
				<form onsubmit={(e) => totpSubmit(e, true)}>
					<label for="totp-on">Code from the app</label>
					<input id="totp-on" bind:value={totpCode} inputmode="numeric" required />
					<p><button disabled={busy}>Turn on</button></p>
				</form>
			{:else}
				<p class="muted">Off. When on, signing in with a recovery code also needs an app code.</p>
				<button class="secondary" onclick={totpBegin} disabled={busy}>Set up</button>
			{/if}
		</section>
	{/if}

	<section aria-labelledby="sessions">
		<h2 id="sessions">Signed-in devices</h2>
		<ul class="list">
			{#each sessions as s (s.id)}
				<li>
					<span
						>{s.user_agent || 'Unknown device'}
						<span class="muted"
							>· {s.current ? 'this device' : `active ${when(s.last_seen_at)}`}</span
						></span
					>
					<button class="secondary" onclick={() => revoke(s)} disabled={busy}>Sign out</button>
				</li>
			{/each}
		</ul>
		<button class="danger" onclick={signOutEverywhere} disabled={busy}>Sign out everywhere</button>
	</section>

	<section aria-labelledby="tokens">
		<h2 id="tokens">Access tokens</h2>
		<p class="muted">
			For scripts and sync tools. Tokens can read (or change) your data but never manage your
			account.
		</p>
		<ul class="list">
			{#each tokens as t (t.id)}
				<li>
					<span>{t.name} <span class="muted">· {t.scope} · expires {when(t.expires_at)}</span></span
					>
					<button class="secondary" onclick={() => revokeToken(t)} disabled={busy}>Revoke</button>
				</li>
			{/each}
		</ul>
		{#if newToken}<p><code>{newToken}</code></p>{/if}
		<form onsubmit={createToken}>
			<label for="token-name">Name</label>
			<input id="token-name" bind:value={tokenName} maxlength="100" required />
			<label for="token-scope">Access</label>
			<select id="token-scope" bind:value={tokenScope}>
				<option value="read">Read only</option>
				<option value="write">Read and write</option>
			</select>
			<label for="token-days">Expires after (days)</label>
			<input id="token-days" type="number" min="1" max="365" bind:value={tokenDays} required />
			<p><button disabled={busy}>Create token</button></p>
		</form>
	</section>

	<section aria-labelledby="privacy">
		<h2 id="privacy">Privacy choices</h2>
		<p class="muted">
			Everything is off until you turn it on. You can change your mind at any time.
		</p>
		{#each consents as c (c.id)}
			<label>
				<input
					type="checkbox"
					style="width:auto"
					checked={c.granted}
					disabled={busy}
					onchange={(e) => setConsent(c, e.currentTarget.checked)}
				/>
				{c.description}
			</label>
		{/each}
	</section>

	<section aria-labelledby="data">
		<h2 id="data">Your data</h2>
		<p>
			<a href="/api/v1/me/export" download>Download everything</a> (JSON and a Markdown summary).
		</p>
		<form onsubmit={deleteAccount}>
			<label for="confirm">Delete your account and all its data. Type DELETE to confirm.</label>
			<input id="confirm" bind:value={confirmDelete} autocomplete="off" required />
			<p>
				<button class="danger" disabled={busy || confirmDelete !== 'DELETE'}>Delete account</button>
			</p>
		</form>
	</section>

	<section aria-labelledby="activity">
		<h2 id="activity">Recent activity</h2>
		<ul class="list">
			{#each events.slice(0, 20) as ev (ev.id)}
				<li><span>{ev.action}</span><span class="muted">{when(ev.created_at)} · {ev.ip}</span></li>
			{/each}
		</ul>
	</section>
{:else if error}
	<p class="error" role="alert">{error}</p>
{:else}
	<p class="muted">Loading…</p>
{/if}
