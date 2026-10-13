<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { onDestroy } from 'svelte';
	import { ApiError, api, message, type Schemas } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Household sharing (EP-61): lists shared with the people who joined with
	// a code; nothing personal is shared.
	type Household = Schemas['Household'];
	type Item = Household['lists'][number]['items'][number];

	let h = $state<Household | null>(null);
	let none = $state(false);
	let error = $state('');
	let me = $state('');
	let name = $state('');
	let code = $state('');
	let invite = $state<{ code: string; expires_at: string } | null>(null);
	let listTitle = $state('');
	let listKind = $state<'shopping' | 'chores' | 'other'>('shopping');
	let drafts = $state<Record<string, string>>({});
	let confirmLeave = $state(false);
	let timer: ReturnType<typeof setInterval> | undefined;

	async function load() {
		error = '';
		try {
			h = await api<Household>('/v1/household');
			none = false;
		} catch (err) {
			if (err instanceof ApiError && err.status === 404) {
				h = null;
				none = true;
			} else error = message(err);
		}
	}
	load();
	api<{ id: string }>('/v1/me')
		.then((m) => (me = m.id))
		.catch(() => {});
	// Others change the lists too: look again every 15 seconds while the page is open.
	timer = setInterval(() => {
		if (document.visibilityState === 'visible' && h) load();
	}, 15000);
	onDestroy(() => clearInterval(timer));

	async function call(method: string, path: string, body?: unknown, said?: string) {
		try {
			const res = await api<Household | undefined>(path, { method, body });
			if (res && 'members' in res) h = res;
			else await load();
			none = !h;
			if (said) toast(said);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const memberName = (id: string | null) => h?.members.find((m) => m.user_id === id)?.name ?? '';
	const amOwner = $derived(!!h?.members.find((m) => m.user_id === me)?.owner);

	async function newInvite() {
		try {
			invite = await api('/v1/household/invites', { method: 'POST' });
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function addItem(listId: string, e: SubmitEvent) {
		e.preventDefault();
		const title = (drafts[listId] ?? '').trim();
		if (!title) return;
		drafts[listId] = '';
		call('POST', `/v1/household/lists/${listId}/items`, { title });
	}

	function tick(it: Item) {
		call('PATCH', `/v1/household/items/${it.id}`, { done: !it.done_at });
	}
</script>

<svelte:head><title>{t('Household · Axon')}</title></svelte:head>

<h1>{t('Household')}</h1>
<p class="muted">
	{t(
		'Lists you share with the people you live with: shopping, chores, anything. Only what you put on a household list is shared; your tasks, notes, money, and everything else stay yours.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if none}
	<div class="start">
		<section class="card" aria-labelledby="new-title">
			<h2 id="new-title">{t('Start a household')}</h2>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					call(
						'POST',
						'/v1/household',
						{ name },
						'Household started. Invite the others with a code.'
					);
				}}
			>
				<TextField
					id="household-name"
					label={t('Name')}
					placeholder={t('Home')}
					bind:value={name}
					maxlength={100}
					required
				/>
				<Button type="submit">{t('Start')}</Button>
			</form>
		</section>
		<section class="card" aria-labelledby="join-title">
			<h2 id="join-title">{t('Join one')}</h2>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					call('POST', '/v1/household/join', { code }, 'Welcome home.');
				}}
			>
				<TextField
					id="household-code"
					label={t('Invite code')}
					placeholder={t('ABCD-EFGH-JKLM-NPQR')}
					bind:value={code}
					maxlength={40}
					required
					autocomplete="off"
				/>
				<Button type="submit" variant="secondary">{t('Join')}</Button>
			</form>
			<p class="muted small">{t('Members will see your display name.')}</p>
		</section>
	</div>
{:else if !h}
	<Status kind="loading" message={t('Loading the household…')} />
{:else}
	<section class="card" aria-labelledby="members-title">
		<h2 id="members-title">{h.name}</h2>
		<ul class="members">
			{#each h.members as m (m.user_id)}
				<li>
					{m.name}{#if m.owner}<span class="muted small">
							{t('· owner')}</span
						>{/if}{#if m.user_id === me}<span class="muted small"> {t('· you')}</span>{/if}
					{#if amOwner && m.user_id !== me}
						<Button
							variant="ghost"
							onclick={() =>
								call(
									'DELETE',
									`/v1/household/members/${m.user_id}`,
									undefined,
									`${m.name} is no longer in the household.`
								)}
							aria-label={t('Remove {name}', { name: m.name })}>{t('Remove')}</Button
						>
					{/if}
				</li>
			{/each}
		</ul>
		{#if invite}
			<div class="invite" role="status">
				<p>
					{t(
						'Give this code to who you want to invite. It works once, until {value}, and is not shown again.',
						{ value: new Date(invite.expires_at).toLocaleDateString(appLocale()) }
					)}
				</p>
				<code>{invite.code}</code>
			</div>
		{/if}
		<div class="row">
			<Button variant="secondary" onclick={newInvite}>{t('Invite someone')}</Button>
			<Button variant="ghost" onclick={() => (confirmLeave = true)}
				>{t('Leave the household')}</Button
			>
		</div>
	</section>

	<div class="lists">
		{#each h.lists as l (l.id)}
			<section class="card" aria-labelledby="list-{l.id}">
				<div class="head">
					<h2 id="list-{l.id}">{l.title}</h2>
					<span class="muted small">{t('{open} to do', { open: l.open })}</span>
				</div>
				<ul class="items">
					{#each l.items as it (it.id)}
						<li class:done={!!it.done_at}>
							<label>
								<input type="checkbox" checked={!!it.done_at} onchange={() => tick(it)} />
								<span>{it.title}</span>
							</label>
							{#if it.done_at}
								<span class="muted small">{memberName(it.done_by) || 'done'}</span>
							{:else}
								<select
									aria-label={t('Who does “{title}”', { title: it.title })}
									value={it.assignee ?? ''}
									onchange={(e) =>
										call('PATCH', `/v1/household/items/${it.id}`, {
											assignee: e.currentTarget.value
										})}
								>
									<option value="">{t('Anyone')}</option>
									{#each h.members as m (m.user_id)}<option value={m.user_id}>{m.name}</option
										>{/each}
								</select>
							{/if}
							<Button
								variant="ghost"
								onclick={() => call('DELETE', `/v1/household/items/${it.id}`)}
								aria-label={t('Delete {title}', { title: it.title })}>×</Button
							>
						</li>
					{/each}
				</ul>
				<form class="add" onsubmit={(e) => addItem(l.id, e)}>
					<label for="add-{l.id}" class="visually-hidden"
						>{t('Add to {title}', { title: l.title })}</label
					>
					<input
						id="add-{l.id}"
						placeholder={t('Add an item')}
						bind:value={drafts[l.id]}
						maxlength="300"
						autocomplete="off"
					/>
					<Button type="submit" variant="secondary">{t('Add')}</Button>
				</form>
				<Button
					variant="ghost"
					onclick={() =>
						call('DELETE', `/v1/household/lists/${l.id}`, undefined, `“${l.title}” deleted.`)}
					aria-label={t('Delete the list {title}', { title: l.title })}>{t('Delete list')}</Button
				>
			</section>
		{/each}
		<section class="card" aria-labelledby="newlist-title">
			<h2 id="newlist-title">{t('New list')}</h2>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					call('POST', '/v1/household/lists', { title: listTitle, kind: listKind });
					listTitle = '';
				}}
			>
				<TextField
					id="list-title"
					label={t('List')}
					placeholder={t('Groceries')}
					bind:value={listTitle}
					maxlength={100}
					required
				/>
				<label for="list-kind">{t('For')}</label>
				<select id="list-kind" bind:value={listKind}>
					<option value="shopping">{t('Shopping')}</option>
					<option value="chores">{t('Chores')}</option>
					<option value="other">{t('Anything')}</option>
				</select>
				<Button type="submit" variant="secondary">{t('Create')}</Button>
			</form>
		</section>
	</div>
{/if}

<Dialog bind:open={confirmLeave} title={t('Leave the household?')}>
	<p>{t('You will no longer see its lists. What you were given to do goes back to anyone.')}</p>
	<div class="row">
		<Button
			variant="danger"
			onclick={() => {
				confirmLeave = false;
				call('DELETE', '/v1/household/me', undefined, 'You left the household.');
			}}>{t('Leave')}</Button
		>
		<Button variant="secondary" onclick={() => (confirmLeave = false)}>{t('Stay')}</Button>
	</div>
</Dialog>

<style>
	.start,
	.lists {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 20rem), 1fr));
		gap: var(--space-3);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-3);
	}
	.card h2 {
		margin-top: 0;
		font-size: var(--text-lg);
	}
	.head {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
	}
	.members,
	.items {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-1);
	}
	.items li {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}
	.items label {
		flex: 1;
		display: flex;
		gap: var(--space-2);
		align-items: center;
	}
	.items .done span {
		text-decoration: line-through;
		color: var(--muted);
	}
	.add {
		display: flex;
		gap: var(--space-2);
		margin: var(--space-2) 0;
	}
	.add input {
		flex: 1;
		font: inherit;
	}
	.invite code {
		font-size: var(--text-lg);
		letter-spacing: 0.08em;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
