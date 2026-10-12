import { readdirSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { pageAgents } from './pageAgents';

// The agents a page offers must exist, and every suggestion must belong to one of them.
const skills = readdirSync(new URL('../../../api/internal/agents/skills', import.meta.url));

describe('page agents', () => {
	it('names only agents that exist, and suggests only for them', () => {
		for (const [page, { agents, suggestions }] of Object.entries(pageAgents)) {
			for (const a of agents) expect(skills, `${page}: ${a}`).toContain(a);
			for (const a of Object.keys(suggestions)) expect(agents, `${page}: ${a}`).toContain(a);
		}
	});

	it('gives every page its own agent first, with something to ask', () => {
		for (const { agents, suggestions } of Object.values(pageAgents)) {
			expect(suggestions[agents[0]]?.length).toBeGreaterThan(0);
		}
	});
});
