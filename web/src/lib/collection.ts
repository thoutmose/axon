// The collection (EP-119): how each card is named, and what its next tier takes.
import { api } from './api';
import { t } from './i18n.svelte';

export type Card = {
	id: string;
	family:
		| 'skill'
		| 'habit'
		| 'roadmap'
		| 'project'
		| 'goal'
		| 'cert'
		| 'resource'
		| 'practice'
		| 'focus'
		| 'reviews'
		| 'streak'
		| 'decisions'
		| 'milestones'
		| 'projects'
		| 'goals'
		| 'checkins'
		| 'exams';
	title: string;
	kind: string;
	tier: '' | 'bronze' | 'silver' | 'gold';
	value: number;
	unit: string;
	next: { tier: 'bronze' | 'silver' | 'gold'; target: number } | null;
};

const tierNames: Record<string, string> = { bronze: 'bronze', silver: 'silver', gold: 'gold' };

/** Cards named after one of the person's items (EP-170). */
const own = ['skill', 'habit', 'roadmap', 'project', 'goal', 'cert', 'resource'];

/** A skill's, habit's, roadmap's, project's, or goal's own name; the others by what they count. */
export function cardName(c: Card): string {
	if (own.includes(c.family)) return c.title;
	const names: Record<string, string> = {
		practice: { easy: t('Easy problems'), medium: t('Medium problems'), hard: t('Hard problems') }[
			c.kind as 'easy' | 'medium' | 'hard'
		],
		focus: t('Deep focus'),
		reviews: t('Flashcard reviews'),
		streak: t('Daily streak'),
		decisions: t('Forecaster'),
		milestones: t('Milestones reached'),
		projects: t('Projects shipped'),
		goals: t('Goals achieved'),
		checkins: t('Habit check-ins'),
		exams: t('Certifications passed')
	};
	return names[c.family];
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
		case 'roadmap':
			return t('{target} % of its skills learned for {tier}', { target: c.next.target, tier });
		case 'project':
			return t('{target} % of its cards done for {tier}', { target: c.next.target, tier });
		case 'goal':
			return t('{target} % of its milestones reached for {tier}', { target: c.next.target, tier });
		case 'cert':
			return [
				t('Book the exam for {tier}', { tier }),
				t('Pass it for {tier}', { tier }),
				t('Link its credential for {tier}', { tier })
			][c.next.target - 1];
		case 'resource':
			return t('{n} more notes taken from it for {tier}', { n, tier });
		case 'milestones':
			return t('{n} more milestones reached for {tier}', { n, tier });
		case 'projects':
			return t('{n} more projects done for {tier}', { n, tier });
		case 'goals':
			return t('{n} more goals achieved for {tier}', { n, tier });
		case 'checkins':
			return t('{n} more habit check-ins for {tier}', { n, tier });
		case 'exams':
			return t('{n} more certifications passed for {tier}', { n, tier });
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

/** Skills, roadmaps, goals, projects, habits, and the rest, in that order; empty groups left out. */
export function groups(cards: Card[]) {
	return [
		{
			id: 'skills',
			title: t('Roadmap trophies'),
			cards: cards.filter((c) => c.family === 'skill')
		},
		{
			id: 'roadmaps',
			title: t('Roadmaps'),
			cards: cards.filter((c) => c.family === 'roadmap')
		},
		{ id: 'goals', title: t('Goals'), cards: cards.filter((c) => c.family === 'goal') },
		{ id: 'projects', title: t('Projects'), cards: cards.filter((c) => c.family === 'project') },
		{ id: 'habits', title: t('Habit cards'), cards: cards.filter((c) => c.family === 'habit') },
		{ id: 'certs', title: t('Certifications'), cards: cards.filter((c) => c.family === 'cert') },
		{
			id: 'resources',
			title: t('Roadmap resources'),
			cards: cards.filter((c) => c.family === 'resource')
		},
		{
			id: 'milestones',
			title: t('Milestones'),
			cards: cards.filter((c) => !own.includes(c.family))
		}
	].filter((g) => g.cards.length);
}

/** What an earned card stands for, in a sentence (to share it, EP-121). */
export function achievement(c: Card): string {
	const n = Math.floor(c.value);
	switch (c.family) {
		case 'skill':
			return t('A skill of my Top-5% engineering roadmap.');
		case 'habit':
			return t('Kept {n} scheduled days in a row.', { n });
		case 'roadmap':
			return t('{n} % of a roadmap of my own learned.', { n });
		case 'project':
			return t('{n} % of a project of my roadmap done.', { n });
		case 'goal':
			return t('{n} % of the way to a goal of mine.', { n });
		case 'cert':
			return [
				t('A certification exam booked.'),
				t('A certification passed.'),
				t('A certification passed, its credential linked.')
			][Math.max(0, n - 1)];
		case 'resource':
			return t('{n} notes taken from a resource of my roadmap.', { n });
		case 'milestones':
			return t('{n} milestones reached.', { n });
		case 'projects':
			return t('{n} projects shipped.', { n });
		case 'goals':
			return t('{n} goals achieved.', { n });
		case 'checkins':
			return t('{n} habit check-ins.', { n });
		case 'exams':
			return t('{n} certifications passed.', { n });
		case 'practice':
			return t('{n} problems solved unaided.', { n });
		case 'focus':
			return t('{n} hours of deep focus.', { n });
		case 'reviews':
			return t('{n} flashcard reviews.', { n });
		case 'streak':
			return t('{n} active days in a row.', { n });
		default:
			return t('{n} decisions reviewed against what happened.', { n });
	}
}

/** Turns progress rewards on, keeping every other setting (the AI part has its own endpoints). */
export async function turnOnRewards(): Promise<void> {
	const { ai: _ai, ...doc } = await api<Record<string, unknown>>('/v1/me/settings');
	void _ai;
	await api('/v1/me/settings', {
		method: 'PUT',
		body: { ...doc, gamification: { enabled: true } }
	});
}
