<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { actions, goTo, vimKeys, type Shortcut } from './shortcuts';
	import Dialog from './ui/Dialog.svelte';

	let { open = $bindable(false) }: { open: boolean } = $props();
	const groups: [string, Shortcut[]][] = $derived([
		[t('Go to'), goTo],
		[t('Actions'), actions],
		[t('Vim mode in the note editor'), vimKeys]
	]);
</script>

<Dialog bind:open title={t('Keyboard shortcuts')}>
	<!-- svelte-ignore a11y_no_noninteractive_tabindex (a scrollable region must be reachable by keyboard) -->
	<div class="groups" role="region" aria-label={t('Keyboard shortcuts')} tabindex="0">
		{#each groups as [title, items] (title)}
			<section aria-label={t(title)}>
				<h3>{t(title)}</h3>
				<dl>
					{#each items as s (s.keys)}
						<dt><kbd>{s.keys}</kbd></dt>
						<dd>{t(s.label)}</dd>
					{/each}
				</dl>
			</section>
		{/each}
	</div>
	<p class="muted small">{t('Turn single-key shortcuts off, or Vim mode on, in Preferences.')}</p>
</Dialog>

<style>
	.groups {
		display: grid;
		gap: var(--space-4);
		max-height: 65vh;
		overflow: auto;
	}
	h3 {
		font-size: var(--text-base);
		margin: 0 0 var(--space-2);
	}
	dl {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: var(--space-1) var(--space-3);
		margin: 0;
	}
	dd {
		margin: 0;
	}
	kbd {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		border: 1px solid var(--border);
		border-bottom-width: 2px;
		border-radius: var(--radius-sm);
		padding: 0 var(--space-1);
		white-space: nowrap;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
