import { describe, expect, it } from 'vitest';
import { dailyContent, dailyPath, slug, templates, zettelId } from './noteTemplates';

const now = new Date(2026, 9, 10, 15, 30); // local 10 Oct 2026 15:30

describe('templates', () => {
	it('places notes by Zettelkasten and PARA conventions', () => {
		const paths = Object.fromEntries(templates.map((t) => [t.id, t.path('Raft & Paxos!', now)]));
		expect(paths).toMatchObject({
			fleeting: '0-Inbox/202610101530-raft-paxos.md',
			literature: 'Zettel/literature/raft-paxos.md',
			permanent: 'Zettel/permanent/202610101530-raft-paxos.md',
			project: '1-Projects/raft-paxos/raft-paxos.md',
			area: '2-Areas/raft-paxos.md',
			resource: '3-Resources/raft-paxos.md'
		});
	});
	it('writes front matter the API reads', () => {
		const permanent = templates.find((t) => t.id === 'permanent')!;
		expect(permanent.content('Proofs', now)).toMatch(
			/^---\ntype: permanent\nid: 202610101530\ntitle: Proofs\n/
		);
	});
	it('makes a daily note linked to yesterday', () => {
		expect(dailyPath(now)).toBe('Daily/2026-10-10.md');
		expect(dailyContent(now)).toContain('Previous: [[2026-10-09]]');
	});
	it('slugs like the API and ids sort by time', () => {
		expect(slug('Réviser le japonais')).toBe('réviser-le-japonais');
		expect(slug('!!!')).toBe('note');
		expect(zettelId(new Date(2026, 0, 2, 3, 4))).toBe('202601020304');
	});
});

import { fill } from './noteTemplates';

describe('fill', () => {
	it('fills Obsidian-style placeholders', () => {
		const out = fill(
			'# {{title}}\ncreated {{date}} · {title} · {date}',
			'Raft',
			new Date('2026-10-11T10:00:00')
		);
		expect(out).toBe('# Raft\ncreated 2026-10-11 · Raft · 2026-10-11');
	});
});
