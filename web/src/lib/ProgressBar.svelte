<script lang="ts">
	import { percentLabel, type Progress } from './goals';

	let { progress, label }: { progress: Progress; label: string } = $props();
</script>

<div class="progress">
	{#if progress.percent !== null}
		<div
			class="track"
			role="progressbar"
			aria-label={label}
			aria-valuemin="0"
			aria-valuemax="100"
			aria-valuenow={progress.percent}
			aria-valuetext="{progress.percent}%: {progress.summary}"
		>
			<div class="fill" style:width="{progress.percent}%"></div>
		</div>
	{/if}
	<p class="muted">
		<strong>{percentLabel(progress)}</strong>{progress.percent !== null
			? ` · ${progress.summary}`
			: ''}
	</p>
</div>

<style>
	.track {
		height: 0.5rem;
		border-radius: 999px;
		background: var(--accent-soft);
		overflow: hidden;
	}
	.fill {
		height: 100%;
		background: var(--accent);
		border-radius: 999px;
		transition: width var(--motion) var(--ease);
	}
	p {
		margin: var(--space-1) 0 0;
		font-size: var(--text-sm);
	}
	strong {
		color: var(--fg);
	}
</style>
