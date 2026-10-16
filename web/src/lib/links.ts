import type { Schemas } from './api';

export type Relation = Schemas['Relation'];
export type Link = Schemas['Link'];
export type Entity = Schemas['Entity'];

/** How a relation reads from each end. */
const wording: Record<string, { forward: string; reverse: string }> = {
	relates_to: { forward: 'Related to', reverse: 'Related to' },
	supports: { forward: 'Supports', reverse: 'Supported by' },
	blocks: { forward: 'Blocks', reverse: 'Blocked by' },
	references: { forward: 'Mentions', reverse: 'Mentioned in' },
	in_area: { forward: 'In area', reverse: 'Includes' }
};

/** Relations made automatically (wikilinks) are shown but not offered. */
const automatic = new Set(['references']);

export function label(relation: string, outgoing: boolean): string {
	const w = wording[relation];
	if (!w) return relation.replaceAll('_', ' ');
	return outgoing ? w.forward : w.reverse;
}

export type Option = { relation: string; outgoing: boolean; label: string };

/** Ways `self` can be connected to `other`, in either direction. */
export function options(relations: Relation[], selfType: string, otherType: string): Option[] {
	const fits = (list: string[], t: string) => list.length === 0 || list.includes(t);
	const out: Option[] = [];
	for (const r of relations) {
		if (automatic.has(r.name)) continue;
		if (fits(r.sources, selfType) && fits(r.targets, otherType)) {
			out.push({ relation: r.name, outgoing: true, label: label(r.name, true) });
		}
		// Symmetric wording needs only one direction.
		if (r.name !== 'relates_to' && fits(r.sources, otherType) && fits(r.targets, selfType)) {
			out.push({ relation: r.name, outgoing: false, label: label(r.name, false) });
		}
	}
	// Specific relations first; the generic one is the fallback.
	return [
		...out.filter((o) => o.relation !== 'relates_to'),
		...out.filter((o) => o.relation === 'relates_to')
	];
}

/** Where an entity lives in the app, if it has its own page. */
export function href(e: { id: string; type: string }): string | null {
	const pages: Record<string, string> = {
		note: '/notes/',
		goal: '/goals/',
		habit: '/habits/',
		task: '/tasks/',
		project: '/projects/',
		roadmap: '/roadmaps/',
		area: '/areas/',
		resource: '/resources/',
		media: '/media/',
		feed_item: '/watch?item='
	};
	return pages[e.type] ? pages[e.type] + e.id : null;
}

export const typeNames: Record<string, string> = {
	task: 'Task',
	note: 'Note',
	goal: 'Goal',
	milestone: 'Milestone',
	habit: 'Habit',
	project: 'Project',
	roadmap: 'Roadmap',
	skill: 'Skill',
	area: 'Area',
	resource: 'Resource',
	media: 'Media',
	feed_item: 'News'
};
