<script lang="ts">
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
		Level {p.level} · {p.streak?.days ?? 0}-day run{p.today_xp ? ` · +${p.today_xp} today` : ''}
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
