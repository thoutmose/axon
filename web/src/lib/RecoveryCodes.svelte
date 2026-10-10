<script lang="ts">
	import Button from './ui/Button.svelte';

	let { codes }: { codes: string[] } = $props();

	function download() {
		const blob = new Blob([`Axon recovery codes\n\n${codes.join('\n')}\n`], { type: 'text/plain' });
		const a = document.createElement('a');
		a.href = URL.createObjectURL(blob);
		a.download = 'axon-recovery-codes.txt';
		a.click();
		URL.revokeObjectURL(a.href);
	}
</script>

<p>Save these now, for example in a password manager. They will not be shown again.</p>
<ul class="codes" aria-label="Recovery codes">
	{#each codes as c (c)}<li>{c}</li>{/each}
</ul>
<p><Button variant="secondary" onclick={download}>Download as a text file</Button></p>

<style>
	.codes {
		font-family: var(--font-mono);
		list-style: none;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
		gap: var(--space-2);
		padding: var(--space-4);
		background: var(--accent-soft);
		border-radius: var(--radius);
	}
</style>
