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
