<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Avatar, AvatarFallback } from '$lib/components/ui/avatar';
	import CompassIcon from '@lucide/svelte/icons/compass';
	import ListIcon from '@lucide/svelte/icons/list';
	import UsersIcon from '@lucide/svelte/icons/users';
	import SettingsIcon from '@lucide/svelte/icons/settings';
	import LogOutIcon from '@lucide/svelte/icons/log-out';

	const navItems = $derived(
		[
			{ href: '/discover', label: 'Discover', icon: CompassIcon },
			{ href: '/requests', label: 'Requests', icon: ListIcon },
			...(auth.isAdmin
				? [
						{ href: '/admin/users', label: 'Users', icon: UsersIcon },
						{ href: '/admin/settings', label: 'Settings', icon: SettingsIcon },
					]
				: []),
		],
	);

	function initials(name: string | null) {
		if (!name) return '?';
		return name.slice(0, 2).toUpperCase();
	}

	async function handleLogout() {
		await auth.logout();
		goto('/login');
	}
</script>

<Sidebar.Root>
	<Sidebar.Header>
		<span class="px-2 py-1.5 font-bold text-lg">TipsArr</span>
	</Sidebar.Header>
	<Sidebar.Content>
		<Sidebar.Group>
			<Sidebar.Menu>
				{#each navItems as item (item.href)}
					<Sidebar.MenuItem>
						<Sidebar.MenuButton isActive={page.url.pathname.startsWith(item.href)}>
							{#snippet child({ props })}
								<a href={item.href} {...props}>
									<item.icon />
									<span>{item.label}</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				{/each}
			</Sidebar.Menu>
		</Sidebar.Group>
	</Sidebar.Content>
	<Sidebar.Footer>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-sidebar-accent">
				<Avatar class="size-6">
					<AvatarFallback>{initials(auth.username)}</AvatarFallback>
				</Avatar>
				<span class="truncate">{auth.username || 'Account'}</span>
			</DropdownMenu.Trigger>
			<DropdownMenu.Content side="top" align="start" class="w-48">
				<DropdownMenu.Item onclick={handleLogout}>
					<LogOutIcon />
					Log out
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</Sidebar.Footer>
</Sidebar.Root>
