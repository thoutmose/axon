<script lang="ts">
	// A five-point scale (EP-91): native radio buttons, so arrow keys and
	// screen readers work, drawn as a row of faces. Choosing the chosen one
	// again clears it.
	let {
		legend,
		labels,
		faces,
		value = $bindable(null),
		onchange
	}: {
		legend: string;
		labels: string[];
		faces: string[];
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
			<span aria-hidden="true">{faces[i]}</span>
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
		font-size: 1.5rem;
		border-radius: 999px;
		border: 1px solid var(--border);
		background: var(--surface);
		filter: grayscale(0.8);
		transition: filter var(--motion-fast) var(--ease);
	}
	input:checked + span {
		filter: none;
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
