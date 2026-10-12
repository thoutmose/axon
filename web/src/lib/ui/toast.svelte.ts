export type Toast = {
	id: number;
	kind: 'ok' | 'error';
	text: string;
	action?: { label: string; run: () => void };
};

let next = 0;

/** Messages announced through the shell's live region and shown briefly. */
export const toasts = $state<Toast[]>([]);

/** Shows a message; with an action (Undo), it stays longer so there is time to use it. */
export function toast(text: string, kind: Toast['kind'] = 'ok', action?: Toast['action']) {
	const id = ++next;
	toasts.push({ id, kind, text, action });
	setTimeout(() => dismiss(id), kind === 'error' || action ? 8000 : 4000);
}

export function dismiss(id: number) {
	const i = toasts.findIndex((t) => t.id === id);
	if (i >= 0) toasts.splice(i, 1);
}
