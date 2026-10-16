<script lang="ts" module>
	import { t } from '$lib/i18n.svelte';
	import type { Task } from '$lib/tasks';

	export type BoardColumn = {
		/** The status a card gets when it is dropped here. */
		status: Task['status'];
		label: string;
		hint: string;
		tasks: Task[];
		/** Show the cards under one heading per GTD context. */
		byContext?: boolean;
	};
</script>

<script lang="ts">
	import { SvelteSet } from 'svelte/reactivity';
	import { fold, overdue, relativeDay } from '$lib/tasks';

	let {
		label,
		columns,
		today,
		onmove,
		onopen,
		ontoggle
	}: {
		label: string;
		columns: BoardColumn[];
		today: string;
		onmove: (task: Task, status: Task['status']) => void;
		onopen: (id: string) => void;
		ontoggle: (task: Task) => void;
	} = $props();

	let dragging = $state<Task | null>(null);
	let over = $state<string | null>(null);
	// Long columns fold after a few cards (EP-128); a column opened, or a card
	// just moved, stays in sight.
	const opened = new SvelteSet<string>();
	const moved = new SvelteSet<string>();

	function move(task: Task, status: Task['status']) {
		moved.add(task.id);
		onmove(task, status);
	}

	function groups(col: BoardColumn): [string, Task[]][] {
		if (!col.byContext) return [['', col.tasks]];
		const out = new Map<string, Task[]>();
		for (const task of col.tasks) {
			const key = task.context ?? t('No context');
			out.set(key, [...(out.get(key) ?? []), task]);
		}
		return [...out].sort(([a], [b]) =>
			a === 'No context' ? 1 : b === 'No context' ? -1 : a.localeCompare(b)
		);
	}

	// Shift+Arrow moves a card to the neighbouring column, as on project boards.
	function onkeydown(e: KeyboardEvent, task: Task, at: number) {
		if (!e.shiftKey || (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight')) return;
		const to = columns[at + (e.key === 'ArrowLeft' ? -1 : 1)];
		if (!to) return;
		e.preventDefault();
		move(task, to.status);
	}

	function drop(e: DragEvent, col: BoardColumn) {
		e.preventDefault();
		over = null;
		if (dragging && dragging.status !== col.status) move(dragging, col.status);
		dragging = null;
	}
</script>

<div class="board" role="group" aria-label={label}>
	{#each columns as col, i (col.status)}
		{@const folded = opened.has(col.status)
			? { groups: groups(col), hidden: 0 }
			: fold(groups(col), moved)}
		<section
			class="column"
			class:over={over === col.status}
			aria-labelledby="col-{col.status}"
			ondragover={(e) => {
				e.preventDefault();
				over = col.status;
			}}
			ondragleave={() => (over = null)}
			ondrop={(e) => drop(e, col)}
		>
			<h2 id="col-{col.status}">{col.label} <span class="count">{col.tasks.length}</span></h2>
			<p class="hint">{col.hint}</p>
			{#each folded.groups as [context, tasks] (context)}
				{#if context}<h3>{context}</h3>{/if}
				<ul>
					{#each tasks as item (item.id)}
						<li
							draggable="true"
							ondragstart={() => (dragging = item)}
							ondragend={() => (dragging = null)}
							class:done={item.status === 'done'}
						>
							<input
								type="checkbox"
								checked={item.status === 'done'}
								onchange={() => ontoggle(item)}
								aria-label={t('Complete {title}', { title: item.title })}
							/>
							<button
								class="title"
								onclick={() => onopen(item.id)}
								onkeydown={(e) => onkeydown(e, item, i)}>{item.title}</button
							>
							<span class="meta">
								{#if item.recurrence}<span title={t('Repeats')} aria-label={t('Repeats')}>↻</span
									>{/if}
								{#if item.context && !col.byContext}<span class="chip">{item.context}</span>{/if}
								{#if item.due_on}<span class:overdue={overdue(item, today)}
										>{t('Due {value}', { value: relativeDay(item.due_on, today) })}</span
									>
								{:else if item.scheduled_on}<span>{relativeDay(item.scheduled_on, today)}</span
									>{/if}
							</span>
							<label class="visually-hidden" for="move-{item.id}"
								>{t('Move {title} to', { title: item.title })}</label
							>
							<select
								id="move-{item.id}"
								value={item.status}
								onchange={(e) => move(item, e.currentTarget.value as Task['status'])}
							>
								{#each columns as c (c.status)}<option value={c.status}>{c.label}</option>{/each}
							</select>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="empty">{t('Nothing here.')}</p>
			{/each}
			{#if folded.hidden}
				<button type="button" class="more" onclick={() => opened.add(col.status)}
					>{t('Show {n} more', { n: folded.hidden })}</button
				>
			{:else if opened.has(col.status)}
				<button type="button" class="more" onclick={() => opened.delete(col.status)}
					>{t('Show less')}</button
				>
			{/if}
		</section>
	{/each}
</div>
<p class="keys">
	{t('Drag a card to another column, use its menu, or focus its title and press')}
	<kbd>Shift</kbd>+<kbd>←</kbd>/<kbd>→</kbd>.
</p>

<style>
	.board {
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: minmax(16rem, 1fr);
		gap: var(--space-4);
		overflow-x: auto;
		align-items: start;
		padding-bottom: var(--space-2);
	}
	.column {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-3);
		min-height: 12rem;
		transition: border-color var(--motion-fast) var(--ease);
	}
	.column.over {
		border-color: var(--accent);
		background: var(--accent-soft);
	}
	h2 {
		font-size: var(--text-base);
		margin: 0;
		display: flex;
		justify-content: space-between;
	}
	h3 {
		font-size: var(--text-sm);
		color: var(--muted);
		margin: var(--space-3) 0 var(--space-1);
	}
	.count {
		color: var(--muted);
		font-weight: 400;
	}
	.hint,
	.empty,
	.keys {
		color: var(--muted);
		font-size: var(--text-sm);
		margin: var(--space-1) 0 var(--space-2);
	}
	ul {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: var(--space-2);
	}
	li {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
		background: var(--surface-solid);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-2) var(--space-3);
		cursor: grab;
	}
	li:focus-within {
		border-color: var(--accent);
	}
	input[type='checkbox'] {
		width: 1.1rem;
		height: 1.1rem;
		accent-color: var(--accent);
	}
	.title {
		flex: 1 1 calc(100% - 2rem); /* the title gets its own line; meta and menu go below */
		min-width: 0;
		text-align: left;
		font: inherit;
		color: var(--fg);
		background: none;
		border: none;
		padding: var(--space-1) 0;
		cursor: pointer;
	}
	.done .title {
		color: var(--muted);
		text-decoration: line-through;
	}
	.meta {
		display: flex;
		gap: var(--space-2);
		color: var(--muted);
		font-size: var(--text-sm);
	}
	.chip {
		background: var(--accent-soft);
		color: var(--fg);
		border-radius: 999px;
		padding: 0 var(--space-2);
	}
	.overdue {
		color: var(--bad);
		font-weight: 600;
	}
	select {
		font: inherit;
		font-size: var(--text-sm);
		color: var(--fg);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		padding: 0 var(--space-1);
		margin-left: auto;
	}
	.more {
		width: 100%;
		margin-top: var(--space-2);
		padding: var(--space-2);
		font: inherit;
		font-size: var(--text-sm);
		color: var(--accent);
		background: none;
		border: 1px dashed var(--border);
		border-radius: var(--radius);
		cursor: pointer;
	}
	.more:hover {
		border-color: var(--accent);
	}
	kbd {
		font-family: var(--font-mono);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		padding: 0 0.3em;
	}
</style>
