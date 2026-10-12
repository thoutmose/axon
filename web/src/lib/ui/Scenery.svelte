<script lang="ts" module>
	export type TimeOfDay = 'dawn' | 'day' | 'dusk' | 'night';

	/** The palette for the person's local hour. */
	export function timeOfDay(hour = new Date().getHours()): TimeOfDay {
		if (hour >= 5 && hour < 10) return 'dawn';
		if (hour >= 10 && hour < 17) return 'day';
		if (hour >= 17 && hour < 21) return 'dusk';
		return 'night';
	}
</script>

<script lang="ts">
	// A layered mountain landscape drawn in SVG and shipped with Axon: no image
	// request ever leaves the server. Decorative only.
	let { time = timeOfDay() }: { time?: TimeOfDay } = $props();

	const palettes: Record<
		TimeOfDay,
		{ sky: string[]; sun: string; glow: string; hills: string[]; stars: boolean }
	> = {
		dawn: {
			sky: ['#2b3a7a', '#8a6fb8', '#f4a6a6', '#ffd8a8'],
			sun: '#fff1d6',
			glow: '#ffc58a',
			hills: ['#9b8ac4', '#7466a8', '#4f4a86', '#2f2d5c'],
			stars: false
		},
		day: {
			sky: ['#3d7bd9', '#6aa6ee', '#a9d0f7', '#e6f2fd'],
			sun: '#fffdf2',
			glow: '#fff3c4',
			hills: ['#8fb7d9', '#5f92c0', '#3c6e9e', '#244d75'],
			stars: false
		},
		dusk: {
			sky: ['#1d1a4a', '#5b2d74', '#c4527a', '#ff9a62'],
			sun: '#ffd1a1',
			glow: '#ff8a5c',
			hills: ['#7a4a86', '#56356f', '#3a2453', '#211638'],
			stars: false
		},
		night: {
			sky: ['#03050f', '#0a1230', '#152150', '#24356e'],
			sun: '#e9eeff',
			glow: '#7f9cff',
			hills: ['#253467', '#1a2650', '#111a3a', '#090f25'],
			stars: true
		}
	};
	const p = $derived(palettes[time]);
	// Fixed star positions, so the picture never shifts between renders.
	const stars = Array.from({ length: 70 }, (_, i) => ({
		x: (i * 997) % 1440,
		y: (i * 613) % 330,
		r: i % 7 === 0 ? 1.6 : 0.9,
		o: 0.35 + ((i * 37) % 60) / 100
	}));
</script>

<svg
	class="scenery"
	viewBox="0 0 1440 640"
	preserveAspectRatio="xMidYMid slice"
	aria-hidden="true"
	focusable="false"
>
	<defs>
		<linearGradient id="sky-{time}" x1="0" y1="0" x2="0" y2="1">
			{#each p.sky as c, i (i)}<stop offset={i / (p.sky.length - 1)} stop-color={c} />{/each}
		</linearGradient>
		<radialGradient id="glow-{time}" cx="0.5" cy="0.5" r="0.5">
			<stop offset="0" stop-color={p.glow} stop-opacity="0.85" />
			<stop offset="1" stop-color={p.glow} stop-opacity="0" />
		</radialGradient>
		{#each p.hills as c, i (i)}
			<linearGradient id="hill-{time}-{i}" x1="0" y1="0" x2="0" y2="1">
				<stop offset="0" stop-color={c} />
				<stop offset="1" stop-color={p.hills[Math.min(i + 1, p.hills.length - 1)]} />
			</linearGradient>
		{/each}
	</defs>
	<rect width="1440" height="640" fill="url(#sky-{time})" />
	{#if p.stars}
		{#each stars as s, i (i)}<circle cx={s.x} cy={s.y} r={s.r} fill="#fff" opacity={s.o} />{/each}
	{/if}
	<circle cx="1040" cy="300" r="260" fill="url(#glow-{time})" />
	<circle cx="1040" cy="300" r="54" fill={p.sun} />
	<path
		fill="url(#hill-{time}-0)"
		d="M0 400 L120 330 L230 372 L360 290 L470 350 L600 270 L720 340 L860 260 L990 330 L1110 280 L1240 350 L1340 300 L1440 340 V640 H0Z"
	/>
	<path
		fill="url(#hill-{time}-1)"
		d="M0 450 L150 380 L280 430 L420 360 L560 425 L700 365 L830 430 L980 370 L1120 440 L1260 385 L1440 430 V640 H0Z"
	/>
	<path
		fill="url(#hill-{time}-2)"
		d="M0 510 C180 450 300 470 440 500 C600 535 720 455 900 480 C1060 502 1200 470 1440 505 V640 H0Z"
	/>
	<path
		fill="url(#hill-{time}-3)"
		d="M0 575 C220 540 380 560 560 580 C760 602 940 545 1140 565 C1280 579 1360 570 1440 575 V640 H0Z"
	/>
</svg>

<style>
	.scenery {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		display: block;
	}
</style>
