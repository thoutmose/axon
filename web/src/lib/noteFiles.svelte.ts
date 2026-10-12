import { api, type Schemas } from './api';

export type NoteFile = Schemas['NoteFile'];

/** The pictures and PDFs notes embed, for resolving ![[x.png]] as you type. */
export const noteFiles = $state({ files: [] as NoteFile[], loaded: false });

export async function refreshFiles(): Promise<void> {
	const res = await api<{ items: NoteFile[] }>('/v1/notes/files');
	noteFiles.files = res.items;
	noteFiles.loaded = true;
}

export const pictureTypes = ['image/png', 'image/jpeg', 'image/gif', 'image/webp', 'image/svg+xml'];
const accepted = [...pictureTypes, 'application/pdf'];
const extensions: Record<string, string> = {
	'image/png': 'png',
	'image/jpeg': 'jpg',
	'image/gif': 'gif',
	'image/webp': 'webp',
	'image/svg+xml': 'svg',
	'application/pdf': 'pdf'
};

export function canUpload(file: File): boolean {
	return accepted.includes(file.type);
}

/** Uploads a picture or PDF and returns the Markdown that embeds it. */
export async function upload(file: File): Promise<string> {
	const ext = extensions[file.type];
	// Pasted screenshots are called "image.png"; give the name the right extension.
	const base = file.name.replace(/\.[^.]*$/, '') || 'picture';
	const res = await fetch(`/api/v1/notes/files?name=${encodeURIComponent(`${base}.${ext}`)}`, {
		method: 'POST',
		headers: { 'Content-Type': file.type },
		body: file
	});
	const body = await res.json().catch(() => ({}));
	if (!res.ok) throw new Error(body.detail ?? res.statusText);
	const f = body as NoteFile;
	noteFiles.files = [...noteFiles.files.filter((x) => x.path !== f.path), f];
	const alt = base.replace(/[[\]]/g, '');
	return f.content_type === 'application/pdf' ? `[${alt}](${f.path})` : `![${alt}](${f.path})`;
}

export function fileURL(path: string): string {
	return `/api/v1/notes/file?path=${encodeURIComponent(path)}`;
}

/**
 * Finds the file an embed means, as Obsidian does: the exact vault path,
 * then a path relative to the note's folder, then any file with that name.
 */
export function resolveFile(
	target: string,
	notePath = '',
	files = noteFiles.files
): NoteFile | undefined {
	let t = target.trim();
	try {
		t = decodeURIComponent(t);
	} catch {
		// keep the raw text
	}
	t = t.replace(/^\.\//, '').replace(/^\//, '');
	const lower = t.toLowerCase();
	const dir = notePath.includes('/') ? notePath.replace(/\/[^/]*$/, '/') : '';
	return (
		files.find((f) => f.path.toLowerCase() === lower) ??
		files.find((f) => f.path.toLowerCase() === (dir + t).toLowerCase()) ??
		(t.includes('/')
			? undefined
			: files.find((f) => f.path.toLowerCase().replace(/^.*\//, '') === lower))
	);
}
