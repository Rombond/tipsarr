<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';
	import { counts } from '$lib/stores/counts.svelte';
	import { theme } from '$lib/theme.svelte';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Avatar, AvatarFallback } from '$lib/components/ui/avatar';
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

	const browse = [
		{ href: '/discover', label: 'Discover', icon: CompassIcon },
		{ href: '/boxoffice', label: 'Box office', icon: TicketIcon },
		{ href: '/requests', label: 'Requests', icon: ListIcon },
		{ href: '/watchlist', label: 'Watchlist', icon: BookmarkIcon },
	];
	const admin = [
		{ href: '/admin/users', label: 'Users', icon: UsersIcon },
		{ href: '/admin/services', label: 'Radarr / Sonarr', icon: ServerIcon },
		{ href: '/admin/webhooks', label: 'Webhooks', icon: BellIcon },
		{ href: '/admin/settings', label: 'Settings', icon: SettingsIcon },
	];

	const sidebar = Sidebar.useSidebar();

	function initials(name: string | null) {
		return name ? name.slice(0, 2).toUpperCase() : '?';
	}

	function navigate(href: string) {
		sidebar.setOpenMobile(false);
		goto(href);
	}

	async function handleLogout() {
		await auth.logout();
		goto('/login');
	}
</script>

{#snippet item(it: { href: string; label: string; icon: typeof CompassIcon })}
	<Sidebar.MenuItem>
		<Sidebar.MenuButton isActive={page.url.pathname.startsWith(it.href)}>
			{#snippet child({ props })}
				<a href={it.href} {...props} onclick={() => sidebar.setOpenMobile(false)}>
					<it.icon />
					<span>{it.label}</span>
				</a>
			{/snippet}
		</Sidebar.MenuButton>
		{#if it.href === '/requests' && auth.isAdmin && counts.pending > 0}
			<Sidebar.MenuBadge>{counts.pending}</Sidebar.MenuBadge>
		{/if}
	</Sidebar.MenuItem>
{/snippet}

<Sidebar.Root>
	<Sidebar.Header>
		<a href="/discover" class="flex items-center gap-2 px-2 py-1.5 font-bold text-lg" onclick={() => sidebar.setOpenMobile(false)}>
			<span class="inline-flex size-7 items-center justify-center rounded-md bg-primary text-primary-foreground text-sm">T</span>
			Tipsarr
		</a>
	</Sidebar.Header>
	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.GroupLabel>Browse</Sidebar.GroupLabel>
			<Sidebar.Menu>
				{#each browse as it (it.href)}{@render item(it)}{/each}
			</Sidebar.Menu>
		</Sidebar.Group>
		{#if auth.isAdmin}
			<Sidebar.Group>
				<Sidebar.GroupLabel>Admin</Sidebar.GroupLabel>
				<Sidebar.Menu>
					{#each admin as it (it.href)}{@render item(it)}{/each}
				</Sidebar.Menu>
			</Sidebar.Group>
		{/if}
	</Sidebar.Content>
	<Sidebar.Footer>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger class="flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-sidebar-accent">
				<Avatar class="size-7">
					<AvatarFallback>{initials(auth.username)}</AvatarFallback>
				</Avatar>
				<span class="grid min-w-0 text-left leading-tight">
					<span class="truncate">{auth.username || 'Account'}</span>
					<span class="text-[11px] text-muted-foreground">{auth.isAdmin ? 'Administrator' : 'Member'}</span>
				</span>
			</DropdownMenu.Trigger>
			<DropdownMenu.Content side="top" align="start" class="w-56">
				<DropdownMenu.Item onclick={() => navigate('/profile')}>
					<UserIcon />
					Profile
				</DropdownMenu.Item>
				<DropdownMenu.Separator />
				<DropdownMenu.Label class="text-xs text-muted-foreground">Theme</DropdownMenu.Label>
				<DropdownMenu.Item onclick={() => theme.set('light')}>
					<SunIcon />
					Light {theme.value === 'light' ? '✓' : ''}
				</DropdownMenu.Item>
				<DropdownMenu.Item onclick={() => theme.set('dark')}>
					<MoonIcon />
					Dark {theme.value === 'dark' ? '✓' : ''}
				</DropdownMenu.Item>
				<DropdownMenu.Item onclick={() => theme.set('system')}>
					<MonitorIcon />
					System {theme.value === 'system' ? '✓' : ''}
				</DropdownMenu.Item>
				<DropdownMenu.Separator />
				<DropdownMenu.Item onclick={handleLogout}>
					<LogOutIcon />
					Log out
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</Sidebar.Footer>
</Sidebar.Root>
