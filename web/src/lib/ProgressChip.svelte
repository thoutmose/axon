<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api } from './api';

	type P = {
		enabled: boolean;
		level?: number;
		today_xp?: number;
		streak?: { days: number; state: string };
	};
	let p = $state<P | null>(null);
	api<P>('/v1/progress')
		.then((r) => (p = r))
		.catch(() => {});
</script>

{#if p?.enabled}
	<a class="chip" href="/progress">
		{t('Level {level} · {days}-day run', { level: p.level, days: p.streak?.days ?? 0 }) +
			(p.today_xp ? t(' · +{xp} today', { xp: p.today_xp }) : '')}
	</a>
{/if}

<style>
	.chip {
		display: inline-block;
		padding: var(--space-1) var(--space-3);
		border-radius: 999px;
		background: var(--accent-soft);
		border: 1px solid var(--border);
		color: var(--fg);
		text-decoration: none;
		font-size: var(--text-sm);
		font-weight: 600;
	}
</style>
