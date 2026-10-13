import { t } from './i18n.svelte';

/** The three scales of the journal (EP-91), from 1 to 5. */
export const scales = () => ({
	mood: {
		legend: t('Mood'),
		faces: ['😞', '😕', '😐', '🙂', '😄'],
		labels: [t('Very low'), t('Low'), t('Okay'), t('Good'), t('Great')]
	},
	energy: {
		legend: t('Energy'),
		faces: ['🪫', '😴', '😐', '🙂', '⚡'],
		labels: [t('Drained'), t('Tired'), t('Okay'), t('Good'), t('Full of energy')]
	},
	stress: {
		legend: t('Stress'),
		faces: ['😌', '🙂', '😐', '😣', '😫'],
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
