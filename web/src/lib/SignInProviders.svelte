<script lang="ts">
	import { api } from './api';
	import { t } from './i18n.svelte';

	// Sign in with another account (EP-109): one button per provider the
	// operator configured. The browser goes to the provider and comes back.
	let { intent = 'signin', next = '/' }: { intent?: 'signin' | 'link'; next?: string } = $props();
	let providers = $state<{ id: string; name: string }[]>([]);
	api<{ items: { id: string; name: string }[] }>('/v1/auth/providers')
		.then((r) => (providers = r.items))
		.catch(() => (providers = []));

	const icons: Record<string, string> = {
		google: 'G',
		microsoft: '⊞',
		apple: 'A',
		discord: '◉',
		slack: '#',
		oidc: '🔑'
	};
	const href = (id: string) =>
		`/api/v1/auth/oauth/${id}/start?intent=${intent}&next=${encodeURIComponent(next)}`;
</script>

{#if providers.length}
	<div class="providers">
		{#each providers as p (p.id)}
			<a class="provider {p.id}" href={href(p.id)} data-sveltekit-reload>
				<span class="icon" aria-hidden="true">{icons[p.id] ?? '🔑'}</span>
				{intent === 'link'
					? t('Link {name}', { name: p.name })
					: t('Continue with {name}', { name: p.name })}
			</a>
		{/each}
	</div>
{/if}

<style>
	.providers {
		display: grid;
		gap: var(--space-2);
		max-width: 22rem;
		margin: var(--space-3) 0;
	}
	.provider {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-4);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--fg);
		text-decoration: none;
		font-weight: 600;
	}
	.provider:hover {
		border-color: var(--accent);
	}
	.provider:focus-visible {
		outline: 2px solid var(--focus);
		outline-offset: 2px;
	}
	.icon {
		width: 1.5rem;
		text-align: center;
		font-weight: 700;
	}
</style>
