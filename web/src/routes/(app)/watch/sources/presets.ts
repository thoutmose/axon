// How often a source is checked: the cron schedules offered in the lists.
export const presets: [string, string][] = [
	['0 0 * * *', 'Every day at midnight'],
	['0 7 * * *', 'Every morning at 07:00'],
	['0 7,19 * * *', 'Twice a day'],
	['0 */6 * * *', 'Every 6 hours'],
	['0 * * * *', 'Every hour'],
	['0 7 * * 1', 'Every Monday']
];
