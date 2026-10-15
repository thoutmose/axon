// The collection (EP-119): how each card is named, and what its next tier takes.
import { t } from './i18n.svelte';

export type Card = {
	id: string;
	family: 'skill' | 'habit' | 'practice' | 'focus' | 'reviews' | 'streak' | 'decisions';
	title: string;
	kind: string;
	tier: '' | 'bronze' | 'silver' | 'gold';
	value: number;
	unit: string;
	next: { tier: 'bronze' | 'silver' | 'gold'; target: number } | null;
};

const tierNames: Record<string, string> = { bronze: 'bronze', silver: 'silver', gold: 'gold' };

/** A skill's or habit's own name; the others by what they count. */
export function cardName(c: Card): string {
	if (c.family === 'skill' || c.family === 'habit') return c.title;
	return {
		practice: { easy: t('Easy problems'), medium: t('Medium problems'), hard: t('Hard problems') }[
			c.kind as 'easy' | 'medium' | 'hard'
		],
		focus: t('Deep focus'),
		reviews: t('Flashcard reviews'),
		streak: t('Daily streak'),
		decisions: t('Forecaster')
	}[c.family] as string;
}

/** What the next tier takes, or Complete. */
export function nextStep(c: Card): string {
	if (!c.next) return t('Complete');
	const tier = t(tierNames[c.next.tier]);
	const n = Math.max(1, Math.ceil(c.next.target - c.value));
	if (c.family === 'skill') {
		if (c.next.tier === 'bronze') return t('Start learning it for {tier}', { tier });
		if (c.next.tier === 'gold') return t('Mark it learned for {tier}', { tier });
		return t('{n} more hours of focus on it for {tier}', { n, tier });
	}
	switch (c.family) {
		case 'habit':
			return t('{n} more days in a row for {tier}', { n, tier });
		case 'practice':
			return t('{n} more solved for {tier}', { n, tier });
		case 'focus':
			return t('{n} more hours of focus for {tier}', { n, tier });
		case 'reviews':
			return t('{n} more reviews for {tier}', { n, tier });
		case 'streak':
			return t('A run of {target} active days for {tier}', { target: c.next.target, tier });
		default:
			return t('{n} more decisions reviewed for {tier}', { n, tier });
	}
}

/** Skills, habits, and the rest, in that order; empty groups left out. */
export function groups(cards: Card[]) {
	return [
		{
			id: 'skills',
			title: t('Roadmap trophies'),
			cards: cards.filter((c) => c.family === 'skill')
		},
		{ id: 'habits', title: t('Habit cards'), cards: cards.filter((c) => c.family === 'habit') },
		{
			id: 'milestones',
			title: t('Milestones'),
			cards: cards.filter((c) => c.family !== 'skill' && c.family !== 'habit')
		}
	].filter((g) => g.cards.length);
}
