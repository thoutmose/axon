<script lang="ts">
	import { scaleArt, type ScaleKind } from './journal';

	// A five-point scale (EP-91): native radio buttons, so arrow keys and
	// screen readers work, drawn as a row of line icons (EP-129). Choosing the
	// chosen one again clears it.
	let {
		legend,
		labels,
		kind,
		value = $bindable(null),
		onchange
	}: {
		legend: string;
		labels: string[];
		kind: ScaleKind;
		value?: number | null;
		onchange?: () => void;
	} = $props();
	const name = `scale-${Math.random().toString(36).slice(2)}`;
</script>

<fieldset class="scale">
	<legend>{legend}</legend>
	{#each labels as label, i (i)}
		<label title={label}>
			<input
				type="radio"
				{name}
				value={i + 1}
				checked={value === i + 1}
				onclick={() => {
					value = value === i + 1 ? null : i + 1;
					onchange?.();
				}}
			/>
			<span aria-hidden="true"
				><svg
					viewBox="0 0 24 24"
					width="24"
					height="24"
					fill="none"
					stroke="currentColor"
					stroke-width="1.6"
					stroke-linecap="round"
					stroke-linejoin="round"
					><!-- Constant drawing from scaleArt, never user input. -->{@html scaleArt(
						kind,
						i + 1
					)}</svg
				></span
			>
			<span class="visually-hidden">{label}</span>
		</label>
	{/each}
	<span class="small muted" aria-live="polite">{value ? labels[value - 1] : ''}</span>
</fieldset>

<style>
	.scale {
		border: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
	}
	legend {
		font-weight: 600;
		margin-bottom: var(--space-1);
		width: 100%;
	}
	label {
		position: relative;
		cursor: pointer;
	}
	input {
		position: absolute;
		opacity: 0;
		inset: 0;
		cursor: pointer;
	}
	label span[aria-hidden] {
		display: grid;
		place-items: center;
		width: 2.75rem;
		height: 2.75rem;
		border-radius: 999px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--muted);
		transition:
			color var(--motion-fast) var(--ease),
			border-color var(--motion-fast) var(--ease);
	}
	label:hover span[aria-hidden] {
		color: var(--fg);
	}
	input:checked + span {
		color: var(--accent);
		border-color: var(--accent);
		background: var(--accent-soft);
	}
	input:focus-visible + span {
		box-shadow: var(--focus);
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
