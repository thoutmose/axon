/**
 * Quick add (EP-57): one line in Capture becomes the right thing.
 *
 *   Call the plumber @phone tomorrow        a task, context and due date
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

export type Kind =
	'task' | 'note' | 'book' | 'film' | 'series' | 'habit' | 'goal' | 'project' | 'card';

export type Parsed = {
	kind: Kind;
	title: string;
	context?: string; // tasks: "@phone"
	due?: string; // tasks: due date; goals: target date (YYYY-MM-DD)
	status?: 'someday';
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

const iso = (d: Date) =>
	`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

function addDays(d: Date, n: number): Date {
	const out = new Date(d);
	out.setDate(out.getDate() + n);
	return out;
}

/** A date word at the end of the text, as YYYY-MM-DD, and the text without it. */
function takeDate(text: string, today: Date): [string, string | undefined] {
	const rules: [RegExp, (m: RegExpMatchArray) => string | undefined][] = [
		[/\s+(\d{4}-\d{2}-\d{2})$/, (m) => m[1]],
		[/\s+(today|aujourd'hui|aujourd’hui)$/i, () => iso(today)],
		[/\s+(tomorrow|demain)$/i, () => iso(addDays(today, 1))],
		[/\s+(?:in|dans)\s+(\d{1,3})\s+(?:days?|jours?)$/i, (m) => iso(addDays(today, Number(m[1])))],
		[/\s+(?:next\s+week|la\s+semaine\s+prochaine)$/i, () => iso(addDays(today, 7))],
		[
			/\s+(?:on\s+|next\s+)?([a-zé]+)$/i,
			(m) => {
				const wd = weekdays[m[1].toLowerCase()];
				if (wd === undefined) return undefined;
				const diff = (wd - today.getDay() + 7) % 7 || 7; // the next one, never today
				return iso(addDays(today, diff));
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
		if (/(^|\s)(someday|un jour)$/i.test(text)) {
			out.status = 'someday';
			text = text.replace(/\s*(someday|un jour)$/i, '');
		}
		[text, out.due] = takeDate(text.trim(), today);
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
		habit: 'Habit, every day',
		goal: 'Goal',
		project: 'Project',
		card: 'Flashcard'
	};
	const out = [label[p.kind]];
	if (p.context) out.push(p.context);
	if (p.due) out.push(`${p.kind === 'goal' ? 'by' : 'due'} ${p.due}`);
	if (p.status) out.push('someday');
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
		habit: 'Add habit',
		goal: 'Add goal',
		project: 'Add project',
		card: 'Add card'
	}[kind];
}
