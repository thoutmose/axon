import { describe, expect, it } from 'vitest';
import apiTemplates from '../../../api/internal/vault/templates.json';
import {
	builtInFiles,
	dailyContent,
	dailyPath,
	fill,
	isTemplatePath,
	mediaNote,
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
			literature: 'Zettel/literature/202610101530-raft-paxos.md',
			permanent: 'Zettel/permanent/202610101530-raft-paxos.md',
			project: '1-Projects/raft-paxos/raft-paxos.md',
			area: '2-Areas/raft-paxos/raft-paxos.md',
			resource: '3-Resources/raft-paxos.md',
			// The owner's vault (EP-100): plans in Tracker/, periodic notes, and retrieval tests.
			roadmap: 'Roadmaps/raft-paxos.md',
			goal: 'Tracker/goals/raft-paxos.md',
			milestone: 'Tracker/milestones/raft-paxos.md',
			habit: 'Tracker/habits/raft-paxos.md',
			media: 'Tracker/media/raft-paxos.md',
			weekly: 'Weekly/2026-W41.md',
			monthly: 'Monthly/2026-10.md',
			'test-5w': 'Reviews/raft-paxos/5w.md',
			'test-feynman': 'Reviews/raft-paxos/feynman.md'
		});
	});
	it('writes front matter the API reads', () => {
		const permanent = templates.find((t) => t.id === 'permanent')!;
		expect(permanent.content('Proofs', now)).toMatch(
			/^---\ntype: permanent\ntitle: Proofs\ncreated: 2026-10-10 15:30\nstatus: draft\n/
		);
		for (const tmpl of templates) {
			expect(tmpl.content('Proofs', now), tmpl.id).not.toContain('{{');
		}
		const goal = templates.find((t) => t.id === 'goal')!.content('Top 5%', now);
		expect(goal).toContain('Milestones link here with `goal: "[[top-5]]"`');
		const test = templates.find((t) => t.id === 'test-5w')!.content('Raft', now);
		expect(test).toContain('note: "[[Raft]]"');
	});
	it('makes a daily note from the vault’s daily template', () => {
		expect(dailyPath(now)).toBe('Daily/2026-10-10.md');
		expect(dailyContent(now)).toMatch(
			/^---\ntype: daily\ntitle: 2026-10-10\ncreated: 2026-10-10 15:30\n/
		);
		expect(dailyContent(now)).toContain('## Top 3');
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
		expect(t.path('Dune', now)).toBe('Zettel/literature/202610101530-dune.md');
		const task = vaultTemplate(
			{ id: '2', path: 'Templates/Chore.md', title: 'Chore', kind: 'task' },
			'',
			'Life'
		);
		expect(task.path('Mow the lawn', now)).toBe('Life/Tasks/Mow the lawn.md');
		expect(t.content('Dune', now)).toBe('# Dune');
	});
	it('round-trips the built-ins through the templates folder', () => {
		const files = builtInFiles(now);
		const permanent = files.find((f) => f.path === 'Templates/permanent.md')!;
		expect(permanent.content).toContain('created: {{created}}');
		expect(fill(permanent.content, 'Proofs', now)).toBe(
			templates.find((t) => t.id === 'permanent')!.content('Proofs', now)
		);
	});
});

// The API writes the same templates into an empty vault (EP-106): its copy
// must follow this file. Regenerate it from builtInFiles when one changes.
describe('the templates an empty vault starts with', () => {
	it('match the API copy', () => {
		expect(builtInFiles(new Date(2026, 0, 2, 3, 4))).toEqual(apiTemplates);
	});
});

describe('a note for a film, series, or book', () => {
	it('fills its template from the item, as a literature note', () => {
		const n = mediaNote(
			{
				kind: 'book',
				title: 'Dune',
				creators: 'Frank Herbert',
				year: 1965,
				genres: ['Science fiction'],
				total: 612,
				cover_id: null
			},
			new Date(2026, 0, 2, 3, 4)
		);
		expect(n.path).toBe('Zettel/literature/202601020304-dune.md');
		expect(n.content).toContain('type: literature\nsource_type: book\ntitle: "Dune"');
		expect(n.content).toContain('author: Frank Herbert\nyear: 1965\ngenres: [Science fiction]');
		expect(n.content).toContain('pages: 612');
		expect(n.content).toContain('## Reading log');
		expect(n.content).not.toMatch(/\{\{/);
	});
});
