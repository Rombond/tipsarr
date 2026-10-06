<script lang="ts">
	import * as Sidebar from '$lib/components/ui/sidebar';
	import { auth } from '$lib/stores/auth.svelte';
	import { counts } from '$lib/stores/counts.svelte';
	import AppSidebar from './app-sidebar.svelte';
	import DryRunBanner from './dry-run-banner.svelte';
	import SearchBox from './search-box.svelte';
	import MobileNav from './mobile-nav.svelte';

	let { children } = $props();

	$effect(() => (auth.isAdmin ? counts.track() : undefined));
</script>

<Sidebar.Provider>
	<AppSidebar />
	<Sidebar.Inset>
		<DryRunBanner />
		<header class="sticky top-0 z-30 flex items-center gap-2 border-b border-border bg-background/85 px-4 py-2 backdrop-blur">
			<SearchBox />
		</header>
		<div class="p-4 pb-24 md:p-6 md:pb-6">
			{@render children()}
		</div>
		<MobileNav pending={auth.isAdmin ? counts.pending : 0} />
	</Sidebar.Inset>
</Sidebar.Provider>
