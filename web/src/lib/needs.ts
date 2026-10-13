/** How far a day's intake is towards a need (EP-88): the bar's share, and
 * whether it went over. Over is a fact, not a fault: shown in words only. */
export function progress(eaten: number, need: number): { share: number; over: boolean } {
	if (need <= 0) return { share: 0, over: false };
	return { share: Math.min(1, eaten / need), over: eaten > need };
}
