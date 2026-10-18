/** Shell-wide connection state: browser offline, or API unreachable. Read it
 * in the browser only: on the server one module serves every visitor. */
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
