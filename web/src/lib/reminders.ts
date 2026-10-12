import { api } from './api';

/** Turns notifications on for this device (EP-56): asks permission, subscribes, and registers. */
export async function enablePush(publicKey: string): Promise<void> {
	if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
		throw new Error(
			'This browser cannot receive notifications. Install Axon first on iPhone or iPad.'
		);
	}
	if ((await Notification.requestPermission()) !== 'granted') {
		throw new Error('Notifications are blocked for Axon in this browser’s settings.');
	}
	const reg = await navigator.serviceWorker.ready;
	const sub =
		(await reg.pushManager.getSubscription()) ??
		(await reg.pushManager.subscribe({
			userVisibleOnly: true,
			applicationServerKey: fromBase64(publicKey)
		}));
	const json = sub.toJSON();
	await api('/v1/me/push-subscriptions', {
		method: 'POST',
		body: {
			endpoint: json.endpoint,
			p256dh: json.keys?.p256dh,
			auth: json.keys?.auth,
			device: deviceName()
		}
	});
}

/** Stops notifications on this device: the browser forgets the subscription too. */
export async function disablePush(id: string): Promise<void> {
	await api(`/v1/me/push-subscriptions/${id}`, { method: 'DELETE' });
	const sub = await (await navigator.serviceWorker?.ready)?.pushManager.getSubscription();
	await sub?.unsubscribe();
}

function fromBase64(s: string): Uint8Array<ArrayBuffer> {
	const b = atob(s.replace(/-/g, '+').replace(/_/g, '/') + '='.repeat((4 - (s.length % 4)) % 4));
	const out = new Uint8Array(new ArrayBuffer(b.length));
	for (let i = 0; i < b.length; i++) out[i] = b.charCodeAt(i);
	return out;
}

/** "Firefox on Linux": enough to tell devices apart, nothing more. */
export function deviceName(ua = navigator.userAgent): string {
	const browser = /Edg\//.test(ua)
		? 'Edge'
		: /Firefox\//.test(ua)
			? 'Firefox'
			: /Chrome\//.test(ua)
				? 'Chrome'
				: /Safari\//.test(ua)
					? 'Safari'
					: 'Browser';
	const os = /Android/.test(ua)
		? 'Android'
		: /iPhone|iPad/.test(ua)
			? 'iOS'
			: /Mac OS X/.test(ua)
				? 'macOS'
				: /Windows/.test(ua)
					? 'Windows'
					: /Linux/.test(ua)
						? 'Linux'
						: 'a device';
	return `${browser} on ${os}`;
}
