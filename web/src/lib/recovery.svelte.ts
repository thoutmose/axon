import { browserSupportsWebAuthn } from '@simplewebauthn/browser';

/** Recovery codes left, shared by the warning banner and Account › Security. */
export const recovery = $state<{ remaining: number | null }>({ remaining: null });

/** From this many codes down, Axon asks for new ones. */
export const LOW_RECOVERY_CODES = 3;

export const lowOnRecoveryCodes = (remaining: number | null) =>
	remaining !== null && remaining <= LOW_RECOVERY_CODES;

/** Throws the error browsers give when passkeys are unavailable, before a ceremony starts. */
export function requirePasskeys() {
	if (!browserSupportsWebAuthn()) {
		const err = new Error('Passkeys are not supported here.');
		err.name = 'NotSupportedError';
		throw err;
	}
}
