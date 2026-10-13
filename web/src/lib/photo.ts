/**
 * Photos from the camera (EP-89). A phone photo is made small before it is
 * sent (a vision model reads at most about 1,568 px on the long side), and a
 * barcode is read in the browser where it can, so no photo leaves the device
 * just to find a product.
 */

/** The size that fits within max on the long side, never larger than the original. */
export function fitSize(width: number, height: number, max: number) {
	const scale = Math.min(1, max / Math.max(width, height));
	return { width: Math.round(width * scale), height: Math.round(height * scale) };
}

/** The photo as a JPEG no larger than max on its long side. */
export async function shrink(file: Blob, max = 1568): Promise<Blob> {
	const bitmap = await createImageBitmap(file);
	const { width, height } = fitSize(bitmap.width, bitmap.height, max);
	const canvas = document.createElement('canvas');
	canvas.width = width;
	canvas.height = height;
	canvas.getContext('2d')!.drawImage(bitmap, 0, 0, width, height);
	bitmap.close();
	return new Promise((resolve, reject) =>
		canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('encode'))), 'image/jpeg', 0.85)
	);
}

type Detector = { detect(source: ImageBitmap): Promise<{ rawValue: string }[]> };
type DetectorClass = new (options: { formats: string[] }) => Detector;

/** Whether this browser can read barcodes from a photo (Chrome, Edge, Android). */
export const canReadBarcodes = () => typeof window !== 'undefined' && 'BarcodeDetector' in window;

/** The product barcode in a photo, or null. */
export async function readBarcode(file: Blob): Promise<string | null> {
	const Barcode = (window as unknown as { BarcodeDetector: DetectorClass }).BarcodeDetector;
	const bitmap = await createImageBitmap(file);
	try {
		const found = await new Barcode({ formats: ['ean_13', 'ean_8', 'upc_a', 'upc_e'] }).detect(
			bitmap
		);
		return found.find((b) => /^\d{8,14}$/.test(b.rawValue))?.rawValue ?? null;
	} finally {
		bitmap.close();
	}
}
