// Ready-made starting points for goals, milestones, habits, and projects.
// Picking one only fills in or creates a draft you then edit; nothing is
// hidden in the template.

import { addDays, daysBetween } from './dates';

export type GoalTemplate = {
	title: string;
	outcome: string;
	/** Days from today to the target date. */
	days: number;
	measure?: { unit: string; start: number; target: number; current: number };
	/** Milestones, each at a share (0–1) of the way to the target date. */
	milestones: [string, number][];
};

export const goalTemplates: GoalTemplate[] = [
	{
		title: 'Run a 10 km race',
		outcome: 'Finish a 10 km race without walking.',
		days: 90,
		measure: { unit: 'km', start: 0, target: 10, current: 0 },
		milestones: [
			['Run 3 km without stopping', 0.25],
			['Run 5 km', 0.5],
			['Run 8 km', 0.8],
			['Race day', 1]
		]
	},
	{
		title: 'Read 12 books this year',
		outcome: 'One book a month, with a literature note for each.',
		days: 365,
		measure: { unit: 'books', start: 0, target: 12, current: 0 },
		milestones: [
			['3 books', 0.25],
			['6 books', 0.5],
			['9 books', 0.75],
			['12 books', 1]
		]
	},
	{
		title: 'Learn a new language to B1',
		outcome: 'Hold a 20-minute conversation and read a news article without a dictionary.',
		days: 365,
		milestones: [
			['A1: introduce myself and order food', 0.2],
			['A2: describe my day and past events', 0.5],
			['B1 practice exam passed', 0.9],
			['B1 exam', 1]
		]
	},
	{
		title: 'Build an emergency fund',
		outcome: 'Three months of expenses saved in a separate account.',
		days: 365,
		measure: { unit: '€', start: 0, target: 6000, current: 0 },
		milestones: [
			['One month of expenses', 0.33],
			['Two months of expenses', 0.66],
			['Three months of expenses', 1]
		]
	},
	{
		title: 'Get a professional certification',
		outcome: 'Pass the exam on the first attempt.',
		days: 120,
		milestones: [
			['Study plan and exam booked', 0.1],
			['All modules studied once', 0.6],
			['Two practice exams above 80 %', 0.9],
			['Exam passed', 1]
		]
	},
	{
		title: 'Ship a side project',
		outcome: 'A first version real people use, with feedback from five of them.',
		days: 90,
		milestones: [
			['Problem and scope written down', 0.1],
			['First working version', 0.5],
			['Launched publicly', 0.8],
			['Feedback from five users', 1]
		]
	}
];

/** ISO date `share` of the way from `from` to `to` (both YYYY-MM-DD). */
export function between(from: string, to: string, share: number): string {
	return addDays(from, Math.round(daysBetween(from, to) * share));
}

/** Milestone sets for an existing goal, spread up to its target date. */
export const milestonePlans: { title: string; steps: [string, number][] }[] = [
	{
		title: 'Quarterly checkpoints',
		steps: [
			['25 % checkpoint', 0.25],
			['Halfway', 0.5],
			['75 % checkpoint', 0.75],
			['Done', 1]
		]
	},
	{
		title: 'Plan, build, review',
		steps: [
			['Plan written', 0.15],
			['First result', 0.5],
			['Review and adjust', 0.75],
			['Finished', 1]
		]
	},
	{
		title: 'Learning path',
		steps: [
			['Fundamentals studied', 0.3],
			['First project with it', 0.6],
			['Teach it to someone or write it up', 0.85],
			['Assessed', 1]
		]
	}
];

export type HabitTemplate = {
	title: string;
	kind: 'build' | 'break';
	identity: string;
	stack_after?: string;
	intention_time?: string;
	intention_place?: string;
	cue?: string;
	craving?: string;
	two_minute: string;
	reward?: string;
	days?: number[];
};

export const habitTemplates: HabitTemplate[] = [
	{
		title: 'Read 20 pages',
		kind: 'build',
		identity: 'I am a reader',
		stack_after: 'After I pour my morning coffee',
		intention_time: '07:15',
		intention_place: 'in my reading chair',
		cue: 'Leave the book on the chair the night before',
		two_minute: 'Read one page',
		reward: 'Coffee while reading'
	},
	{
		title: 'Exercise for 30 minutes',
		kind: 'build',
		identity: 'I am an active person',
		stack_after: 'After I get home from work',
		intention_time: '18:30',
		intention_place: 'at the park',
		cue: 'Put my shoes by the door',
		two_minute: 'Put on my training clothes',
		reward: 'A favourite podcast only while training',
		days: [1, 3, 5]
	},
	{
		title: 'Meditate',
		kind: 'build',
		identity: 'I am a calm person',
		stack_after: 'After I brush my teeth in the morning',
		intention_place: 'on the cushion in the bedroom',
		two_minute: 'One mindful breath',
		reward: 'Tick it off in Axon'
	},
	{
		title: 'Write a journal entry',
		kind: 'build',
		identity: 'I am someone who reflects',
		stack_after: 'After I get into bed',
		intention_time: '22:30',
		cue: 'Notebook and pen on the pillow',
		two_minute: 'Write one sentence about today'
	},
	{
		title: 'Review my notes',
		kind: 'build',
		identity: 'I am a lifelong learner',
		stack_after: 'After lunch',
		two_minute: 'Open one note from last week',
		days: [1, 2, 3, 4, 5]
	},
	{
		title: 'Practise a language',
		kind: 'build',
		identity: 'I speak more than one language',
		stack_after: 'After I sit down on the train',
		two_minute: 'Learn one word'
	},
	{
		title: 'Scroll my phone in bed',
		kind: 'break',
		identity: 'I am a person who sleeps well',
		cue: 'Charge the phone outside the bedroom',
		craving: 'I sleep worse and wake up tired',
		two_minute: 'Log out of social apps at night',
		reward: 'Put a coin in a jar each time I slip, for a charity'
	},
	{
		title: 'Snack on sweets',
		kind: 'break',
		identity: 'I am someone who eats well',
		cue: 'Keep sweets out of the house',
		craving: 'The sugar crash in the afternoon',
		two_minute: 'Keep fruit on the desk instead'
	}
];

export type CardTemplate = { title: string; kind: 'epic' | 'issue'; children?: string[] };
export type ProjectTemplate = {
	title: string;
	description: string;
	wip_limit: number | null;
	cards: CardTemplate[];
};

export const projectTemplates: ProjectTemplate[] = [
	{
		title: 'Software project',
		description:
			'Outcome: a first release in users’ hands.\nDefinition of done: tested, documented, deployed.',
		wip_limit: 3,
		cards: [
			{
				title: 'Discovery',
				kind: 'epic',
				children: ['Write the problem statement', 'List the must-have features']
			},
			{
				title: 'First release',
				kind: 'epic',
				children: ['Set up the repository and CI', 'Build the core feature', 'Deploy to production']
			},
			{
				title: 'Feedback',
				kind: 'epic',
				children: ['Ask five users for feedback', 'Plan the next release']
			}
		]
	},
	{
		title: 'Learning project',
		description: 'Outcome: I can use the skill on a real problem, and I wrote down what I learned.',
		wip_limit: 2,
		cards: [
			{ title: 'Choose a roadmap and resources', kind: 'issue' },
			{ title: 'Study the fundamentals', kind: 'issue' },
			{ title: 'Build a small project with it', kind: 'issue' },
			{ title: 'Write a permanent note on what I learned', kind: 'issue' }
		]
	},
	{
		title: 'Event',
		description: 'Outcome: the event happens on the date, on budget, and people enjoy it.',
		wip_limit: null,
		cards: [
			{ title: 'Date, budget, and guest list', kind: 'issue' },
			{ title: 'Venue booked', kind: 'issue' },
			{ title: 'Invitations sent', kind: 'issue' },
			{ title: 'Food and drinks', kind: 'issue' },
			{ title: 'Thank-you messages', kind: 'issue' }
		]
	},
	{
		title: 'Home improvement',
		description: 'Outcome: the room is finished, within budget.',
		wip_limit: 2,
		cards: [
			{ title: 'Measure and plan', kind: 'issue' },
			{ title: 'Get three quotes', kind: 'issue' },
			{ title: 'Buy materials', kind: 'issue' },
			{ title: 'Do the work', kind: 'issue' },
			{ title: 'Clean up and check', kind: 'issue' }
		]
	},
	{
		title: 'Writing project',
		description: 'Outcome: a published piece, reviewed by one person before it goes out.',
		wip_limit: 1,
		cards: [
			{ title: 'Outline', kind: 'issue' },
			{ title: 'First draft', kind: 'issue' },
			{ title: 'Review by a friend', kind: 'issue' },
			{ title: 'Edit and publish', kind: 'issue' }
		]
	}
];
