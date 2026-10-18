import type { Schemas } from './api';
import { t } from './i18n.svelte';

export type Goal = Schemas['Goal'];
export type Milestone = Schemas['Milestone'];
export type Measure = Schemas['Measure'];
export type Progress = Schemas['Progress'];

/** "50%" or "No progress measure yet", for labels and screen readers. */
export function percentLabel(p: Progress): string {
	return p.percent === null ? t('No progress measure yet') : `${p.percent}%`;
}
