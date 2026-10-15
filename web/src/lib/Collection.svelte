<script lang="ts">
	import { api, message } from '$lib/api';
	import { cardName, groups, nextStep, type Card } from '$lib/collection';
	import { t } from '$lib/i18n.svelte';
	import Status from '$lib/ui/Status.svelte';

	// The collection (EP-119): a trophy for each roadmap skill, a card for
	// each habit, and milestones, raised from bronze to silver to gold.
	type Collection = { cards: Card[]; earned: number; total: number };
	let c = $state<Collection | null>(null);
	let error = $state('');
	api<Collection>('/v1/progress/collection')
		.then((r) => (c = r))
		.catch((err) => (error = message(err)));

	const rank = { gold: 3, silver: 2, bronze: 1, '': 0 };
	const sorted = (cards: Card[]) =>
		[...cards].sort(
			(a, b) => rank[b.tier] - rank[a.tier] || cardName(a).localeCompare(cardName(b))
		);
	const count = (tier: Card['tier']) => c?.cards.filter((k) => k.tier === tier).length ?? 0;
	const tierName = (tier: Card['tier']) =>
		({ gold: t('Gold'), silver: t('Silver'), bronze: t('Bronze'), '': t('Locked') })[tier];
	// One emblem per family, drawn in the card's metal.
	const emblems: Record<Card['family'], string> = {
		skill: 'M12 3 4 7v5c0 5 3.5 8 8 9 4.5-1 8-4 8-9V7z',
		habit: 'M12 21c-4 0-7-3-7-7 0-5 7-11 7-11s7 6 7 11c0 4-3 7-7 7z',
		practice: 'M8 6 2 12l6 6M16 6l6 6-6 6',
		focus: 'M6 3h12M6 21h12M7 3c0 6 10 6 10 9s-10 3-10 9M17 3c0 6-10 6-10 9',
		reviews: 'M4 6h12v14H4zM8 3h12v14',
		streak: 'M12 2c1 4 6 6 6 12a6 6 0 0 1-12 0c0-3 2-5 3-6 0 3 2 4 3 4 0-4-2-6 0-10z',
		decisions: 'M12 3v18M12 7h7l2 2-2 2h-7M12 13H5l-2 2 2 2h7'
	};
</script>

<section class="card glass" aria-labelledby="collection-title">
	<h2 id="collection-title">{t('Your collection')}</h2>
	{#if error}
		<Status kind="error" message={error} />
	{:else if !c}
		<Status kind="loading" message={t('Loading…')} />
	{:else}
		<p class="summary">
			{t('{earned} of {total} tiers collected', { earned: c.earned, total: c.total })}
			<span class="medals"
				><span class="m gold">{t('{n} gold', { n: count('gold') })}</span>
				<span class="m silver">{t('{n} silver', { n: count('silver') })}</span>
				<span class="m bronze">{t('{n} bronze', { n: count('bronze') })}</span></span
			>
		</p>
		<progress max={c.total} value={c.earned} aria-label={t('Tiers collected')}></progress>
		{#each groups(c.cards) as g (g.id)}
			<h3>{g.title}</h3>
			<ul class="cards">
				{#each sorted(g.cards) as k (k.id)}
					<li class="cc {k.tier || 'locked'}">
						<svg viewBox="0 0 24 24" width="40" height="40" aria-hidden="true">
							<path d={emblems[k.family]} />
						</svg>
						<strong class="name">{cardName(k)}</strong>
						<span class="tier">{tierName(k.tier)}</span>
						{#if k.next && k.next.target > 0}
							<progress
								max={k.next.target}
								value={Math.min(k.value, k.next.target)}
								aria-label={t('Toward {tier}', { tier: tierName(k.next.tier) })}
							></progress>
						{/if}
						<span class="how">{nextStep(k)}</span>
					</li>
				{/each}
			</ul>
		{/each}
	{/if}
</section>

<style>
	.summary {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		align-items: baseline;
		margin: 0 0 var(--space-2);
	}
	.medals {
		display: flex;
		gap: var(--space-2);
		font-size: var(--text-sm);
	}
	progress {
		width: 100%;
	}
	h3 {
		margin: var(--space-4) 0 var(--space-2);
		font-size: var(--text-lg);
	}
	.cards {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(10.5rem, 1fr));
		gap: var(--space-3);
	}
	/* A card face in its metal: bronze, silver, gold; locked ones in grey. */
	.cc {
		--metal: #8a8f98;
		--face: var(--surface-solid);
		display: grid;
		gap: var(--space-1);
		justify-items: center;
		text-align: center;
		padding: var(--space-3);
		border-radius: var(--radius-lg);
		border: 2px solid var(--metal);
		background:
			radial-gradient(
				circle at 50% 0%,
				color-mix(in srgb, var(--metal) 28%, transparent),
				transparent 70%
			),
			var(--face);
		color: var(--fg);
	}
	.cc.bronze {
		--metal: #b0763f;
	}
	.cc.silver {
		--metal: #9aa3ad;
	}
	.cc.gold {
		--metal: #d4a72c;
		box-shadow:
			0 0 0 1px color-mix(in srgb, var(--metal) 40%, transparent),
			0 6px 18px -8px var(--metal);
	}
	.cc.locked {
		border-style: dashed;
	}
	.cc svg path {
		fill: color-mix(in srgb, var(--metal) 35%, transparent);
		stroke: var(--metal);
		stroke-width: 1.6;
		stroke-linejoin: round;
		stroke-linecap: round;
	}
	.cc.locked svg path {
		fill: none;
		stroke: var(--muted);
	}
	.name {
		font-size: var(--text-sm);
		line-height: 1.3;
	}
	.tier {
		font-size: 0.75rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}
	.how {
		font-size: 0.75rem;
		color: var(--muted);
	}
	.m {
		padding: 0 var(--space-2);
		border-radius: 999px;
		border: 1px solid var(--border);
	}
</style>
