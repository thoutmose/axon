import { api, message } from './api';
import { toast } from './ui/toast.svelte';

/**
 * Says what was deleted, with Undo (EP-66): the deletion is in the trash for
 * 30 days, and Undo puts it back with what was deleted along with it.
 */
export function deleted(text: string, entityId: string, after?: () => void) {
	toast(text, 'ok', {
		label: 'Undo',
		run: async () => {
			try {
				await api('/v1/trash/undo', { method: 'POST', body: { entity_id: entityId } });
				toast('Put back.');
				after?.();
			} catch (err) {
				toast(message(err), 'error');
			}
		}
	});
}
