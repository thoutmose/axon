<script lang="ts">
	import Collection from '$lib/Collection.svelte';
	import { t } from '$lib/i18n.svelte';

	// The collection (EP-119) on its own page, so the menu can name it (EP-127).
</script>

<svelte:head><title>{t('Collection · Axon')}</title></svelte:head>

<h1>{t('Collection')}</h1>
<p class="muted">
	{t(
		'Trophies for your roadmap skills, cards for your habits, and milestones, raised from bronze to silver to gold as your streaks, practice, and focus grow.'
	)}
</p>
<Collection />
