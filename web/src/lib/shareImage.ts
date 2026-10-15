// Sharing an element (EP-121): a picture of it, drawn here in the browser
// on a canvas at the size social networks show (1200 × 630), and links to
// post the words. Nothing leaves the device until the person shares it.

export type ShareSpec = {
	title: string; // what it is: "Gold trophy", "Read 20 pages"
	value: string; // the big figure: "Level 7", "21 days"
	caption: string; // one sentence
	kind: 'gold' | 'silver' | 'bronze' | 'accent';
};

/**
 * Words into lines no wider than max (by the given measure), at most n
 * lines, the last ending with … when the text goes on. A word wider than
 * max stays whole on its line.
 */
export function wrap(
	measure: (s: string) => number,
	text: string,
	max: number,
	n: number
): string[] {
	const lines: string[] = [];
	let line = '';
	const words = text.split(/\s+/).filter(Boolean);
	for (let i = 0; i < words.length; i++) {
		const next = line ? `${line} ${words[i]}` : words[i];
		if (line && measure(next) > max) {
			lines.push(line);
			line = words[i];
			if (lines.length === n) {
				lines[n - 1] += '…';
				return lines;
			}
		} else line = next;
	}
	if (line) lines.push(line);
	return lines;
}

/** Links that open a post with the text on X, LinkedIn, and Bluesky. */
export function socialLinks(text: string): { name: string; url: string }[] {
	const q = encodeURIComponent(text);
	return [
		{ name: 'X', url: `https://x.com/intent/post?text=${q}` },
		{ name: 'LinkedIn', url: `https://www.linkedin.com/feed/?shareActive=true&text=${q}` },
		{ name: 'Bluesky', url: `https://bsky.app/intent/compose?text=${q}` }
	];
}

const metals = {
	gold: ['#f6d77a', '#b8860b'],
	silver: ['#e3e7ec', '#7d8793'],
	bronze: ['#e6b98a', '#8a5a2b'],
	accent: ['#8fb8ff', '#3b5bdb']
};

/** Draws the card and returns it as a PNG. */
export async function drawCard(spec: ShareSpec, footer: string): Promise<Blob> {
	const W = 1200;
	const H = 630;
	const canvas = document.createElement('canvas');
	canvas.width = W;
	canvas.height = H;
	const g = canvas.getContext('2d')!;
	const [light, deep] = metals[spec.kind];
	// A night sky toward the metal, a glow, and a frame.
	const sky = g.createLinearGradient(0, 0, W, H);
	sky.addColorStop(0, '#0f1424');
	sky.addColorStop(1, '#1d2440');
	g.fillStyle = sky;
	g.fillRect(0, 0, W, H);
	const glow = g.createRadialGradient(W * 0.82, H * 0.2, 10, W * 0.82, H * 0.2, 520);
	glow.addColorStop(0, light + 'aa');
	glow.addColorStop(1, 'transparent');
	g.fillStyle = glow;
	g.fillRect(0, 0, W, H);
	g.strokeStyle = deep;
	g.lineWidth = 6;
	g.strokeRect(24, 24, W - 48, H - 48);
	// A medal.
	g.beginPath();
	g.arc(W - 200, 200, 90, 0, Math.PI * 2);
	const medal = g.createLinearGradient(W - 290, 110, W - 110, 290);
	medal.addColorStop(0, light);
	medal.addColorStop(1, deep);
	g.fillStyle = medal;
	g.fill();
	g.lineWidth = 10;
	g.strokeStyle = '#ffffff55';
	g.stroke();
	// The words.
	const font = (size: number, weight = 600) =>
		`${weight} ${size}px system-ui, -apple-system, "Segoe UI", sans-serif`;
	g.fillStyle = '#ffffff';
	g.font = font(36, 500);
	g.fillText(spec.title, 80, 130, W - 420);
	g.font = font(112, 700);
	g.fillStyle = light;
	g.fillText(spec.value, 76, 290, W - 420);
	g.font = font(38, 400);
	g.fillStyle = '#e8ecff';
	wrap((s) => g.measureText(s).width, spec.caption, W - 200, 3).forEach((l, i) =>
		g.fillText(l, 80, 380 + i * 52)
	);
	g.font = font(28, 600);
	g.fillStyle = '#ffffffb0';
	g.fillText(footer, 80, H - 70);
	return new Promise((resolve, reject) =>
		canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('canvas'))), 'image/png')
	);
}
