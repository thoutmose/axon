// An illustration for each type of note (EP-98): a small original line
// drawing, and a sentence on what the type is for and what it nods to.
// Drawn in a 48 × 48 box with the current colour.

export type NoteType = { title: string; about: string; d: string };

const sheet = 'M14 6 H30 L36 12 V42 H14 Z M30 6 V12 H36 M19 20 H31 M19 26 H31 M19 32 H27';

export const noteTypes: Record<string, NoteType> = {
	permanent: {
		title: 'Permanent note',
		about:
			'A keystone: one idea, in your words, that holds others up, as each of the 90,000 cards in Niklas Luhmann’s slip-box did.',
		d: 'M6 42 V24 A18 18 0 0 1 42 24 V42 M14 42 V25 A10 10 0 0 1 34 25 V42 M20 8 L28 8 L26 15 L22 15 Z'
	},
	literature: {
		title: 'Literature note',
		about:
			'An open book with a note in the margin, like Fragonard’s Young Girl Reading (c. 1770): what a source says, in your own words.',
		d: 'M24 12 C18 8 10 8 4 10 V38 C10 36 18 36 24 40 C30 36 38 36 44 38 V10 C38 8 30 8 24 12 Z M24 12 V40 M9 17 H19 M9 22 H19 M29 17 H39 M29 22 H35'
	},
	fleeting: {
		title: 'Fleeting note',
		about:
			'A feather caught before it drifts away, as in the opening of Forrest Gump: a passing thought, to sort out later.',
		d: 'M40 6 C26 8 14 20 10 38 M40 6 C38 22 28 32 12 34 M18 26 L28 24 M22 18 L32 16 M8 42 L12 34'
	},
	daily: {
		title: 'Daily note',
		about:
			'A sunrise over the day: its three priorities, the tests due, the habits, and what you learned.',
		d: 'M4 34 H44 M12 34 A12 12 0 0 1 36 34 M24 10 V16 M10 18 L14 22 M38 18 L34 22 M10 42 H38'
	},
	weekly: {
		title: 'Weekly note',
		about:
			'A wheel of seven days, turned once a week: the GTD weekly review, empty inbox to next actions.',
		d: 'M24 6 A18 18 0 1 1 23.9 6 Z M24 24 L24 6 M24 24 L38 13 M24 24 L42 28 M24 24 L32 40 M24 24 L16 40 M24 24 L6 28 M24 24 L10 13'
	},
	monthly: {
		title: 'Monthly note',
		about:
			'The moon’s phases, a month at a glance: goals, milestones, and the roadmap checked against the plan.',
		d: 'M8 24 A6 6 0 1 1 7.9 24 Z M24 18 A6 6 0 0 1 24 30 Z M24 18 A6 6 0 0 0 24 30 M40 18 A6 6 0 1 1 39.9 18 Z'
	},
	project: {
		title: 'Project',
		about:
			'A blueprint and a gear: a short effort with a clear outcome and a deadline, its next actions on the board.',
		d: 'M6 8 H34 V40 H6 Z M10 14 H22 M10 20 H18 M10 26 H24 M38 30 A6 6 0 1 1 37.9 30 Z M38 20 V24 M38 36 V40 M28 30 H32 M44 30 H48'
	},
	area: {
		title: 'Area',
		about:
			'A compass rose, as on the portolan charts: an ongoing responsibility with a standard to keep, and no end date.',
		d: 'M24 4 L28 20 L44 24 L28 28 L24 44 L20 28 L4 24 L20 20 Z M24 20 A4 4 0 1 1 23.9 20 Z'
	},
	resource: {
		title: 'Resource',
		about:
			'A shelf of books kept for later: a topic of ongoing interest, its links and literature notes.',
		d: 'M4 42 H44 M8 42 V12 H14 V42 M16 42 V8 H22 V42 M24 42 V14 H30 V42 M33 42 L37 12 L43 13 L39 42'
	},
	roadmap: {
		title: 'Roadmap',
		about:
			'A winding path from one stage to the next, like the overworld map of Super Mario World: phases, milestones, and the habits that carry you.',
		d: 'M6 40 C16 40 12 28 22 28 C32 28 28 16 40 14 M6 40 A2 2 0 1 1 5.9 40 M22 28 A2 2 0 1 1 21.9 28 M40 14 L40 4 L46 7 L40 10'
	},
	goal: {
		title: 'Goal',
		about:
			'A flag on a summit, as in Celeste’s last climb: a specific, measurable outcome, with its milestones on the way.',
		d: 'M4 42 L18 18 L26 30 L32 22 L44 42 Z M32 22 V6 L42 10 L32 14'
	},
	milestone: {
		title: 'Milestone',
		about: 'A flag on a step: the definition of done for one stage, and the evidence that it is.',
		d: 'M4 42 H16 V32 H28 V22 H40 V42 M34 22 V8 L42 11 L34 14'
	},
	habit: {
		title: 'Habit',
		about:
			'A sprout that grows a little every day: cue, routine, reward, and the identity it votes for.',
		d: 'M24 42 V22 M24 30 C16 30 10 24 10 16 C18 16 24 22 24 30 M24 22 C24 14 30 8 38 8 C38 16 32 22 24 22 M14 42 H34'
	},
	media: {
		title: 'Film, series, or book',
		about:
			'A strip of film beside a book: what you want to see or read, are watching, or loved, and why.',
		d: 'M6 8 H22 V40 H6 Z M6 14 H22 M6 34 H22 M9 10 V12 M9 36 V38 M19 10 V12 M19 36 V38 M28 10 H42 V40 H28 Z M32 16 H38'
	},
	test: {
		title: 'Retrieval test',
		about:
			'A pencil and a question, the testing effect that Make It Stick describes: recall from memory, then check.',
		d: 'M8 40 L12 30 L34 8 L40 14 L18 36 Z M12 30 L18 36 M30 12 L36 18 M34 30 C34 26 42 26 42 31 C42 35 38 35 38 38 M38 42 V43'
	},
	moc: {
		title: 'Map of content',
		about: 'A hub with spokes, like a station on a metro map: the notes on one theme, gathered.',
		d: 'M24 24 m-5 0 a5 5 0 1 0 10 0 a5 5 0 1 0 -10 0 M24 19 V6 M29 24 H42 M24 29 V42 M19 24 H6 M24 6 a2 2 0 1 0 0.1 0 M42 24 a2 2 0 1 0 0.1 0 M24 42 a2 2 0 1 0 0.1 0 M6 24 a2 2 0 1 0 0.1 0'
	},
	journal: {
		title: 'Journal day',
		about: 'An inkwell and a quill: how the day felt, the night, and what you are grateful for.',
		d: 'M14 42 H34 L32 32 H16 Z M18 32 V28 H30 V32 M24 28 C20 18 26 8 40 4 C36 14 32 22 26 28'
	},
	task: {
		title: 'Task',
		about: 'A box to tick: one next action, in its context, with a date if it has one.',
		d: 'M8 8 H40 V40 H8 Z M14 24 L21 31 L34 16'
	},
	workout: {
		title: 'Workout',
		about: 'A dumbbell: the session, its sets, and how hard it felt.',
		d: 'M4 20 V28 M8 14 V34 M14 14 V34 M14 24 H34 M34 14 V34 M40 14 V34 M44 20 V28'
	},
	'food-log': {
		title: 'Food log',
		about: 'A plate and its cutlery: a day’s meals and what they held.',
		d: 'M24 24 m-12 0 a12 12 0 1 0 24 0 a12 12 0 1 0 -24 0 M24 24 m-6 0 a6 6 0 1 0 12 0 a6 6 0 1 0 -12 0 M4 10 V22 M4 16 H8 M8 10 V40 M44 10 C40 14 40 22 44 24 V40'
	},
	meeting: {
		title: 'Meeting',
		about: 'Two speech bubbles: who was there, what was decided, and the next actions it left.',
		d: 'M4 8 H28 V24 H14 L8 30 V24 H4 Z M20 28 V34 H34 L40 40 V34 H44 V18 H32'
	},
	archive: {
		title: 'Archive',
		about: 'A box with its lid on: what is finished or paused, kept out of the way but never lost.',
		d: 'M4 8 H44 V16 H4 Z M8 16 V42 H40 V16 M18 24 H30'
	},
	note: {
		title: 'Note',
		about: 'A sheet of paper: anything worth writing down, in plain Markdown that stays yours.',
		d: sheet
	}
};
noteTypes.film = noteTypes.series = noteTypes.book = noteTypes.media;

/** The illustration for a note type, or a plain sheet. */
export const noteType = (kind: string): NoteType => noteTypes[kind] ?? noteTypes.note;
