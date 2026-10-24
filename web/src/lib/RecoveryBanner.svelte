<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { lowOnRecoveryCodes, recovery } from '$lib/recovery.svelte';
</script>

{#if lowOnRecoveryCodes(recovery.remaining)}
	<p class="banner" role="status">
		{t('Recovery codes left: {n}. Issue new ones before you run out.', { n: recovery.remaining })}
		<a href="/account/security#recovery">{t('Issue new codes')}</a>
	</p>
{/if}

<style>
	.banner {
		margin: 0 0 var(--space-4);
		padding: var(--space-2) var(--space-4);
		border: 1px solid var(--bad);
		border-radius: var(--radius);
		background: var(--accent-soft);
		font-size: var(--text-sm);
	}
</style>
