export type Toast = { id: number; kind: 'ok' | 'error'; text: string };

let next = 0;

/** Messages announced through the shell's live region and shown briefly. */
export const toasts = $state<Toast[]>([]);

export function toast(text: string, kind: Toast['kind'] = 'ok') {
	const id = ++next;
	toasts.push({ id, kind, text });
	setTimeout(() => dismiss(id), kind === 'error' ? 8000 : 4000);
}

export function dismiss(id: number) {
	const i = toasts.findIndex((t) => t.id === id);
	if (i >= 0) toasts.splice(i, 1);
}
