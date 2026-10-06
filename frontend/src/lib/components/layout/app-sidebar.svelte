<script lang="ts">
	import { t, type Key } from '$lib/i18n/index.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';
	import { counts } from '$lib/stores/counts.svelte';
	import { theme } from '$lib/theme.svelte';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import UserAvatar from '$lib/components/users/user-avatar.svelte';
	import FlagIcon from '@lucide/svelte/icons/flag';
	import CompassIcon from '@lucide/svelte/icons/compass';
	import ListIcon from '@lucide/svelte/icons/list';
	import UsersIcon from '@lucide/svelte/icons/users';
	import SettingsIcon from '@lucide/svelte/icons/settings';
	import LogOutIcon from '@lucide/svelte/icons/log-out';
	import BookmarkIcon from '@lucide/svelte/icons/bookmark';
	import UserIcon from '@lucide/svelte/icons/user';
	import TicketIcon from '@lucide/svelte/icons/ticket';
	import ServerIcon from '@lucide/svelte/icons/server';
	import BellIcon from '@lucide/svelte/icons/bell';
	import SunIcon from '@lucide/svelte/icons/sun';
	import MoonIcon from '@lucide/svelte/icons/moon';
	import MonitorIcon from '@lucide/svelte/icons/monitor';

	type NavItem = { href: string; key: Key; icon: typeof CompassIcon };
	const browse: NavItem[] = [
		{ href: '/discover', key: 'nav.discover', icon: CompassIcon },
		{ href: '/boxoffice', key: 'nav.boxoffice', icon: TicketIcon },
		{ href: '/requests', key: 'nav.requests', icon: ListIcon },
		{ href: '/watchlist', key: 'nav.watchlist', icon: BookmarkIcon },
		{ href: '/issues', key: 'nav.issues', icon: FlagIcon },
	];
	const admin: NavItem[] = [
		{ href: '/admin/users', key: 'nav.users', icon: UsersIcon },
		{ href: '/admin/services', key: 'nav.services', icon: ServerIcon },
		{ href: '/admin/webhooks', key: 'nav.webhooks', icon: BellIcon },
		{ href: '/admin/settings', key: 'nav.settings', icon: SettingsIcon },
	];

	const sidebar = Sidebar.useSidebar();

	function navigate(href: string) {
		sidebar.setOpenMobile(false);
		goto(href);
	}

	async function handleLogout() {
		await auth.logout();
		goto('/login');
	}
</script>

{#snippet item(it: NavItem)}
	<Sidebar.MenuItem>
		<Sidebar.MenuButton isActive={page.url.pathname.startsWith(it.href)} class="h-11 gap-3 px-3 text-[15px] [&_svg]:size-5">
			{#snippet child({ props })}
				<a href={it.href} {...props} onclick={() => sidebar.setOpenMobile(false)}>
					<it.icon />
					<span>{t(it.key)}</span>
				</a>
			{/snippet}
		</Sidebar.MenuButton>
		{#if it.href === '/requests' && auth.isAdmin && counts.pending > 0}
			<Sidebar.MenuBadge>{counts.pending}</Sidebar.MenuBadge>
		{:else if it.href === '/issues' && auth.isAdmin && counts.issues > 0}
			<Sidebar.MenuBadge>{counts.issues}</Sidebar.MenuBadge>
		{/if}
	</Sidebar.MenuItem>
{/snippet}

<Sidebar.Root>
	<Sidebar.Header>
		<a href="/discover" class="flex items-center gap-3 px-3 py-3 font-bold text-xl" onclick={() => sidebar.setOpenMobile(false)}>
			<span class="inline-flex size-9 items-center justify-center rounded-lg bg-primary text-primary-foreground">T</span>
			Tipsarr
		</a>
	</Sidebar.Header>
	<Sidebar.Content>
		<Sidebar.Group class="px-3 py-3">
			<Sidebar.GroupLabel class="px-3 text-xs font-semibold uppercase tracking-wider">{t('nav.browse')}</Sidebar.GroupLabel>
			<Sidebar.Menu class="gap-1">
				{#each browse as it (it.href)}{@render item(it)}{/each}
			</Sidebar.Menu>
		</Sidebar.Group>
		{#if auth.isAdmin}
			<Sidebar.Group class="px-3 py-3">
				<Sidebar.GroupLabel class="px-3 text-xs font-semibold uppercase tracking-wider">{t('nav.admin')}</Sidebar.GroupLabel>
				<Sidebar.Menu class="gap-1">
					{#each admin as it (it.href)}{@render item(it)}{/each}
				</Sidebar.Menu>
			</Sidebar.Group>
		{/if}
	</Sidebar.Content>
	<Sidebar.Footer>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger class="flex w-full cursor-pointer items-center gap-3 rounded-lg px-3 py-2.5 text-sm hover:bg-sidebar-accent">
				{#if auth.user}<UserAvatar id={auth.user.id} name={auth.user.name} class="size-9" />{/if}
				<span class="grid min-w-0 text-left leading-tight">
					<span class="truncate">{auth.username || t('nav.account')}</span>
					<span class="text-[11px] text-muted-foreground">{auth.isAdmin ? t('nav.administrator') : t('nav.member')}</span>
				</span>
			</DropdownMenu.Trigger>
			<DropdownMenu.Content side="top" align="start" class="w-56">
				<DropdownMenu.Item onclick={() => navigate('/profile')}>
					<UserIcon />
					{t('nav.profile')}
				</DropdownMenu.Item>
				<DropdownMenu.Separator />
				<DropdownMenu.Label class="text-xs text-muted-foreground">{t('theme.title')}</DropdownMenu.Label>
				<DropdownMenu.Item onclick={() => theme.set('light')}>
					<SunIcon />
					{t('theme.light')} {theme.value === 'light' ? '✓' : ''}
				</DropdownMenu.Item>
				<DropdownMenu.Item onclick={() => theme.set('dark')}>
					<MoonIcon />
					{t('theme.dark')} {theme.value === 'dark' ? '✓' : ''}
				</DropdownMenu.Item>
				<DropdownMenu.Item onclick={() => theme.set('system')}>
					<MonitorIcon />
					{t('theme.system')} {theme.value === 'system' ? '✓' : ''}
				</DropdownMenu.Item>
				<DropdownMenu.Separator />
				<DropdownMenu.Item onclick={handleLogout}>
					<LogOutIcon />
					{t('nav.logout')}
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</Sidebar.Footer>
</Sidebar.Root>
