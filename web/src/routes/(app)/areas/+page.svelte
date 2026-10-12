<script lang="ts">
	import ParaList from '$lib/ParaList.svelte';
	import { areas } from '$lib/para';
</script>

<svelte:head><title>{areas.title} · Axon</title></svelte:head>

<ParaList kind={areas} />
