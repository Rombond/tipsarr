<script lang="ts">
	import { page } from '$app/state';
	import { useSidebar } from '$lib/components/ui/sidebar';
	import CompassIcon from '@lucide/svelte/icons/compass';
	import TicketIcon from '@lucide/svelte/icons/ticket';
	import ListIcon from '@lucide/svelte/icons/list';
	import BookmarkIcon from '@lucide/svelte/icons/bookmark';
	import MenuIcon from '@lucide/svelte/icons/menu';

	let { pending = 0 }: { pending?: number } = $props();
	const sidebar = useSidebar();

	const tabs = [
		{ href: '/discover', label: 'Discover', icon: CompassIcon },
		{ href: '/boxoffice', label: 'Box office', icon: TicketIcon },
		{ href: '/requests', label: 'Requests', icon: ListIcon },
		{ href: '/watchlist', label: 'Watchlist', icon: BookmarkIcon },
	];
</script>

<!-- Phone navigation: the five things people actually use, one thumb away. -->
<nav
	class="fixed inset-x-0 bottom-0 z-40 grid grid-cols-5 border-t border-border bg-background/95 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden"
	aria-label="Main"
>
	{#each tabs as t (t.href)}
		{@const active = page.url.pathname.startsWith(t.href)}
		<a
			href={t.href}
			class="relative flex flex-col items-center gap-0.5 py-2 text-[11px] {active ? 'text-foreground' : 'text-muted-foreground'}"
			aria-current={active ? 'page' : undefined}
		>
			<t.icon class="size-5" />
			{t.label}
			{#if t.href === '/requests' && pending > 0}
				<span class="absolute top-1 left-1/2 ml-2 min-w-4 rounded-full bg-primary px-1 text-center text-[10px] leading-4 text-primary-foreground">{pending}</span>
			{/if}
		</a>
	{/each}
	<button type="button" class="flex cursor-pointer flex-col items-center gap-0.5 py-2 text-[11px] text-muted-foreground" onclick={() => sidebar.setOpenMobile(true)}>
		<MenuIcon class="size-5" />
		Menu
	</button>
</nav>
