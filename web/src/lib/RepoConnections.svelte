<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api, message, type Schemas } from './api';
	import type { Project } from './projects';
	import Button from './ui/Button.svelte';
	import TextField from './ui/TextField.svelte';
	import { toast } from './ui/toast.svelte';

	type Repo = Schemas['RepoConnection'];
	let repos = $state<Repo[]>([]);
	let canStore = $state(true);
	let projects = $state<Project[]>([]);
	let form = $state({ name: '', token: '', project: '', comments: false });

	async function load() {
		const [r, p] = await Promise.all([
			api<{ items: Repo[]; can_store_tokens: boolean }>('/v1/repos'),
			api<{ items: Project[] }>('/v1/projects')
		]);
		repos = r.items;
		canStore = r.can_store_tokens;
		projects = p.items;
	}
	load().catch((err) => toast(message(err), 'error'));

	const projectName = (id: string | null) => projects.find((p) => p.id === id)?.title;

	async function connect(e: SubmitEvent) {
		e.preventDefault();
		const [owner, repo] = form.name
			.replace(/^https?:\/\/github\.com\//, '')
			.replace(/\.git$/, '')
			.split('/');
		try {
			await api('/v1/repos', {
				method: 'POST',
				body: {
					owner,
					repo,
					...(form.token ? { token: form.token } : {}),
					...(form.project ? { project_id: form.project } : {}),
					allow_comments: form.comments
				}
			});
			form = { name: '', token: '', project: '', comments: false };
			toast(t('Repository connected.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function update(r: Repo, body: Record<string, unknown>) {
		try {
			await api(`/v1/repos/${r.id}`, { method: 'PATCH', body });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove(r: Repo) {
		try {
			await api(`/v1/repos/${r.id}`, { method: 'DELETE' });
			toast(t('Disconnected.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<section class="card" aria-labelledby="repos-title">
	<h2 id="repos-title">{t('Repositories')}</h2>
	<p class="muted">
		{t(
			'The Side-Projects and Code Review agents can read these. They never push, approve, or merge. Drafted issues and comments wait for your approval, and comments are posted only where you allow them.'
		)}
	</p>
	{#if repos.length}
		<ul class="rows">
			{#each repos as r (r.id)}
				<li>
					<div class="grow">
						<strong>{r.owner}/{r.repo}</strong>
						<span class="muted small"
							>{t(r.has_token ? 'token stored' : 'public, no token') +
								' · ' +
								t('issues to {project}', {
									project: projectName(r.project_id) ?? t('no project yet')
								})}</span
						>
					</div>
					<label class="check">
						<input
							type="checkbox"
							checked={r.allow_comments}
							onchange={(e) => update(r, { allow_comments: e.currentTarget.checked })}
						/>
						{t('Allow approved comments')}
					</label>
					<label class="visually-hidden" for="repo-project-{r.id}"
						>{t('Project for issues from {repo}', { repo: r.repo })}</label
					>
					<select
						id="repo-project-{r.id}"
						value={r.project_id ?? ''}
						onchange={(e) => update(r, { project_id: e.currentTarget.value || null })}
					>
						<option value="">{t('No project')}</option>
						{#each projects as p (p.id)}<option value={p.id}>{p.title}</option>{/each}
					</select>
					<Button
						variant="ghost"
						onclick={() => remove(r)}
						aria-label={t('Disconnect {owner}/{repo}', { owner: r.owner, repo: r.repo })}
						>{t('Disconnect')}</Button
					>
				</li>
			{/each}
		</ul>
	{/if}
	<form onsubmit={connect}>
		<TextField
			id="repo-name"
			label={t('Repository')}
			placeholder={t('owner/name')}
			bind:value={form.name}
			required
		/>
		<TextField
			id="repo-token"
			label={t('Token (optional)')}
			type="password"
			autocomplete="off"
			hint={canStore
				? t(
						'A fine-grained GitHub token with read access to Contents and Pull requests. Add Pull requests write only if you want approved comments posted. Stored encrypted.'
					)
				: t(
						'This server cannot store tokens until AXON_ENCRYPTION_KEY is set; public repositories still work.'
					)}
			bind:value={form.token}
			disabled={!canStore}
		/>
		<div class="field">
			<label for="repo-project">{t('Project for drafted issues')}</label>
			<select id="repo-project" bind:value={form.project}>
				<option value="">{t('Choose later')}</option>
				{#each projects as p (p.id)}<option value={p.id}>{p.title}</option>{/each}
			</select>
		</div>
		<label class="check"
			><input type="checkbox" bind:checked={form.comments} />
			{t('Allow approved comments on pull requests')}</label
		>
		<Button type="submit" variant="secondary">{t('Connect repository')}</Button>
	</form>
</section>

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	@media (max-width: 30rem) {
		.card {
			padding: var(--space-4);
		}
	}
	h2 {
		font-size: var(--text-lg);
	}
	.rows {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-4);
	}
	.rows li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		flex-wrap: wrap;
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--border);
	}
	.grow {
		flex: 1;
		min-width: 12rem;
		display: grid;
	}
	.small {
		font-size: var(--text-sm);
	}
	.check {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin-bottom: var(--space-3);
		font-size: var(--text-sm);
	}
	.check input {
		accent-color: var(--accent);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
		max-width: 28rem;
	}
	.field label {
		font-weight: 600;
	}
	select {
		font: inherit;
		min-height: 2.5rem;
		padding: var(--space-1) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		max-width: 100%;
	}
</style>
