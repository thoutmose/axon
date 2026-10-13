<script lang="ts">
	import { mix, skyAt } from '$lib/sky';
	import { skyState } from '$lib/skyState.svelte';

	// The sky that follows the sun (EP-96): its colours come from the sun's
	// altitude in the person's time zone, the sun and the moon (with its
	// phase) stand where they are, the stars come out after dusk, and the
	// landscape is the person's choice. Drawn in SVG and shipped with Axon:
	// no image request ever leaves the server. Decorative only.
	let { at, landscape }: { at?: Date; landscape?: string } = $props();

	const sky = $derived(skyAt(at ?? skyState.now, skyState.zone));
	const land = $derived(landscape ?? skyState.landscape);
	const p = $derived(sky.palette);
	// How much daylight there is, 0 at night to 1 in full day.
	const day = $derived(Math.min(1, Math.max(0, (sky.sun.altitude + 6) / 30)));
	const horizon = $derived(land === 'ocean' ? 430 : 410);

	/** Where a body stands in the picture: east on the left, high up at the top. */
	const place = (b: { altitude: number; hourAngle: number }) => ({
		x: Math.min(1380, Math.max(60, 720 + (b.hourAngle / 100) * 640)),
		y: horizon - (Math.min(b.altitude, 70) / 70) * (horizon - 70)
	});
	const sun = $derived(place(sky.sun));
	const moon = $derived(place(sky.moon));
	const sunUp = $derived(sky.sun.altitude > -3);
	// Around the new moon there is nothing to see.
	const moonUp = $derived(sky.moon.altitude > -3 && Math.abs(sky.moon.phase - 0.5) < 0.47);

	/** The lit part of the moon for its phase: waxing lit on the right. */
	function lit(cx: number, cy: number, r: number, phase: number): string {
		const k = Math.cos(2 * Math.PI * phase);
		const waxing = phase < 0.5;
		const termSweep = k > 0 === waxing ? 0 : 1;
		return `M${cx} ${cy - r} A${r} ${r} 0 0 ${waxing ? 1 : 0} ${cx} ${cy + r} A${Math.abs(k) * r} ${r} 0 0 ${termSweep} ${cx} ${cy - r}Z`;
	}

	// Fixed star positions, so the picture never shifts between renders.
	const stars = Array.from({ length: 90 }, (_, i) => ({
		x: (i * 997) % 1440,
		y: (i * 613) % 360,
		r: i % 7 === 0 ? 1.6 : 0.9,
		o: 0.35 + ((i * 37) % 60) / 100
	}));

	/** A row of pines standing on a line, with its ground below. */
	function pines(base: number, h: number, w: number, step: number, seed: number): string {
		let d = '';
		for (let x = -30; x < 1470; x += step) {
			const j = ((x * seed) % 37) / 37;
			const [hh, xx] = [h * (0.7 + 0.6 * j), x + j * step * 0.4];
			d += `M${xx} ${base} L${xx + w / 2} ${base - hh} L${xx + w} ${base}Z M${xx + w * 0.12} ${base - hh * 0.35} L${xx + w / 2} ${base - hh * 1.12} L${xx + w * 0.88} ${base - hh * 0.35}Z `;
		}
		return d + `M0 ${base - 2} H1440 V640 H0Z`;
	}

	const sand = ['#e2b47a', '#d39a5c', '#bf8048', '#a2663a'];
	const tint = (i: number, toward: string[], amount: number) => mix(p.hills[i], toward[i], amount);
	const dunes = $derived([0, 1, 2, 3].map((i) => tint(i, sand, day * 0.8)));
	const sails = $derived(mix(p.hills[0], '#f4f1e8', 0.25 + day * 0.7));
	const sea = $derived([mix(p.sky[3], p.hills[1], 0.35), mix(p.hills[2], p.hills[3], 0.6)]);
	const space = $derived(land === 'earth');
	const id = $props.id();
</script>

<svg
	class="scenery"
	viewBox="0 0 1440 640"
	preserveAspectRatio="xMidYMid slice"
	aria-hidden="true"
	focusable="false"
	data-landscape={land}
	data-light={sky.light}
>
	<defs>
		<linearGradient id="{id}-sky" x1="0" y1="0" x2="0" y2="1">
			{#if space}
				<stop offset="0" stop-color="#01020a" /><stop offset="1" stop-color="#070b1f" />
			{:else}
				{#each p.sky as c, i (i)}<stop offset={i / (p.sky.length - 1)} stop-color={c} />{/each}
			{/if}
		</linearGradient>
		<radialGradient id="{id}-glow" cx="0.5" cy="0.5" r="0.5">
			<stop offset="0" stop-color={sunUp ? p.glow : '#9fb4ff'} stop-opacity={sunUp ? 0.85 : 0.45} />
			<stop offset="1" stop-color={sunUp ? p.glow : '#9fb4ff'} stop-opacity="0" />
		</radialGradient>
		<linearGradient id="{id}-sea" x1="0" y1="0" x2="0" y2="1">
			<stop offset="0" stop-color={sea[0]} /><stop offset="1" stop-color={sea[1]} />
		</linearGradient>
		<radialGradient id="{id}-earth" cx="0.5" cy="0" r="0.6">
			<stop offset="0" stop-color={mix('#0d1a3a', '#3f86c8', day)} />
			<stop offset="1" stop-color={mix('#050a18', '#1b4a82', day)} />
		</radialGradient>
	</defs>

	<rect width="1440" height="640" fill="url(#{id}-sky)" />
	<g opacity={space ? 0.9 : p.stars}>
		{#each stars as s, i (i)}<circle cx={s.x} cy={s.y} r={s.r} fill="#fff" opacity={s.o} />{/each}
	</g>
	{#if sunUp}
		<circle cx={sun.x} cy={sun.y} r="260" fill="url(#{id}-glow)" />
		<circle cx={sun.x} cy={sun.y} r="46" fill={p.sun} />
	{:else if moonUp}
		<circle cx={moon.x} cy={moon.y} r="200" fill="url(#{id}-glow)" />
	{/if}
	{#if moonUp}
		<g opacity={sunUp ? 0.5 : 1}>
			{#if !sunUp}<circle cx={moon.x} cy={moon.y} r="30" fill="#0b1230" opacity="0.35" />{/if}
			<path d={lit(moon.x, moon.y, 30, sky.moon.phase)} fill="#f3f0e4" />
		</g>
	{/if}

	{#if land === 'forest'}
		<path
			fill={p.hills[0]}
			d="M0 430 C200 380 380 400 560 420 C760 440 940 370 1140 395 C1280 410 1360 400 1440 410 V640 H0Z"
		/>
		<path fill={p.hills[1]} d={pines(480, 70, 34, 26, 7)} />
		<path fill={p.hills[2]} d={pines(555, 110, 52, 42, 11)} />
		<path fill={p.hills[3]} d={pines(640, 170, 84, 74, 13)} />
	{:else if land === 'ocean'}
		<path fill={p.hills[1]} d="M0 430 L0 372 C90 360 170 380 250 404 C300 418 340 426 380 430Z" />
		<rect y="430" width="1440" height="210" fill="url(#{id}-sea)" />
		{#each Array.from({ length: 9 }, (_, i) => i) as i (i)}
			{@const at = sunUp ? sun : moon}
			{#if sunUp || moonUp}
				<ellipse
					cx={at.x}
					cy={440 + i * i * 2.4}
					rx={70 - i * 5}
					ry="2.4"
					fill={sunUp ? p.glow : '#e8ecff'}
					opacity={0.75 - i * 0.07}
				/>
			{/if}
		{/each}
		{#each [470, 520, 580] as y, i (y)}
			<path
				d="M0 {y} Q180 {y - 6} 360 {y} T720 {y} T1080 {y} T1440 {y}"
				fill="none"
				stroke={p.hills[0]}
				stroke-opacity={0.35 - i * 0.08}
				stroke-width="1.5"
			/>
		{/each}
	{:else if land === 'desert'}
		<path
			fill={dunes[0]}
			d="M0 440 C180 405 320 420 470 440 C640 462 760 410 960 420 C1140 430 1300 405 1440 420 V640 H0Z"
		/>
		<path
			fill={dunes[1]}
			d="M0 490 C160 455 330 470 520 500 C700 528 860 455 1060 470 C1220 482 1340 462 1440 470 V640 H0Z"
		/>
		<path
			fill={dunes[2]}
			d="M0 560 C220 510 420 520 620 556 C820 590 1000 520 1200 530 C1320 536 1390 548 1440 552 V640 H0Z"
		/>
		<path fill={dunes[3]} d="M0 610 C260 580 480 590 720 612 C960 634 1200 590 1440 600 V640 H0Z" />
	{:else if space}
		<circle
			cx="720"
			cy="1980"
			r="1600"
			fill="none"
			stroke={mix('#2a4d9a', '#8fd0ff', day)}
			stroke-width="26"
			stroke-opacity="0.35"
		/>
		<circle cx="720" cy="1980" r="1590" fill="url(#{id}-earth)" />
		<g fill="#ffd27a" opacity={(1 - day) * 0.9}>
			{#each stars.slice(0, 40) as s, i (i)}<circle
					cx={(s.x * 7) % 1440}
					cy={420 + ((s.y * 3) % 200)}
					r={s.r * 0.9}
				/>{/each}
		</g>
	{:else if land === 'eiffel-tower'}
		<path
			fill={p.hills[1]}
			d="M0 500 C240 470 480 488 720 500 C960 512 1200 478 1440 492 V640 H0Z"
		/>
		<path
			fill={p.hills[3]}
			d="M935 600 L960 600 Q975 520 985 470 L1015 470 Q1025 520 1040 600 L1065 600 Q1030 520 1018 440 L1012 340 L1006 250 L1003 190 L1001 140 L999 140 L997 190 L994 250 L988 340 L982 440 Q970 520 935 600Z M972 464 H1028 V472 H972Z M984 334 H1016 V340 H984Z M993 244 H1007 V249 H993Z"
		/>
		<path
			fill={p.hills[2]}
			d="M0 590 C300 575 600 585 900 598 C1100 606 1300 592 1440 596 V640 H0Z"
		/>
	{:else if land === 'pyramids'}
		<path
			fill={dunes[0]}
			d="M0 520 C300 500 600 512 900 520 C1150 526 1300 514 1440 518 V640 H0Z"
		/>
		<path fill={dunes[2]} d="M780 600 L960 400 L1000 600Z" /><path
			fill={dunes[3]}
			d="M960 400 L1150 600 L1000 600Z"
		/>
		<path fill={dunes[2]} d="M1110 600 L1235 462 L1262 600Z" /><path
			fill={dunes[3]}
			d="M1235 462 L1365 600 L1262 600Z"
		/>
		<path fill={dunes[2]} d="M1330 600 L1392 532 L1405 600Z" /><path
			fill={dunes[3]}
			d="M1392 532 L1454 600 L1405 600Z"
		/>
		<path
			fill={dunes[1]}
			d="M0 595 C360 580 720 590 1080 600 C1260 605 1380 598 1440 600 V640 H0Z"
		/>
	{:else if land === 'taj-mahal'}
		<path
			fill={p.hills[1]}
			d="M0 520 C360 505 720 515 1080 520 C1260 522 1380 515 1440 518 V640 H0Z"
		/>
		<g fill={sails}>
			<path d="M870 572 H1250 V600 H870Z M955 472 H1165 V572 H955Z" />
			<path
				d="M1060 330 L1062 352 C1102 368 1124 410 1112 462 H1008 C996 410 1018 368 1058 352Z M1015 456 H1105 V474 H1015Z"
			/>
			<path
				d="M968 472 C962 448 972 436 985 432 C998 436 1008 448 1002 472Z M1118 472 C1112 448 1122 436 1135 432 C1148 436 1158 448 1152 472Z"
			/>
			<path
				d="M884 572 V420 H898 V572Z M1222 572 V420 H1236 V572Z M882 420 L891 400 L900 420Z M1220 420 L1229 400 L1238 420Z"
			/>
		</g>
		<path fill={p.hills[3]} d="M1036 572 V520 C1036 498 1084 498 1084 520 V572Z" />
		<path fill={p.hills[2]} d="M0 600 H1440 V640 H0Z" />
		<rect x="1030" y="606" width="60" height="34" fill={p.sky[2]} opacity="0.6" />
	{:else if land === 'statue-of-liberty'}
		<rect y="560" width="1440" height="80" fill="url(#{id}-sea)" />
		<path
			fill={p.hills[3]}
			d="M930 600 L955 572 L1125 572 L1150 600Z M1006 572 V512 L996 506 V494 H1084 V506 L1074 512 V572Z"
		/>
		<path
			fill={p.hills[3]}
			d="M1018 494 L1062 494 C1060 440 1054 400 1050 366 L1030 366 C1026 400 1020 440 1018 494Z M1026 420 L1012 432 L1016 440 L1030 430Z"
		/>
		<circle cx="1040" cy="354" r="11" fill={p.hills[3]} />
		<path
			fill={p.hills[3]}
			d="M1046 364 L1064 300 L1071 302 L1054 368Z M1058 300 H1076 L1072 288 H1062Z M1028 346 L1022 336 M1034 342 L1032 330 M1046 342 L1048 330 M1052 346 L1058 336"
			stroke={p.hills[3]}
			stroke-width="3"
		/>
		<path fill={p.glow} d="M1067 288 C1060 278 1068 270 1067 262 C1074 272 1076 280 1067 288Z" />
	{:else if land === 'christ-the-redeemer'}
		<path fill={p.hills[1]} d="M520 640 C560 560 610 520 660 520 C710 524 740 580 770 640Z" />
		<path
			fill={p.hills[2]}
			d="M760 640 L870 480 C910 428 950 404 1000 398 C1050 404 1090 434 1130 488 L1280 640Z"
		/>
		<path
			fill={p.hills[3]}
			d="M990 398 H1010 V382 H990Z M994 382 L1006 382 L1005 330 L995 330Z M956 330 H1044 V338 H956Z"
		/>
		<circle cx="1000" cy="322" r="7" fill={p.hills[3]} />
	{:else if land === 'opera-house'}
		<rect y="585" width="1440" height="55" fill="url(#{id}-sea)" />
		<path d="M1240 585 Q1420 395 1600 585" fill="none" stroke={p.hills[2]} stroke-width="10" />
		<path fill={p.hills[2]} d="M1200 548 H1640 V556 H1200Z M1218 520 H1242 V585 H1218Z" />
		<path fill={p.hills[3]} d="M800 560 H1200 V585 H800Z" />
		<g fill={sails}>
			<path d="M850 560 C860 500 900 470 940 455 C925 490 925 530 935 560Z" />
			<path d="M910 560 C925 490 970 450 1020 440 C1000 480 998 525 1010 560Z" />
			<path d="M990 560 C1005 500 1040 470 1080 460 C1065 495 1062 530 1070 560Z" />
			<path d="M1060 560 C1070 515 1100 495 1135 488 C1122 515 1120 540 1126 560Z" />
		</g>
	{:else}
		<path
			fill={p.hills[0]}
			d="M0 400 L120 330 L230 372 L360 290 L470 350 L600 270 L720 340 L860 260 L990 330 L1110 280 L1240 350 L1340 300 L1440 340 V640 H0Z"
		/>
		<path
			fill={p.hills[1]}
			d="M0 450 L150 380 L280 430 L420 360 L560 425 L700 365 L830 430 L980 370 L1120 440 L1260 385 L1440 430 V640 H0Z"
		/>
		<path
			fill={p.hills[2]}
			d="M0 510 C180 450 300 470 440 500 C600 535 720 455 900 480 C1060 502 1200 470 1440 505 V640 H0Z"
		/>
		<path
			fill={p.hills[3]}
			d="M0 575 C220 540 380 560 560 580 C760 602 940 545 1140 565 C1280 579 1360 570 1440 575 V640 H0Z"
		/>
	{/if}
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
