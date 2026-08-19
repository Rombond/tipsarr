<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';

	let { users = [], loading = false }: { users?: any[]; loading?: boolean } = $props();

	const PERMISSION_ADMIN = 2;

	function isAdmin(u: any) {
		return (u.permissions & PERMISSION_ADMIN) !== 0;
	}
</script>

<Card>
	<CardHeader>
		<CardTitle>Seerr Users</CardTitle>
	</CardHeader>
	<CardContent class="grid gap-2">
		{#if loading}
			{#each { length: 4 } as _, i (i)}
				<Skeleton class="h-6 w-full" />
			{/each}
		{:else if users.length === 0}
			<p class="text-muted-foreground text-sm">No users found.</p>
		{:else}
			{#each users as u (u.id)}
				<div class="flex items-center justify-between border-b border-border py-1.5 text-sm">
					<span>{u.displayName || u.username || u.jellyfinUsername}</span>
					<div class="flex gap-1.5">
						{#if isAdmin(u)}
							<Badge>Admin</Badge>
						{:else}
							<Badge variant="secondary">User</Badge>
						{/if}
					</div>
				</div>
			{/each}
		{/if}
	</CardContent>
</Card>
