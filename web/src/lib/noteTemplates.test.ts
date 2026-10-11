import { describe, expect, it } from 'vitest';
import {
	builtInFiles,
	dailyContent,
	dailyPath,
	fill,
	isTemplatePath,
	slug,
	templateName,
	templates,
	vaultTemplate,
	zettelId
} from './noteTemplates';

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

describe('vault templates', () => {
	it('recognises the usual template folders', () => {
		expect(isTemplatePath('Templates/Book.md')).toBe(true);
		expect(isTemplatePath('_templates/Book.md')).toBe(true);
		expect(isTemplatePath('99-Templates/Book.md')).toBe(true);
		expect(isTemplatePath('Templates.md')).toBe(false);
		expect(isTemplatePath('Zettel/templates/x.md')).toBe(false);
	});
	it('fills Axon, Obsidian, and Templater placeholders', () => {
		expect(
			fill(
				'# {{title}} {title} <% tp.file.title %>\n{{date}} {{date:YYYY-MM-DD}} {{time}} {{zettel}}',
				'Raft',
				now
			)
		).toBe('# Raft Raft Raft\n2026-10-10 2026-10-10 15:30 202610101530');
	});
	it('files a note by its template type', () => {
		const t = vaultTemplate(
			{ id: '1', path: 'Templates/Book.md', title: 'Book', kind: 'literature' },
			'# {{title}}'
		);
		expect(t.label).toBe('Book');
		expect(templateName('_templates/Weekly review.md')).toBe('Weekly review');
		expect(t.path('Dune', now)).toBe('Zettel/literature/dune.md');
		expect(t.content('Dune', now)).toBe('# Dune');
	});
	it('round-trips the built-ins through the templates folder', () => {
		const files = builtInFiles(now);
		const permanent = files.find((f) => f.path === 'Templates/Permanent note.md')!;
		expect(permanent.content).toContain('id: {{zettel}}');
		expect(fill(permanent.content, 'Proofs', now)).toBe(
			templates.find((t) => t.id === 'permanent')!.content('Proofs', now)
		);
	});
});
