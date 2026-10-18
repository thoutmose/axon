import { t } from './i18n.svelte';

/** The three scales of the journal (EP-91), from 1 to 5. */
export const scales = () => ({
	mood: {
		legend: t('Mood'),
		kind: 'mood' as const,
		labels: [t('Very low'), t('Low'), t('Okay'), t('Good'), t('Great')]
	},
	energy: {
		legend: t('Energy'),
		kind: 'energy' as const,
		labels: [t('Drained'), t('Tired'), t('Okay'), t('Good'), t('Full of energy')]
	},
	stress: {
		legend: t('Stress'),
		kind: 'stress' as const,
		labels: [t('Calm'), t('A little'), t('Some'), t('A lot'), t('Overwhelmed')]
	}
});

export type JournalFields = {
	mood: number | null;
	energy: number | null;
	stress: number | null;
	sleep_hours: number | null;
	gratitude: string[];
	note: string;
};

export const emptyDay = (): JournalFields => ({
	mood: null,
	energy: null,
	stress: null,
	sleep_hours: null,
	gratitude: ['', '', ''],
	note: ''
});

/** Whether a day has anything worth saving: the API refuses an empty one. */
/** The fields an entry is written with: the API takes no others (not its
 * day or dates), and refuses the whole body if they are sent. */
export function fieldsOf(e: JournalFields): JournalFields {
	return {
		mood: e.mood,
		energy: e.energy,
		stress: e.stress,
		sleep_hours: e.sleep_hours,
		gratitude: e.gratitude,
		note: e.note
	};
}

export function hasContent(f: JournalFields): boolean {
	return (
		f.mood != null ||
		f.energy != null ||
		f.stress != null ||
		f.sleep_hours != null ||
		f.gratitude.some((g) => g.trim()) ||
		f.note.trim() !== ''
	);
}

export type ScaleKind = 'mood' | 'energy' | 'stress';

/**
 * The line drawing of a step from 1 to 5 (EP-129), on a 24 × 24 grid: a face
 * whose mouth curves from a frown to a smile, a battery that fills, or a gauge
 * whose needle swings from calm to overwhelmed.
 */
export function scaleArt(kind: ScaleKind, step: number): string {
	const k = step - 1; // 0 to 4
	if (kind === 'mood') {
		const mouth = 15.5 + (k - 2) * 1.6; // the control point: below the line smiles
		return (
			'<circle cx="12" cy="12" r="9.5"/>' +
			'<path d="M9 9.5v.5M15 9.5v.5"/>' +
			`<path d="M8 15.5Q12 ${mouth} 16 15.5"/>`
		);
	}
	if (kind === 'energy') {
		const bars = Array.from(
			{ length: k },
			(_, i) =>
				`<rect x="${4.5 + i * 3.6}" y="9.5" width="2" height="5" rx=".5" fill="currentColor" stroke="none"/>`
		).join('');
		return '<rect x="2" y="7" width="18" height="10" rx="2.5"/><path d="M22.5 10.5v3"/>' + bars;
	}
	const a = Math.PI - (k * Math.PI) / 4; // from the left (calm) to the right
	const x = (12 + 6.5 * Math.cos(a)).toFixed(2);
	const y = (17 - 6.5 * Math.sin(a)).toFixed(2);
	return (
		'<path d="M3 17a9 9 0 0 1 18 0"/>' +
		`<path d="M12 17L${x} ${y}"/>` +
		'<circle cx="12" cy="17" r="1.2" fill="currentColor"/>'
	);
}
