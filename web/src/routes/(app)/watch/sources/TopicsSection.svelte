<script lang="ts">
	import { api, message } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Section from '$lib/ui/Section.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import type { WatchTopic } from '$lib/watch';

	// The topics that label the news: the list and the form to add or edit one.
	let { topics, onchange }: { topics: WatchTopic[]; onchange: () => Promise<void> } = $props();

	let topicName = $state('');
	let topicWords = $state('');
	let editing = $state<string | null>(null);
	async function saveTopic(e: SubmitEvent) {
		e.preventDefault();
		const body = {
			name: topicName,
			keywords: topicWords
				.split(',')
				.map((w) => w.trim())
				.filter(Boolean)
		};
		try {
			await api(editing ? `/v1/watch/topics/${editing}` : '/v1/watch/topics', {
				method: editing ? 'PUT' : 'POST',
				body
			});
			topicName = topicWords = '';
			editing = null;
			toast(t('Topic saved; the last month’s news are labelled again.'));
			await onchange();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	function editTopic(tp: WatchTopic) {
		editing = tp.id;
		topicName = tp.name;
		topicWords = tp.keywords.join(', ');
	}
	async function deleteTopic(tp: WatchTopic) {
		if (!confirm(t('Remove the topic {name}?', { name: tp.name }))) return;
		try {
			await api(`/v1/watch/topics/${tp.id}`, { method: 'DELETE' });
			await onchange();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<Section id="topics" title={t('Topics')}>
	<p class="muted small">
		{t(
			'Topics label the news: a keyword in the title, or two in the text, and the item gets the topic. Synonyms from the thesaurus count too.'
		)}
	</p>
	<ul class="topics">
		{#each topics as tp (tp.id)}
			<li>
				<strong>{tp.name}</strong>
				<span class="muted small">{tp.keywords.join(', ')}</span>
				<Button variant="ghost" onclick={() => editTopic(tp)}>{t('Edit')}</Button>
				<Button variant="ghost" onclick={() => deleteTopic(tp)}>{t('Remove')}</Button>
			</li>
		{/each}
	</ul>
	<form class="row" onsubmit={saveTopic}>
		<input
			aria-label={t('Topic')}
			placeholder={t('Topic, e.g. Databases')}
			bind:value={topicName}
			required
		/>
		<input
			aria-label={t('Keywords')}
			placeholder={t('Keywords, separated by commas')}
			bind:value={topicWords}
		/>
		<Button type="submit" variant="secondary">{editing ? t('Save') : t('Add the topic')}</Button>
	</form>
</Section>

<style>
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
	}
	.row input {
		flex: 1 1 14rem;
		min-width: 0;
	}
	.topics {
		list-style: none;
		padding: 0;
	}
	.topics li {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: baseline;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
