<script lang="ts">
	import CaffeineCard from '$lib/CaffeineCard.svelte';
	import { t } from '$lib/i18n.svelte';

	// Coffee and tea (EP-117) on its own page, so the menu can name it (EP-127).
</script>

<svelte:head><title>{t('Coffee and tea · Axon')}</title></svelte:head>

<h1 class="visually-hidden">{t('Coffee and tea')}</h1>
<CaffeineCard />
