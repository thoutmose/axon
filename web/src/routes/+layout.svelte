<script lang="ts">
	import '../app.css';

	let { children } = $props();
</script>

<a class="skip" href="#main">Skip to content</a>
<header>
	<a href="/" class="brand">Axon</a>
	<nav aria-label="Main">
		<a href="/account">Account</a>
	</nav>
</header>
<main id="main">
	{@render children()}
</main>
