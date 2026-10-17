<script lang="ts">
	// Jarvis's picture (EP-153, redrawn at the owner's request): an artificial
	// brain whose neurons are a knowledge graph, in SVG. Signals run along the
	// links in chains and each neuron they reach lights up, in a CSS loop;
	// while Jarvis thinks, more chains fire, faster. Still for people who ask
	// for less motion.
	let { thinking = false }: { thinking?: boolean } = $props();
	// The picture shows twice in the panel; its gradients need their own ids.
	const id = $props.id();

	const outline =
		'M70 170 C60 118 92 68 150 54 C192 40 252 38 292 56 C336 74 362 106 360 150 C358 186 336 206 306 214 C276 222 240 240 196 236 C164 233 146 224 142 208 C130 198 106 206 88 196 C74 188 68 180 70 170 Z';
	const folds = [
		'M228 44 C222 80 206 110 214 140 C220 166 204 186 206 214',
		'M142 208 C170 186 214 174 266 160',
		'M110 92 C130 110 126 140 150 152',
		'M168 60 C176 84 160 104 176 122',
		'M290 64 C276 92 296 116 282 140',
		'M330 100 C312 120 330 146 316 168',
		'M96 140 C110 150 104 170 120 178',
		'M252 190 C276 182 300 194 330 186'
	];
	// [x, y, hub]
	const nodes = [
		[164, 69, 0],
		[225, 111, 0],
		[87, 138, 0],
		[193, 200, 0],
		[319, 96, 0],
		[159, 198, 0],
		[122, 153, 1],
		[255, 112, 1],
		[161, 154, 0],
		[201, 98, 1],
		[300, 176, 1],
		[222, 210, 0],
		[282, 96, 0],
		[272, 155, 0],
		[207, 169, 0],
		[88, 176, 0],
		[151, 121, 0],
		[121, 85, 0],
		[99, 111, 0],
		[248, 69, 0],
		[205, 134, 0],
		[118, 190, 0],
		[251, 215, 0],
		[314, 133, 1],
		[235, 176, 0],
		[215, 75, 0],
		[240, 143, 0],
		[329, 162, 1],
		[283, 198, 0],
		[178, 116, 0],
		[282, 127, 0],
		[339, 115, 0],
		[323, 192, 0],
		[190, 60, 0],
		[124, 124, 1],
		[290, 69, 0],
		[142, 174, 0],
		[178, 176, 0],
		[151, 93, 0]
	];
	// [from, to]
	const links = [
		[0, 33],
		[0, 38],
		[1, 7],
		[1, 9],
		[1, 20],
		[2, 15],
		[2, 18],
		[3, 5],
		[3, 11],
		[3, 37],
		[4, 23],
		[4, 31],
		[4, 35],
		[5, 36],
		[5, 37],
		[6, 15],
		[6, 21],
		[6, 34],
		[6, 36],
		[7, 12],
		[7, 26],
		[7, 30],
		[8, 36],
		[8, 37],
		[9, 25],
		[9, 29],
		[9, 33],
		[10, 13],
		[10, 27],
		[10, 28],
		[10, 32],
		[11, 22],
		[11, 24],
		[12, 19],
		[12, 35],
		[13, 26],
		[13, 30],
		[14, 24],
		[14, 37],
		[15, 21],
		[16, 29],
		[16, 34],
		[16, 38],
		[17, 18],
		[17, 34],
		[17, 38],
		[18, 34],
		[19, 25],
		[19, 35],
		[20, 29],
		[21, 36],
		[22, 28],
		[23, 27],
		[23, 30],
		[23, 31],
		[24, 26],
		[25, 33],
		[27, 31],
		[27, 32]
	];
	// [from, to, start in seconds, only while thinking]
	const signals: [number, number, number, boolean][] = [
		[6, 15, 0.0, false],
		[15, 2, 0.35, false],
		[2, 18, 0.7, false],
		[18, 34, 1.05, false],
		[34, 17, 1.4, false],
		[17, 38, 1.75, false],
		[10, 32, 1.1, false],
		[32, 27, 1.45, false],
		[27, 31, 1.8, false],
		[31, 23, 2.15, false],
		[23, 30, 2.5, false],
		[27, 32, 2.2, false],
		[32, 10, 2.55, false],
		[10, 28, 2.9, false],
		[28, 22, 3.25, false],
		[22, 11, 3.6, false],
		[7, 1, 3.3, true],
		[1, 9, 3.65, true],
		[9, 29, 4.0, true],
		[29, 16, 4.35, true],
		[16, 38, 4.7, true],
		[9, 33, 4.4, true],
		[33, 0, 4.75, true],
		[0, 38, 5.1, true],
		[38, 16, 5.45, true],
		[16, 34, 5.8, true],
		[23, 4, 5.5, true],
		[4, 31, 5.85, true],
		[31, 27, 6.2, true],
		[27, 10, 6.55, true],
		[10, 13, 6.9, true],
		[34, 18, 6.6, true],
		[18, 17, 6.95, true],
		[17, 38, 7.3, true],
		[38, 0, 7.65, true],
		[0, 33, 8.0, true]
	];
	// [neuron, start in seconds, only while thinking]
	const fires: [number, number, boolean][] = [
		[0, 5.05, true],
		[1, 3.6, true],
		[2, 0.65, false],
		[4, 5.8, true],
		[6, 0.0, false],
		[7, 3.3, true],
		[9, 3.95, true],
		[10, 1.1, false],
		[11, 3.9, false],
		[13, 7.2, true],
		[15, 0.3, false],
		[16, 4.65, true],
		[17, 1.7, false],
		[18, 1.0, false],
		[22, 3.55, false],
		[23, 2.45, false],
		[27, 1.75, false],
		[28, 3.2, false],
		[29, 4.3, true],
		[30, 2.8, false],
		[31, 2.1, false],
		[32, 1.4, false],
		[33, 4.7, true],
		[34, 1.35, false],
		[38, 2.05, false]
	];
</script>

<svg class="brain" class:thinking viewBox="50 24 324 226" role="img" aria-label="Jarvis">
	<defs>
		<radialGradient id="{id}-halo" cx="0.5" cy="0.45" r="0.6">
			<stop offset="0" stop-color="#7a3cff" stop-opacity="0.45" />
			<stop offset="0.7" stop-color="#ff3ec8" stop-opacity="0.08" />
			<stop offset="1" stop-color="#ff3ec8" stop-opacity="0" />
		</radialGradient>
		<linearGradient id="{id}-fill" x1="0" y1="0" x2="1" y2="1">
			<stop offset="0" stop-color="#16244a" />
			<stop offset="1" stop-color="#0b0f24" />
		</linearGradient>
		<linearGradient id="{id}-rim" x1="0" y1="0" x2="1" y2="0">
			<stop offset="0" stop-color="#3ef2ff" />
			<stop offset="1" stop-color="#ff3ec8" />
		</linearGradient>
		<filter id="{id}-glow" x="-50%" y="-50%" width="200%" height="200%">
			<feGaussianBlur stdDeviation="2.4" result="b" />
			<feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
		</filter>
	</defs>
	<ellipse cx="214" cy="138" rx="160" ry="112" fill="url(#{id}-halo)" />
	<path d={outline} fill="url(#{id}-fill)" />
	{#each folds as d (d)}
		<path class="fold" {d} />
	{/each}
	<!-- The knowledge graph: its links, the signals on them, then its neurons. -->
	<g class="links">
		{#each links as [a, b] (`${a}-${b}`)}
			<line x1={nodes[a][0]} y1={nodes[a][1]} x2={nodes[b][0]} y2={nodes[b][1]} />
		{/each}
	</g>
	<g filter="url(#{id}-glow)">
		{#each signals as [a, b, at, extra] (`${a}-${b}-${at}`)}
			<line
				class="signal"
				class:extra
				pathLength="1"
				style:--at={at}
				x1={nodes[a][0]}
				y1={nodes[a][1]}
				x2={nodes[b][0]}
				y2={nodes[b][1]}
			/>
		{/each}
	</g>
	{#each nodes as [x, y, hub], i (i)}
		<circle class="neuron" class:hub cx={x} cy={y} r={hub ? 3.6 : 2.1} />
	{/each}
	{#each fires as [n, at, extra] (n)}
		<circle
			class="fire"
			class:extra
			style:--at={at}
			style:transform-origin="{nodes[n][0]}px {nodes[n][1]}px"
			cx={nodes[n][0]}
			cy={nodes[n][1]}
			r={nodes[n][2] ? 6.6 : 5.1}
		/>
	{/each}
	<path class="rim" d={outline} stroke="url(#{id}-rim)" filter="url(#{id}-glow)" />
</svg>

<style>
	.brain {
		--cycle: 7s;
		--pace: 1s;
		width: 100%;
		height: 100%;
		overflow: visible;
	}
	.thinking {
		--cycle: 2.4s;
		--pace: 0.35s;
	}
	.fold {
		fill: none;
		stroke: #3ef2ff;
		stroke-width: 1;
		stroke-linecap: round;
		opacity: 0.18;
	}
	.links {
		stroke: #3ef2ff;
		stroke-width: 0.8;
		opacity: 0.4;
	}
	.neuron {
		fill: #7af8ff;
		opacity: 0.7;
	}
	.hub {
		fill: #ff8ae0;
		opacity: 0.95;
	}
	.rim {
		fill: none;
		stroke-width: 2.2;
	}
	/* A signal runs along its link, then the neuron it reaches fires. */
	.signal {
		fill: none;
		stroke: #fff7c2;
		stroke-width: 1.8;
		stroke-linecap: round;
		stroke-dasharray: 0.3 1.3;
		stroke-dashoffset: 0.3;
		animation: travel var(--cycle) linear calc(var(--at) * var(--pace)) infinite;
	}
	.fire {
		fill: #fff7c2;
		opacity: 0;
		animation: fire var(--cycle) ease-out calc(var(--at) * var(--pace)) infinite;
	}
	.brain:not(.thinking) .extra {
		display: none;
	}
	@keyframes travel {
		0% {
			stroke-dashoffset: 0.3;
		}
		7%,
		100% {
			stroke-dashoffset: -1;
		}
	}
	@keyframes fire {
		0% {
			opacity: 0;
			transform: scale(0.4);
		}
		3% {
			opacity: 0.85;
			transform: scale(1);
		}
		12%,
		100% {
			opacity: 0;
			transform: scale(1.7);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.signal,
		.fire {
			display: none;
		}
	}
</style>
