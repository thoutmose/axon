/// <reference types="@sveltejs/kit" />
/// <reference no-default-lib="true"/>
/// <reference lib="esnext" />
/// <reference lib="webworker" />

// Installable app (EP-55). The service worker keeps the app's own code and
// the offline page, so Axon opens without a connection and captures wait on
// the device. Nothing personal is cached: the API always goes to the network.
import { build, files, prerendered, version } from '$service-worker';

const sw = self as unknown as ServiceWorkerGlobalScope;
const cache = `axon-${version}`;
const shell = new Set([...build, ...files, ...prerendered]);

sw.addEventListener('install', (event) => {
	event.waitUntil(
		caches
			.open(cache)
			.then((c) => c.addAll([...shell]))
			.then(() => sw.skipWaiting())
	);
});

sw.addEventListener('activate', (event) => {
	event.waitUntil(
		caches
			.keys()
			.then((keys) => Promise.all(keys.filter((k) => k !== cache).map((k) => caches.delete(k))))
			.then(() => sw.clients.claim())
	);
});

sw.addEventListener('fetch', (event) => {
	const req = event.request;
	const url = new URL(req.url);
	if (req.method !== 'GET' || url.origin !== sw.location.origin || url.pathname.startsWith('/api/'))
		return;
	if (shell.has(url.pathname)) {
		event.respondWith(caches.match(url.pathname).then((hit) => hit ?? fetch(req)));
		return;
	}
	if (req.mode === 'navigate') {
		event.respondWith(
			fetch(req).catch(async () => (await caches.match('/offline')) ?? Response.error())
		);
	}
});

// Reminders (EP-56): Axon sends them encrypted; the browser decrypts them
// before this handler sees the text.
sw.addEventListener('push', (event) => {
	let msg = { title: 'Axon', body: '', url: '/today', tag: 'axon' };
	try {
		msg = { ...msg, ...event.data?.json() };
	} catch {
		// keep the defaults
	}
	event.waitUntil(
		sw.registration.showNotification(msg.title, {
			body: msg.body,
			tag: msg.tag,
			icon: '/icons/icon-192.png',
			badge: '/icons/icon-192.png',
			data: { url: msg.url }
		})
	);
});

sw.addEventListener('notificationclick', (event) => {
	event.notification.close();
	const url = new URL(event.notification.data?.url ?? '/today', sw.location.origin).href;
	event.waitUntil(
		sw.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((wins) => {
			const open = wins.find((w) => new URL(w.url).origin === sw.location.origin);
			return open ? open.navigate(url).then((w) => w?.focus()) : sw.clients.openWindow(url);
		})
	);
});
