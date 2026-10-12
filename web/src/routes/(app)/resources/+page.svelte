<script lang="ts">
	import ParaList from '$lib/ParaList.svelte';
	import { resources } from '$lib/para';
</script>

<svelte:head><title>{resources.title} · Axon</title></svelte:head>

<ParaList kind={resources} />
