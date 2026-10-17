<script lang="ts">
	// Jarvis's face (EP-153): an original, abstract face of light lines, not
	// anyone's likeness, drawn in SVG and moved by CSS as a loop: a scan line
	// passes, the eyes glow, and the mouth speaks while Jarvis thinks. Still
	// for people who ask for less motion.
	let { thinking = false }: { thinking?: boolean } = $props();
	const bars = Array.from({ length: 11 }, (_, i) => i);
</script>

<svg class="face" class:thinking viewBox="0 0 200 220" role="img" aria-label="Jarvis">
	<defs>
		<linearGradient id="jv-line" x1="0" y1="0" x2="1" y2="1">
			<stop offset="0" stop-color="#3ef2ff" />
			<stop offset="1" stop-color="#ff3ec8" />
		</linearGradient>
		<radialGradient id="jv-eye">
			<stop offset="0" stop-color="#ffffff" />
			<stop offset="0.35" stop-color="#3ef2ff" />
			<stop offset="1" stop-color="#3ef2ff" stop-opacity="0" />
		</radialGradient>
	</defs>
	<!-- The head, its jaw, and the circuit lines on the temples. -->
	<path
		class="line"
		d="M100 14 C150 14 176 52 176 98 C176 140 160 170 132 192 C118 203 108 208 100 208 C92 208 82 203 68 192 C40 170 24 140 24 98 C24 52 50 14 100 14 Z"
	/>
	<path class="line thin" d="M40 70 H58 L66 78 H80 M160 70 H142 L134 78 H120" />
	<path class="line thin" d="M34 118 H52 L60 110 M166 118 H148 L140 110" />
	<path class="line thin" d="M100 14 V40 M86 22 L100 40 L114 22" />
	<!-- Brows, eyes, and the bridge of the nose. -->
	<path class="line" d="M58 88 Q76 80 92 88 M108 88 Q124 80 142 88" />
	<ellipse class="eye-glow" cx="75" cy="102" rx="16" ry="9" fill="url(#jv-eye)" />
	<ellipse class="eye-glow" cx="125" cy="102" rx="16" ry="9" fill="url(#jv-eye)" />
	<ellipse class="line" cx="75" cy="102" rx="12" ry="5" />
	<ellipse class="line" cx="125" cy="102" rx="12" ry="5" />
	<path class="line thin" d="M100 104 V138 L92 146 H108" />
	<!-- The mouth: a waveform that speaks while Jarvis thinks. -->
	<g class="mouth" transform="translate(70 166)">
		{#each bars as i (i)}
			<rect
				class="bar"
				style:animation-delay="{(i % 5) * 0.11 + Math.floor(i / 5) * 0.05}s"
				x={i * 5.6}
				y="-6"
				width="2.6"
				height="12"
				rx="1.3"
			/>
		{/each}
	</g>
	<rect class="scan" x="20" y="0" width="160" height="3" />
</svg>

<style>
	.face {
		width: 100%;
		height: 100%;
		overflow: visible;
		filter: drop-shadow(0 0 6px rgb(62 242 255 / 0.55));
	}
	.line {
		fill: none;
		stroke: url(#jv-line);
		stroke-width: 2;
		stroke-linecap: round;
		stroke-linejoin: round;
	}
	.thin {
		stroke-width: 1.2;
		opacity: 0.7;
	}
	.eye-glow {
		animation: glow 3.2s ease-in-out infinite;
	}
	.bar {
		fill: #3ef2ff;
		transform-box: fill-box;
		transform-origin: center;
		transform: scaleY(0.25);
	}
	.thinking .bar {
		animation: speak 0.9s ease-in-out infinite;
	}
	.scan {
		fill: #3ef2ff;
		opacity: 0.35;
		animation: scan 4s linear infinite;
	}
	@keyframes glow {
		0%,
		100% {
			opacity: 0.55;
		}
		50% {
			opacity: 1;
		}
	}
	@keyframes speak {
		0%,
		100% {
			transform: scaleY(0.25);
		}
		50% {
			transform: scaleY(1);
		}
	}
	@keyframes scan {
		from {
			transform: translateY(10px);
		}
		to {
			transform: translateY(210px);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.eye-glow,
		.thinking .bar,
		.scan {
			animation: none;
		}
		.scan {
			display: none;
		}
	}
</style>
