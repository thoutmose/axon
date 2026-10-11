<script lang="ts" module>
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
	import { overdue, relativeDay } from '$lib/tasks';

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

	function groups(col: BoardColumn): [string, Task[]][] {
		if (!col.byContext) return [['', col.tasks]];
		const out = new Map<string, Task[]>();
		for (const t of col.tasks) {
			const key = t.context ?? 'No context';
			out.set(key, [...(out.get(key) ?? []), t]);
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
		onmove(task, to.status);
	}

	function drop(e: DragEvent, col: BoardColumn) {
		e.preventDefault();
		over = null;
		if (dragging && dragging.status !== col.status) onmove(dragging, col.status);
		dragging = null;
	}
</script>

<div class="board" role="group" aria-label={label}>
	{#each columns as col, i (col.status)}
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
			{#each groups(col) as [context, tasks] (context)}
				{#if context}<h3>{context}</h3>{/if}
				<ul>
					{#each tasks as t (t.id)}
						<li
							draggable="true"
							ondragstart={() => (dragging = t)}
							ondragend={() => (dragging = null)}
							class:done={t.status === 'done'}
						>
							<input
								type="checkbox"
								checked={t.status === 'done'}
								onchange={() => ontoggle(t)}
								aria-label={`Complete ${t.title}`}
							/>
							<button
								class="title"
								onclick={() => onopen(t.id)}
								onkeydown={(e) => onkeydown(e, t, i)}>{t.title}</button
							>
							<span class="meta">
								{#if t.recurrence}<span title="Repeats" aria-label="Repeats">↻</span>{/if}
								{#if t.context && !col.byContext}<span class="chip">{t.context}</span>{/if}
								{#if t.due_on}<span class:overdue={overdue(t, today)}
										>Due {relativeDay(t.due_on, today)}</span
									>
								{:else if t.scheduled_on}<span>{relativeDay(t.scheduled_on, today)}</span>{/if}
							</span>
							<label class="visually-hidden" for="move-{t.id}">Move {t.title} to</label>
							<select
								id="move-{t.id}"
								value={t.status}
								onchange={(e) => onmove(t, e.currentTarget.value as Task['status'])}
							>
								{#each columns as c (c.status)}<option value={c.status}>{c.label}</option>{/each}
							</select>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="empty">Nothing here.</p>
			{/each}
		</section>
	{/each}
</div>
<p class="keys">
	Drag a card to another column, use its menu, or focus its title and press <kbd>Shift</kbd>+<kbd>←</kbd
	>/<kbd>→</kbd>.
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
		flex: 1;
		min-width: 6rem;
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
	kbd {
		font-family: var(--font-mono);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		padding: 0 0.3em;
	}
</style>
