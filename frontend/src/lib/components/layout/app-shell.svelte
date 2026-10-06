<script lang="ts">
	import { goto } from '$app/navigation';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import { Input } from '$lib/components/ui/input';
	import AppSidebar from './app-sidebar.svelte';
	import SearchIcon from '@lucide/svelte/icons/search';

	let { children } = $props();

	let query = $state('');

	function handleSearchKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && query.trim()) {
			goto(`/search?q=${encodeURIComponent(query.trim())}`);
		}
	}
</script>

<Sidebar.Provider>
	<AppSidebar />
	<Sidebar.Inset>
		<div class="flex items-center gap-2 border-b border-border px-4 py-2">
			<Sidebar.Trigger class="md:hidden" />
			<div class="relative w-full max-w-md">
				<SearchIcon class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
				<Input
					class="pl-8"
					placeholder="Search movies, TV shows, people…"
					bind:value={query}
					onkeydown={handleSearchKeydown}
				/>
			</div>
		</div>
		<div class="p-4 md:p-6">
			{@render children()}
		</div>
	</Sidebar.Inset>
</Sidebar.Provider>
