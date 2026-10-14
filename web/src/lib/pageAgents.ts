/**
 * Which agents fit which page, and what to ask them there (EP-53). The
 * first agent is the page's own; suggestions follow each skill.
 */
export type PageAgents = { agents: string[]; suggestions: Record<string, string[]> };

const teacher = [
	'Write illustrated flashcards from my newest literature and permanent notes, with the why, an example, and a vivid memory image.',
	'Illustrate my cards that keep slipping, with a drawing and a vivid memory image.'
];

export const pageAgents: Record<string, PageAgents> = {
	today: {
		agents: ['task-manager', 'goals', 'habits'],
		suggestions: {
			'task-manager': ['Plan my day from my next actions and what is due.'],
			goals: ['Which goal has not moved this week, and what is one next step?'],
			habits: ['Which habit should I focus on today, and what is its two-minute version?']
		}
	},
	tasks: {
		agents: ['task-manager', 'goals'],
		suggestions: {
			'task-manager': [
				'Clarify my inbox into next actions with contexts.',
				'Which tasks are stuck, and what is the next physical step for each?',
				'Plan my week from my next actions and due dates.'
			],
			goals: ['Link my tasks to the goals they move forward.']
		}
	},
	goals: {
		agents: ['goals', 'task-manager'],
		suggestions: {
			goals: [
				'Suggest milestones for my active goals.',
				'Link my tasks to the goals they move forward.'
			],
			'task-manager': ['Turn my next milestone into concrete next actions.']
		}
	},
	habits: {
		agents: ['habits'],
		suggestions: {
			habits: [
				'Help me restart a habit I dropped.',
				'Design a two-minute version and a cue for a habit I want to build.',
				'Stack a new habit after one I already do every day.'
			]
		}
	},
	notes: {
		agents: ['teacher', 'resource'],
		suggestions: {
			teacher,
			resource: ['Find good free resources on the topics of my permanent notes.']
		}
	},
	learn: {
		agents: ['teacher', 'tutor'],
		suggestions: {
			teacher,
			tutor: ['Draft a roadmap for what I am learning now, from my assessment.']
		}
	},
	roadmaps: {
		agents: ['tutor', 'resource', 'teacher'],
		suggestions: {
			tutor: ['Draft a roadmap for a skill I want to learn next, from my assessment.'],
			resource: ['Suggest free resources for the skills I am learning now.'],
			teacher
		}
	},
	resources: {
		agents: ['resource'],
		suggestions: { resource: ['Suggest free resources for the skills I am learning now.'] }
	},
	media: {
		agents: ['media'],
		suggestions: {
			media: [
				'Suggest three films I would like, from what I rated highly.',
				'Suggest a book that goes with my current notes.'
			]
		}
	},
	watch: {
		agents: ['tech-watch'],
		suggestions: {
			'tech-watch': [
				'Brief me on this week’s news, grouped by theme, and say what changed.',
				'What is rising in my news that none of my topics covers yet?',
				'Suggest a few primary sources for my topics that I do not follow yet.'
			]
		}
	},
	health: {
		agents: ['personal-trainer', 'nutritionist'],
		suggestions: {
			'personal-trainer': ['Plan my training for next week from my recent workouts.'],
			nutritionist: ['Look at my food log this week and suggest one easy improvement.']
		}
	},
	projects: {
		agents: ['side-projects', 'code-review', 'task-manager'],
		suggestions: {
			'side-projects': ['What should I work on next in my connected repositories?'],
			'code-review': ['Review the open pull requests of my connected repositories.'],
			'task-manager': ['Turn this project’s next steps into next actions.']
		}
	}
};
