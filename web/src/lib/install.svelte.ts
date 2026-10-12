/** The browser's install offer, kept from page load so Preferences can show it (EP-55). */
type InstallEvent = Event & {
	prompt: () => Promise<void>;
	userChoice: Promise<{ outcome: string }>;
};

export const install = $state({ offer: null as InstallEvent | null, installed: false });

export function watchInstall() {
	install.installed = matchMedia('(display-mode: standalone)').matches;
	const offer = (e: Event) => {
		e.preventDefault();
		install.offer = e as InstallEvent;
	};
	const done = () => {
		install.installed = true;
		install.offer = null;
	};
	addEventListener('beforeinstallprompt', offer);
	addEventListener('appinstalled', done);
	return () => {
		removeEventListener('beforeinstallprompt', offer);
		removeEventListener('appinstalled', done);
	};
}

export async function promptInstall() {
	const e = install.offer;
	if (!e) return;
	await e.prompt();
	if ((await e.userChoice).outcome === 'accepted') install.installed = true;
	install.offer = null;
}
