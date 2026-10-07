<script lang="ts">
	import { page } from '$app/state';
	import { useSidebar } from '$lib/components/ui/sidebar';
	import { t, type Key } from '$lib/i18n/index.svelte';
	import CompassIcon from '@lucide/svelte/icons/compass';
	import TicketIcon from '@lucide/svelte/icons/ticket';
	import ListIcon from '@lucide/svelte/icons/list';
	import BookmarkIcon from '@lucide/svelte/icons/bookmark';
	import MenuIcon from '@lucide/svelte/icons/menu';

	let { pending = 0 }: { pending?: number } = $props();
	const sidebar = useSidebar();

	const tabs: { href: string; key: Key; icon: typeof CompassIcon }[] = [
		{ href: '/discover', key: 'nav.discover', icon: CompassIcon },
		{ href: '/boxoffice', key: 'nav.boxoffice', icon: TicketIcon },
		{ href: '/requests', key: 'nav.requests', icon: ListIcon },
		{ href: '/watchlist', key: 'nav.watchlist', icon: BookmarkIcon },
	];
</script>

<!-- Phone navigation: the five things people actually use, one thumb away. -->
<nav
	class="fixed inset-x-0 bottom-0 z-40 grid grid-cols-5 border-t border-border bg-background/95 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden"
	aria-label={t('nav.main')}
>
	{#each tabs as tab (tab.href)}
		{@const active = page.url.pathname.startsWith(tab.href)}
		<a
			href={tab.href}
			class="relative flex flex-col items-center gap-0.5 py-2 text-[11px] {active ? 'text-foreground' : 'text-muted-foreground'}"
			aria-current={active ? 'page' : undefined}
		>
			<tab.icon class="size-5" />
			{t(tab.key)}
			{#if tab.href === '/requests' && pending > 0}
				<span class="absolute top-1 left-1/2 ml-2 min-w-5 rounded-full bg-primary px-1.5 text-center text-xs leading-5 font-semibold text-primary-foreground">{pending}</span>
			{/if}
		</a>
	{/each}
	<button type="button" class="flex cursor-pointer flex-col items-center gap-0.5 py-2 text-[11px] text-muted-foreground" onclick={() => sidebar.setOpenMobile(true)}>
		<MenuIcon class="size-5" />
		{t('nav.menu')}
	</button>
</nav>
