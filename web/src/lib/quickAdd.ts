/**
 * Quick add (EP-57): one line in Capture becomes the right thing.
 *
 *   Call the plumber @phone tomorrow        a task, context and due date
 *   Call the bank friday at 15:00           …and a reminder then ("à 15h" too)
 *   note: idea for the garden #ideas        a fleeting note, tagged
 *   book: Atomic Habits                     a book to read
 *   film: / series:                         a film or series to watch
 *   habit: Read one page                    a habit, every day
 *   goal: Run a 10 km race by 2027-05-01    a goal with a target date
 *   project: Garden shed                    a project
 *   card: What is spacing? :: Reviewing just before forgetting #Learning
 *
 * French works too: tâche, livre, série, habitude, objectif, projet, carte;
 * aujourd'hui, demain, lundi…; "d'ici" for a goal's date.
 */

import { addDays, localDate } from './dates';

export type Kind =
	'task' | 'note' | 'book' | 'film' | 'series' | 'game' | 'habit' | 'goal' | 'project' | 'card';

export type Parsed = {
	kind: Kind;
	title: string;
	context?: string; // tasks: "@phone"
	due?: string; // tasks: due date; goals: target date (YYYY-MM-DD)
	status?: 'someday';
	remind?: Date; // tasks: a reminder at this time (EP-68)
	tags: string[]; // notes: tags; cards: the deck
	back?: string; // cards: the answer
};

const prefixes: Record<string, Kind> = {
	task: 'task',
	todo: 'task',
	tâche: 'task',
	tache: 'task',
	note: 'note',
	idea: 'note',
	idée: 'note',
	book: 'book',
	livre: 'book',
	film: 'film',
	movie: 'film',
	series: 'series',
	show: 'series',
	série: 'series',
	serie: 'series',
	game: 'game',
	jeu: 'game',
	habit: 'habit',
	habitude: 'habit',
	goal: 'goal',
	objectif: 'goal',
	project: 'project',
	projet: 'project',
	card: 'card',
	flashcard: 'card',
	carte: 'card'
};

// Short forms that are also words or names ("Call Sam", "mon", "mer") are left out.
const weekdays: Record<string, number> = {
	sunday: 0, dimanche: 0,
	monday: 1, lundi: 1,
	tuesday: 2, tue: 2, mardi: 2,
	wednesday: 3, wed: 3, mercredi: 3,
	thursday: 4, thu: 4, jeudi: 4,
	friday: 5, fri: 5, vendredi: 5,
	saturday: 6, samedi: 6
}; // prettier-ignore

/** A date word at the end of the text, as YYYY-MM-DD, and the text without it. */
function takeDate(text: string, now: Date): [string, string | undefined] {
	const today = localDate(now);
	const rules: [RegExp, (m: RegExpMatchArray) => string | undefined][] = [
		[/\s+(\d{4}-\d{2}-\d{2})$/, (m) => m[1]],
		[/\s+(today|aujourd'hui|aujourd’hui)$/i, () => today],
		[/\s+(tomorrow|demain)$/i, () => addDays(today, 1)],
		[/\s+(?:in|dans)\s+(\d{1,3})\s+(?:days?|jours?)$/i, (m) => addDays(today, Number(m[1]))],
		[/\s+(?:next\s+week|la\s+semaine\s+prochaine)$/i, () => addDays(today, 7)],
		[
			/\s+(?:on\s+|next\s+)?([a-zé]+)$/i,
			(m) => {
				const wd = weekdays[m[1].toLowerCase()];
				if (wd === undefined) return undefined;
				const diff = (wd - now.getDay() + 7) % 7 || 7; // the next one, never today
				return addDays(today, diff);
			}
		]
	];
	for (const [re, f] of rules) {
		const m = text.match(re);
		const d = m ? f(m) : undefined;
		if (m && d) return [text.slice(0, m.index).trimEnd(), d];
	}
	return [text, undefined];
}

/** Reads one line of quick add. */
export function parse(input: string, today = new Date()): Parsed {
	let text = input.trim().replace(/\s+/g, ' ');
	let kind: Kind = 'task';
	const prefix = text.match(/^([\p{L}]+)\s*:\s*(.*)$/u);
	if (prefix && prefixes[prefix[1].toLowerCase()]) {
		kind = prefixes[prefix[1].toLowerCase()];
		text = prefix[2];
	}
	const out: Parsed = { kind, title: text, tags: [] };
	// #tags anywhere (notes and cards); kept in a task's title, where they read as text.
	if (kind === 'note' || kind === 'card') {
		text = text.replace(/(^|\s)#([\p{L}\p{N}_/-]+)/gu, (_, pre: string, tag: string) => {
			out.tags.push(tag);
			return pre;
		});
	}
	if (kind === 'card') {
		const [front, back] = text.split(/\s+::\s+/, 2);
		out.title = front.trim();
		out.back = back?.trim() ?? '';
		return out;
	}
	if (kind === 'task') {
		text = text.replace(/(^|\s)(@[\p{L}\p{N}_-]+)/u, (_, pre: string, ctx: string) => {
			out.context = ctx;
			return pre;
		});
		// A time at the end, "at 15:00", "at 3pm", "à 15h30": a reminder.
		const time = text.match(/\s+(?:at|à)\s+(\d{1,2})(?:[:h](\d{2}))?\s*(am|pm|h)?$/i);
		let hour: number | undefined,
			minute = 0;
		if (time) {
			hour = Number(time[1]) % 24;
			minute = Number(time[2] ?? 0);
			if (time[3]?.toLowerCase() === 'pm' && hour < 12) hour += 12;
			if (time[3]?.toLowerCase() === 'am' && hour === 12) hour = 0;
			if (minute > 59) hour = undefined;
			else text = text.slice(0, time.index).trimEnd();
		}
		if (/(^|\s)(someday|un jour)$/i.test(text)) {
			out.status = 'someday';
			text = text.replace(/\s*(someday|un jour)$/i, '');
		}
		[text, out.due] = takeDate(text.trim(), today);
		if (hour !== undefined) {
			const d = out.due ? new Date(`${out.due}T00:00:00`) : new Date(today);
			d.setHours(hour, minute, 0, 0);
			// Today's hour already gone, without a day said: the next one.
			if (!out.due && d.getTime() < today.getTime()) d.setDate(d.getDate() + 1);
			out.remind = d;
		}
	}
	if (kind === 'goal') {
		const by = text.match(/\s+(?:by|d'ici|d’ici|avant)\s+(.+)$/i);
		if (by) {
			const [rest, date] = takeDate(` ${by[1]}`, today);
			if (date && !rest.trim()) {
				out.due = date;
				text = text.slice(0, by.index);
			}
		}
	}
	out.title = text.trim().replace(/\s+/g, ' ');
	return out;
}

/** What quick add will do, in a few words: "Book · Atomic Habits". */
export function describe(p: Parsed): string[] {
	const label: Record<Kind, string> = {
		task: 'Task',
		note: 'Note',
		book: 'Book to read',
		film: 'Film to watch',
		series: 'Series to watch',
		game: 'Game to play',
		habit: 'Habit, every day',
		goal: 'Goal',
		project: 'Project',
		card: 'Flashcard'
	};
	const out = [label[p.kind]];
	if (p.context) out.push(p.context);
	if (p.due) out.push(`${p.kind === 'goal' ? 'by' : 'due'} ${p.due}`);
	if (p.status) out.push('someday');
	if (p.remind)
		out.push(
			`remind ${p.remind.toLocaleDateString(undefined, { weekday: 'short' })} ${String(p.remind.getHours()).padStart(2, '0')}:${String(p.remind.getMinutes()).padStart(2, '0')}`
		);
	if (p.tags.length)
		out.push(p.kind === 'card' ? `deck ${p.tags[0]}` : p.tags.map((t) => `#${t}`).join(' '));
	if (p.kind === 'card' && !p.back) out.push('add “ :: answer”');
	return out;
}

/** The button's words for what will be added. */
export function action(kind: Kind): string {
	return {
		task: 'Capture task',
		note: 'Save note',
		book: 'Add book',
		film: 'Add film',
		series: 'Add series',
		game: 'Add game',
		habit: 'Add habit',
		goal: 'Add goal',
		project: 'Add project',
		card: 'Add card'
	}[kind];
}
