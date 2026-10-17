<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { math } from '$lib/notes';
	import Button from '$lib/ui/Button.svelte';

	// Writing maths (EP-150): the common structures and symbols one click
	// away, the formula drawn as you type, and $…$ or $$…$$ inserted, as in
	// Obsidian, so the vault shows it too.
	let {
		open = $bindable(false),
		oninsert
	}: { open: boolean; oninsert: (markdown: string) => void } = $props();

	let dialog: HTMLDialogElement;
	let field = $state<HTMLTextAreaElement>();
	let tex = $state('');
	let block = $state(false);
	const example = 'e^{i\\pi} + 1 = 0'; // LaTeX, the same in every language

	$effect(() => {
		if (open && !dialog.open) {
			tex = '';
			dialog.showModal();
			field?.focus();
		}
		if (!open && dialog.open) dialog.close();
	});

	// Each snippet's ◻ is where the caret goes.
	const groups: { name: string; items: [string, string][] }[] = $derived([
		{
			name: t('Structures'),
			items: [
				['a/b', '\\frac{◻}{}'],
				['xⁿ', '^{◻}'],
				['xᵢ', '_{◻}'],
				['√', '\\sqrt{◻}'],
				['ⁿ√', '\\sqrt[n]{◻}'],
				['∑', '\\sum_{i=1}^{n} ◻'],
				['∏', '\\prod_{i=1}^{n} ◻'],
				['∫', '\\int_{a}^{b} ◻ \\, dx'],
				['lim', '\\lim_{n \\to \\infty} ◻'],
				['( )', '\\left( ◻ \\right)'],
				['|x|', '\\lvert ◻ \\rvert'],
				['x̄', '\\bar{◻}'],
				['x⃗', '\\vec{◻}'],
				['ẋ', '\\dot{◻}'],
				['(ⁿₖ)', '\\binom{◻}{k}'],
				[t('Matrix'), '\\begin{pmatrix} ◻ & b \\\\ c & d \\end{pmatrix}'],
				[t('Cases'), 'f(x) = \\begin{cases} ◻ & x \\ge 0 \\\\ -x & x < 0 \\end{cases}'],
				[t('Aligned'), '\\begin{aligned} a &= ◻ \\\\ &= c \\end{aligned}']
			]
		},
		{
			name: t('Symbols'),
			items: [
				['≤', '\\le '],
				['≥', '\\ge '],
				['≠', '\\ne '],
				['≈', '\\approx '],
				['≡', '\\equiv '],
				['∝', '\\propto '],
				['±', '\\pm '],
				['×', '\\times '],
				['·', '\\cdot '],
				['∞', '\\infty '],
				['→', '\\to '],
				['⇒', '\\Rightarrow '],
				['⇔', '\\iff '],
				['∀', '\\forall '],
				['∃', '\\exists '],
				['∈', '\\in '],
				['∉', '\\notin '],
				['⊂', '\\subset '],
				['∪', '\\cup '],
				['∩', '\\cap '],
				['∅', '\\emptyset '],
				['∂', '\\partial '],
				['∇', '\\nabla '],
				['ℝ', '\\mathbb{R}'],
				['ℕ', '\\mathbb{N}'],
				['ℤ', '\\mathbb{Z}'],
				['ℙ', '\\mathbb{P}'],
				['𝔼', '\\mathbb{E}'],
				['O(n)', 'O(◻)']
			]
		},
		{
			name: t('Greek letters'),
			items: (
				[
					['α', 'alpha'],
					['β', 'beta'],
					['γ', 'gamma'],
					['δ', 'delta'],
					['ε', 'epsilon'],
					['θ', 'theta'],
					['λ', 'lambda'],
					['μ', 'mu'],
					['π', 'pi'],
					['ρ', 'rho'],
					['σ', 'sigma'],
					['τ', 'tau'],
					['φ', 'phi'],
					['χ', 'chi'],
					['ω', 'omega'],
					['Δ', 'Delta'],
					['Σ', 'Sigma'],
					['Ω', 'Omega']
				] as [string, string][]
			).map(([g, n]) => [g, `\\${n} `])
		}
	]);

	async function add(snippet: string) {
		const el = field!;
		const at = el.selectionStart;
		const caret = snippet.indexOf('◻');
		const text = snippet.replace('◻', '');
		tex = tex.slice(0, at) + text + tex.slice(el.selectionEnd);
		await Promise.resolve();
		el.focus();
		const pos = at + (caret >= 0 ? caret : text.length);
		el.setSelectionRange(pos, pos);
	}

	const preview = $derived(tex.trim() ? math(tex, block) : '');
	function insert() {
		if (!tex.trim()) return;
		oninsert(block ? `\n$$\n${tex.trim()}\n$$\n` : `$${tex.trim()}$`);
		open = false;
	}
</script>

<dialog
	bind:this={dialog}
	class="maths"
	aria-labelledby="maths-title"
	onclose={() => (open = false)}
>
	<h2 id="maths-title">{t('Write maths')}</h2>
	{#each groups as g (g.name)}
		<div class="group" role="group" aria-label={g.name}>
			<span class="muted small">{g.name}</span>
			{#each g.items as [label, snippet] (label)}
				<button type="button" title={snippet.replace('◻', '…')} onclick={() => add(snippet)}
					>{label}</button
				>
			{/each}
		</div>
	{/each}
	<label class="formula">
		<span>{t('Formula (LaTeX)')}</span>
		<textarea bind:this={field} bind:value={tex} rows="3" spellcheck="false" placeholder={example}
		></textarea>
	</label>
	<div class="preview" aria-live="polite" aria-label={t('Preview')}>
		<!-- KaTeX MathML from $lib/notes: escaped, untrusted commands off. -->
		{#if preview}{@html preview}{:else}<span class="muted small"
				>{t('The formula is drawn here as you type.')}</span
			>{/if}
	</div>
	<div class="row">
		<label class="small"
			><input type="checkbox" bind:checked={block} /> {t('On its own line')}</label
		>
		<span class="grow"></span>
		<Button variant="ghost" onclick={() => (open = false)}>{t('Cancel')}</Button>
		<Button disabled={!tex.trim()} onclick={insert}>{t('Insert')}</Button>
	</div>
	<p class="muted small">
		{t('Notes take LaTeX between $…$ in a line, or $$…$$ on lines of their own, as Obsidian does.')}
	</p>
</dialog>

<style>
	.maths {
		width: min(44rem, calc(100vw - 2 * var(--space-4)));
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--surface);
		color: var(--fg);
		padding: var(--space-4);
		box-shadow: var(--shadow);
	}
	.maths::backdrop {
		background: rgb(0 0 0 / 0.45);
	}
	h2 {
		margin: 0 0 var(--space-3);
		font-size: var(--text-lg);
	}
	.group {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		align-items: center;
		margin-bottom: var(--space-2);
	}
	.group > span {
		width: 100%;
	}
	.group button {
		min-width: 2.25rem;
		height: 2.25rem;
		padding: 0 var(--space-2);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--bg);
		color: var(--fg);
		cursor: pointer;
	}
	.group button:hover {
		border-color: var(--accent);
	}
	.formula {
		display: grid;
		gap: var(--space-1);
		margin-top: var(--space-2);
	}
	.formula textarea {
		font-family: var(--font-mono, monospace);
	}
	.preview {
		min-height: 3.5rem;
		display: grid;
		place-items: center;
		margin: var(--space-2) 0;
		padding: var(--space-2);
		border: 1px dashed var(--border);
		border-radius: var(--radius);
		font-size: 1.2em;
		overflow-x: auto;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		align-items: center;
	}
	.grow {
		flex: 1;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
