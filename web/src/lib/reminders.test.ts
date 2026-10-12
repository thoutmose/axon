import { describe, expect, it } from 'vitest';
import { deviceName } from './reminders';

describe('device names', () => {
	it('names the browser and the system, nothing more', () => {
		expect(
			deviceName('Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0')
		).toBe('Firefox on Linux');
		expect(
			deviceName(
				'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1'
			)
		).toBe('Safari on iOS');
		expect(
			deviceName(
				'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36 Edg/130.0'
			)
		).toBe('Edge on Windows');
	});
});
