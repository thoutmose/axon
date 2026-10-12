import { api, ApiError } from './api';

/**
 * Captures made without a connection (EP-55) wait on this device, in
 * localStorage, and are sent in order when Axon is reachable again. Only
 * the text typed into Capture is kept, and only until it is sent.
 */
export type Waiting = {
	id: string; // sent as the Idempotency-Key, so a capture is created once even if two tabs send it
	kind: 'task' | 'note';
	title: string;
	path?: string;
	content?: string;
	at: string;
};

const key = 'axon.outbox';

export const outbox = $state({ items: [] as Waiting[], sending: false });

function save() {
	try {
		localStorage.setItem(key, JSON.stringify(outbox.items));
	} catch {
		// storage full or blocked: the items stay in memory for this tab
	}
}

export function loadOutbox() {
	try {
		outbox.items = JSON.parse(localStorage.getItem(key) ?? '[]');
	} catch {
		outbox.items = [];
	}
}

/** Keeps a capture for later. */
export function keep(item: Omit<Waiting, 'at' | 'id'>) {
	loadOutbox(); // another tab may have kept or sent some
	outbox.items = [
		...outbox.items,
		{ ...item, id: crypto.randomUUID(), at: new Date().toISOString() }
	];
	save();
}

/** True when an error means "no connection", not "Axon refused". */
export function unreachable(err: unknown): boolean {
	return err instanceof ApiError && err.status === 0;
}

/** Sends what waits, oldest first; stops at the first failure to try again later. */
export async function flush(): Promise<number> {
	if (outbox.sending) return 0;
	// One tab at a time; the idempotency key covers a tab closed mid-send.
	return navigator.locks ? navigator.locks.request('axon-outbox', send) : send();
}

async function send(): Promise<number> {
	outbox.sending = true;
	loadOutbox();
	let sent = 0;
	try {
		while (outbox.items.length) {
			const it = outbox.items[0];
			const headers = { 'Idempotency-Key': `outbox-${it.id}` };
			try {
				if (it.kind === 'task') {
					await api('/v1/tasks', { method: 'POST', body: { title: it.title }, headers });
				} else {
					await api('/v1/notes', {
						method: 'POST',
						body: { path: it.path, content: it.content },
						headers
					});
				}
			} catch (err) {
				if (unreachable(err) || (err instanceof ApiError && err.status === 401)) break;
				// Refused for good (for example a path taken): drop it rather than block the rest.
			}
			outbox.items = outbox.items.slice(1);
			save();
			sent++;
		}
	} finally {
		outbox.sending = false;
	}
	return sent;
}
