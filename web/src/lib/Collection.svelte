<script lang="ts">
	import { api, ApiError, message } from '$lib/api';
	import {
		achievement,
		cardName,
		groups,
		nextStep,
		turnOnRewards,
		type Card
	} from '$lib/collection';
	import Button from '$lib/ui/Button.svelte';
	import ShareButton from '$lib/ShareButton.svelte';
	import { t } from '$lib/i18n.svelte';
	import Status from '$lib/ui/Status.svelte';

	// The collection (EP-119): a trophy for each roadmap skill, a card for
	// each habit, and milestones, raised from bronze to silver to gold.
	type Collection = { cards: Card[]; earned: number; total: number };
	let c = $state<Collection | null>(null);
	let error = $state('');
	// With progress rewards off, the collection offers to turn them on (EP-164).
	let off = $state(false);
	function load() {
		api<Collection>('/v1/progress/collection')
			.then((r) => {
				c = r;
				off = false;
			})
			.catch((err) => {
				if (err instanceof ApiError && err.status === 409) off = true;
				else error = message(err);
			});
	}
	load();
	async function turnOn() {
		try {
			await turnOnRewards();
			load();
		} catch (err) {
			error = message(err);
		}
	}

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
		decisions: 'M12 3v18M12 7h7l2 2-2 2h-7M12 13H5l-2 2 2 2h7',
		roadmap: 'M3 20h4l3-8 4 5 3-11h4M3 20V4',
		project: 'M4 7h16v12H4zM9 7V4h6v3',
		goal: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM12 12h.01',
		cert: 'M12 3a5 5 0 1 0 0 10 5 5 0 0 0 0-10zM9 12l-2 9 5-3 5 3-2-9',
		resource: 'M4 5c3-1 6-1 8 1 2-2 5-2 8-1v14c-3-1-6-1-8 1-2-2-5-2-8-1zM12 6v14',
		milestones: 'M5 21V4M5 4h11l-2 4 2 4H5',
		projects: 'M5 13l4 4L19 7',
		goals: 'M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0zM7 6H4a3 3 0 0 0 3 4M17 6h3a3 3 0 0 1-3 4',
		checkins: 'M4 12l5 5L20 6M4 6h7',
		exams: 'M12 3l9 5-9 5-9-5zM7 11v5c3 2 7 2 10 0v-5'
	};
</script>

<section class="card glass" aria-labelledby="collection-title">
	<h2 id="collection-title">{t('Your collection')}</h2>
	{#if error}
		<Status kind="error" message={error} />
	{:else if off}
		<p>
			{t(
				'The collection comes with progress rewards, which are off. Turn them on to see your trophies and cards; you can turn them off again in Preferences.'
			)}
		</p>
		<Button onclick={turnOn}>{t('Turn on progress rewards')}</Button>
	{:else if !c}
		<Status kind="loading" message={t('Loading…')} />
	{:else}
		<p class="summary">
			{t(
				'{cards} cards, each with three tiers (bronze, silver, gold): {earned} of {total} tiers collected',
				{
					cards: c.cards.length,
					earned: c.earned,
					total: c.total
				}
			)}
			<span class="medals"
				><span class="m gold">{t('{n} gold', { n: count('gold') })}</span>
				<span class="m silver">{t('{n} silver', { n: count('silver') })}</span>
				<span class="m bronze">{t('{n} bronze', { n: count('bronze') })}</span></span
			>
		</p>
		<progress max={c.total} value={c.earned} aria-label={t('Tiers collected')}></progress>
		{#each groups(c.cards) as g (g.id)}
			<details open={g.cards.length <= 40}>
				<summary
					><h3>
						{g.title}
						<span class="muted">{g.cards.filter((k) => k.tier).length}/{g.cards.length}</span>
					</h3></summary
				>
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
							{#if k.tier}
								<ShareButton
									spec={{
										title: t('{tier} on Axon', { tier: tierName(k.tier) }),
										value: cardName(k),
										caption: achievement(k),
										kind: k.tier
									}}
									text={t('{tier}: {name}. {what} #Axon', {
										tier: tierName(k.tier),
										name: cardName(k),
										what: achievement(k)
									})}
								/>
							{/if}
						</li>
					{/each}
				</ul>
			</details>
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
		display: inline;
		font-size: var(--text-lg);
	}
	summary {
		margin: var(--space-4) 0 var(--space-2);
		cursor: pointer;
	}
	.muted {
		color: var(--muted);
		font-size: var(--text-sm);
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
