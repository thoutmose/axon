// Health protocols after Andrew Huberman (EP-118): habits he recommends on
// the Huberman Lab podcast, each with what to do, why, how strong the
// evidence is, the studies behind it, and a caution where one is due.
// Educational, not medical advice. "Debated" means the advice is his and no
// study has tested it directly.

import type { HabitTemplate } from './entityTemplates';

export type Evidence = 'strong' | 'moderate' | 'debated';

export type Protocol = {
	id: string;
	title: string;
	how: string;
	why: string;
	evidence: Evidence;
	sources: { label: string; url: string }[];
	caution?: string;
	habit: HabitTemplate;
};

const huberman = {
	label: 'Huberman Lab, Toolkit for Sleep',
	url: 'https://www.hubermanlab.com/newsletter/toolkit-for-sleep'
};

export const protocols: Protocol[] = [
	{
		id: 'morning-light',
		title: 'Morning sunlight',
		how: 'Within an hour of waking, spend 5 to 10 minutes outside in daylight (10 to 20 when it is cloudy), without sunglasses. Never look straight at the sun.',
		why: 'Bright light in the morning sets the body clock earlier, so you get sleepy earlier at night and wake more easily.',
		evidence: 'strong',
		sources: [
			{
				label: 'Khalsa et al. 2003, J Physiol',
				url: 'https://doi.org/10.1113/jphysiol.2003.040477'
			},
			huberman
		],
		habit: {
			title: 'Morning sunlight',
			kind: 'build',
			identity: 'I am someone who sleeps well',
			stack_after: 'After I get up',
			intention_place: 'outside, on the balcony or a short walk',
			two_minute: 'Step outside with my first drink'
		}
	},
	{
		id: 'regular-wake',
		title: 'Wake at the same time every day',
		how: 'Get up at the same time on weekdays and weekends, within about an hour.',
		why: 'A regular schedule keeps the body clock steady; irregular sleep goes with later, poorer sleep.',
		evidence: 'strong',
		sources: [
			{ label: 'Phillips et al. 2017, Sci Rep', url: 'https://doi.org/10.1038/s41598-017-03171-4' },
			huberman
		],
		habit: {
			title: 'Wake at the same time',
			kind: 'build',
			identity: 'I am someone who sleeps well',
			intention_time: '07:00',
			cue: 'One alarm, across the room',
			two_minute: 'Sit up when the alarm rings'
		}
	},
	{
		id: 'delay-caffeine',
		title: 'Wait 90 minutes before the first coffee',
		how: 'Have your first caffeine 90 to 120 minutes after waking.',
		why: 'Huberman suggests it to avoid an afternoon slump. It is his advice: studies have not tested it directly, and many people feel no difference.',
		evidence: 'debated',
		sources: [huberman],
		habit: {
			title: 'Wait 90 minutes before coffee',
			kind: 'build',
			identity: 'I am in charge of my energy',
			stack_after: 'After my morning sunlight',
			two_minute: 'Drink a glass of water first'
		}
	},
	{
		id: 'caffeine-cutoff',
		title: 'Last coffee 9 hours before bed',
		how: 'Stop caffeine about 9 hours before bedtime; the Coffee and tea card tells you until when.',
		why: 'A coffee drunk 8.8 hours before bed or later shortens and lightens sleep, even when you fall asleep easily.',
		evidence: 'strong',
		sources: [
			{
				label: 'Gardiner et al. 2023, Sleep Med Rev',
				url: 'https://doi.org/10.1016/j.smrv.2023.101764'
			},
			huberman
		],
		habit: {
			title: 'No coffee after the cut-off',
			kind: 'break',
			identity: 'I am someone who sleeps well',
			cue: 'Switch to herbal tea after lunch',
			craving: 'A shorter, lighter night',
			two_minute: 'Choose decaf after 14:00'
		}
	},
	{
		id: 'dim-evening',
		title: 'Dim the lights in the evening',
		how: 'In the 2 to 3 hours before bed, use low, warm lamps rather than bright overhead light, and dim screens.',
		why: 'Ordinary room light before bed delays and shortens the release of melatonin, the hormone of the night.',
		evidence: 'strong',
		sources: [
			{
				label: 'Gooley et al. 2011, J Clin Endocrinol Metab',
				url: 'https://doi.org/10.1210/jc.2010-2098'
			},
			huberman
		],
		habit: {
			title: 'Dim the lights after 21:00',
			kind: 'build',
			identity: 'I am someone who sleeps well',
			intention_time: '21:00',
			cue: 'A lamp on a timer',
			two_minute: 'Switch off the ceiling light'
		}
	},
	{
		id: 'cyclic-sighing',
		title: 'Five minutes of cyclic sighing',
		how: 'Breathe in through the nose, take a second short breath in on top, then breathe out slowly through the mouth. Repeat for 5 minutes.',
		why: 'In a one-month trial, daily cyclic sighing lifted mood and lowered breathing rate more than mindfulness meditation.',
		evidence: 'moderate',
		sources: [
			{
				label: 'Balban et al. 2023, Cell Rep Med',
				url: 'https://doi.org/10.1016/j.xcrm.2022.100895'
			}
		],
		habit: {
			title: 'Cyclic sighing',
			kind: 'build',
			identity: 'I am a calm person',
			stack_after: 'After lunch',
			two_minute: 'Three slow sighs'
		}
	},
	{
		id: 'nsdr',
		title: 'Non-sleep deep rest',
		how: 'Lie down for 10 to 20 minutes with a guided yoga nidra or body-scan recording.',
		why: 'Deep relaxation can restore some energy after a short night; a small study found more dopamine during yoga nidra.',
		evidence: 'moderate',
		sources: [
			{
				label: 'Kjaer et al. 2002, Cogn Brain Res',
				url: 'https://doi.org/10.1016/S0926-6410(01)00106-9'
			},
			huberman
		],
		habit: {
			title: 'Non-sleep deep rest',
			kind: 'build',
			identity: 'I am someone who recovers well',
			stack_after: 'After lunch',
			two_minute: 'Lie down and breathe for two minutes'
		}
	},
	{
		id: 'zone-2',
		title: 'Zone 2 cardio and strength',
		how: 'About 150 to 180 minutes a week of easy cardio you can talk through, and strength training two or three times a week.',
		why: 'It is what the WHO recommends: at least 150 minutes of moderate activity and two days of muscle strengthening a week.',
		evidence: 'strong',
		sources: [
			{
				label: 'WHO guidelines on physical activity, 2020',
				url: 'https://www.who.int/publications/i/item/9789240015128'
			}
		],
		caution: 'With a heart condition or after a long break, start slowly and ask a doctor.',
		habit: {
			title: 'Zone 2 cardio',
			kind: 'build',
			identity: 'I am an active person',
			intention_time: '18:30',
			two_minute: 'Put on my running shoes',
			days: [1, 3, 5]
		}
	},
	{
		id: 'cold',
		title: 'Deliberate cold',
		how: 'About 11 minutes a week in total of cold water, in two to four short sessions: a cold shower or a cold bath you can stay in safely.',
		why: 'Winter swimmers who did about this much had more active brown fat and burned more energy in the cold.',
		evidence: 'moderate',
		sources: [
			{
				label: 'Søberg et al. 2021, Cell Rep Med',
				url: 'https://doi.org/10.1016/j.xcrm.2021.100408'
			},
			huberman
		],
		caution:
			'Not with a heart condition or high blood pressure without a doctor’s advice; never alone in open water.',
		habit: {
			title: 'Cold shower',
			kind: 'build',
			identity: 'I am someone who does hard things',
			stack_after: 'After my morning shower',
			two_minute: 'Thirty seconds of cold at the end',
			days: [1, 3, 5]
		}
	}
];

/** How strong the evidence is, in words. */
export const evidenceNames: Record<Evidence, string> = {
	strong: 'Strong evidence',
	moderate: 'Some evidence',
	debated: 'His advice, not tested'
};
