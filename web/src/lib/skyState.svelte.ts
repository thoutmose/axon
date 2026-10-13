import { skyAt } from './sky';

// The sky now (EP-96): the person's time zone (their account's, or the
// browser's before they sign in), the minute, and the chosen landscape.
const zoneKey = 'axon-zone';

function browserZone(): string {
	try {
		return (
			localStorage.getItem(zoneKey) || Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
		);
	} catch {
		return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
	}
}

export const skyState = $state({ zone: 'UTC', now: new Date(), landscape: 'mountains' });

/** The account's time zone, remembered for the next visit. */
export function setZone(zone: string) {
	skyState.zone = zone;
	try {
		localStorage.setItem(zoneKey, zone);
	} catch {
		// Private browsing: this page only.
	}
}

/** Starts the clock; the sky moves every minute. Returns the stop. */
export function startSky(onTick?: () => void): () => void {
	skyState.zone = browserZone();
	skyState.now = new Date();
	const timer = setInterval(() => {
		skyState.now = new Date();
		onTick?.();
	}, 60_000);
	return () => clearInterval(timer);
}

export const currentSky = () => skyAt(skyState.now, skyState.zone);
