<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api } from './api';

	// Notes close in meaning to this one (EP-62). Shown only when search by
	// meaning is set up and the note has been embedded; silent otherwise.
	let { id }: { id: string } = $props();
	type Hit = { id: string; title: string; path: string; score: number };
	let items = $state<Hit[]>([]);

	$effect(() => {
		void id;
		items = [];
		api<{ items: Hit[] }>(`/v1/notes/${id}/similar`)
			.then((r) => (items = r.items.filter((h) => h.score >= 0.3).slice(0, 6)))
			.catch(() => (items = []));
	});
</script>

{#if items.length}
	<section class="card" aria-labelledby="similar-title">
		<h2 id="similar-title">{t('Close in meaning')}</h2>
		<ul class="rows">
			{#each items as h (h.id)}
				<li>
					<a href="/notes/{h.id}">{h.title}</a>
					<span class="muted small">{h.path} · {Math.round(h.score * 100)}%</span>
				</li>
			{/each}
		</ul>
	</section>
{/if}

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin: var(--space-4) 0;
	}
	.card h2 {
		margin-top: 0;
		font-size: var(--text-lg);
	}
	.rows {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-1);
	}
	.rows li {
		display: flex;
		gap: var(--space-2);
		align-items: baseline;
		flex-wrap: wrap;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
