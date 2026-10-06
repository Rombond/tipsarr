<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';
	import { getUsers } from '$lib/api/seerr';
	import UserList from '$lib/components/admin/user-list.svelte';

	let users: any[] = $state([]);
	let loading = $state(true);
	let error: Error | null = $state(null);

	async function load() {
		loading = true;
		error = null;
		try {
			const data = await getUsers();
			users = data.results || [];
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		if (!auth.isAdmin) {
			goto('/discover');
			return;
		}
		load();
	});
</script>

<svelte:head>
	<meta name="description" content="TipsArr Users" />
</svelte:head>

<div class="grid gap-4">
	<h1 class="font-bold text-2xl">Users</h1>
	{#if error}
		<div class="text-sm text-destructive">
			Error: {error.message}
			<button class="ml-2 underline" onclick={load}>Retry</button>
		</div>
	{:else}
		<UserList {users} {loading} />
	{/if}
</div>
