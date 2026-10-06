<script lang="ts">
	import * as Sidebar from '$lib/components/ui/sidebar';
	import { auth } from '$lib/stores/auth.svelte';
	import { counts } from '$lib/stores/counts.svelte';
	import AppSidebar from './app-sidebar.svelte';
	import DryRunBanner from './dry-run-banner.svelte';
	import SearchBox from './search-box.svelte';
	import MobileNav from './mobile-nav.svelte';

	let { children } = $props();

	let scrollY = $state(0);
	const solid = $derived(scrollY > 24);

	$effect(() => (auth.isAdmin ? counts.track() : undefined));
</script>

<svelte:window bind:scrollY />

<Sidebar.Provider style="--sidebar-width: 17rem">
	<AppSidebar />
	<Sidebar.Inset>
		<DryRunBanner />
		<!-- Full-width search bar that floats over the page: on detail pages the backdrop shows
		     behind it, elsewhere it turns solid once you scroll. -->
		<header
			class="sticky top-0 z-30 -mb-16 flex h-16 items-center px-4 transition-colors md:px-8 {solid
				? 'border-b border-border bg-background/85 backdrop-blur-md'
				: 'bg-gradient-to-b from-background/70 to-transparent'}"
		>
			<SearchBox />
		</header>
		<div class="animate-in fade-in px-4 pt-20 pb-24 duration-300 md:px-8 md:pb-10">
			{@render children()}
		</div>
		<MobileNav pending={auth.isAdmin ? counts.pending : 0} />
	</Sidebar.Inset>
</Sidebar.Provider>
