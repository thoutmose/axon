<script lang="ts">
	import { page } from '$app/state';
	import NoteFolder from './NoteFolder.svelte';
	import type { Folder } from './noteTree';

	let { folder, depth = 0 }: { folder: Folder; depth?: number } = $props();
	const current = (id: string) => page.url.pathname === `/notes/${id}`;
</script>

<ul class="tree" class:nested={depth > 0}>
	{#each folder.folders as f (f.path)}
		<li>
			<details open={depth < 1}>
				<summary title={f.path}
					><span class="folder" aria-hidden="true">▸</span><span class="name">{f.name}</span>
					<span class="count">{f.count}</span></summary
				>
				<NoteFolder folder={f} depth={depth + 1} />
			</details>
		</li>
	{/each}
	{#each folder.notes as n (n.id)}
		<li>
			<a href="/notes/{n.id}" aria-current={current(n.id) ? 'page' : undefined} title={n.path}
				>{n.title}</a
			>
		</li>
	{/each}
</ul>

<style>
	.tree {
		list-style: none;
		margin: 0;
		padding: 0;
		min-width: 0;
	}
	li,
	details {
		min-width: 0;
	}
	.name {
		flex: 1;
		min-width: 0;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.nested {
		padding-left: var(--space-3);
		border-left: 1px solid var(--border);
		margin-left: var(--space-2);
	}
	summary {
		cursor: pointer;
		list-style: none;
		padding: 0.15rem var(--space-1);
		border-radius: var(--radius-sm);
		font-size: var(--text-sm);
		font-weight: 600;
		display: flex;
		gap: var(--space-1);
		align-items: center;
	}
	summary::-webkit-details-marker {
		display: none;
	}
	.folder {
		display: inline-block;
		transition: transform var(--motion-fast) var(--ease);
		color: var(--muted);
	}
	details[open] > summary .folder {
		transform: rotate(90deg);
	}
	.count {
		color: var(--muted);
		font-weight: 400;
		margin-left: auto;
	}
	a {
		display: block;
		padding: 0.15rem var(--space-2);
		border-radius: var(--radius-sm);
		color: var(--fg);
		text-decoration: none;
		font-size: var(--text-sm);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	a:hover,
	summary:hover {
		background: var(--accent-soft);
	}
	a[aria-current='page'] {
		background: var(--accent-soft);
		font-weight: 600;
	}
</style>
