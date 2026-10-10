/** Shell-wide connection state: browser offline, or API unreachable. */
export const connection = $state({ online: true, apiDown: false });

export function watchConnection() {
	connection.online = navigator.onLine;
	const update = () => (connection.online = navigator.onLine);
	addEventListener('online', update);
	addEventListener('offline', update);
	return () => {
		removeEventListener('online', update);
		removeEventListener('offline', update);
	};
}
