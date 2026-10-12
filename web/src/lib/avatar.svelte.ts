// The signed-in person's picture, shared by the header and the account page.
export const avatar = $state({ signedIn: false, version: '' });

/** The picture URL; the version changes it after an upload, so caches refresh. */
export const avatarSrc = (version: string) =>
	`/api/v1/me/avatar${version ? `?v=${encodeURIComponent(version)}` : ''}`;
