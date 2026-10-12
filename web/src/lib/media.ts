/** The hue (0–359) of a generated cover: the same title always gets the same colour. */
export function coverHue(title: string): number {
	let h = 0;
	for (const ch of title) h = (h * 31 + ch.codePointAt(0)!) % 360;
	return h;
}
