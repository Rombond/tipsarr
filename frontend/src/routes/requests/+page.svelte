<script lang="ts">
	import { getRequests, updateRequestStatus, retryRequest, deleteRequest } from '$lib/api/seerr';
	import { auth } from '$lib/stores/auth.svelte';
	import RequestList from '$lib/components/requests/request-list.svelte';

	let requests: any[] = $state([]);
	let loading = $state(true);
	let error: Error | null = $state(null);

	async function load() {
		loading = true;
		error = null;
		try {
			const data = await getRequests({
				requestedBy: auth.isAdmin ? undefined : auth.seerrUser?.id,
			});
			requests = data.results || [];
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	async function handleApprove(id: number) {
		await updateRequestStatus(id, 'approve');
		load();
	}

	async function handleDecline(id: number) {
		await updateRequestStatus(id, 'decline');
		load();
	}

	async function handleRetry(id: number) {
		await retryRequest(id);
		load();
	}

	async function handleDelete(id: number) {
		await deleteRequest(id);
		load();
	}

	load();
</script>

<svelte:head>
	<meta name="description" content="TipsArr Requests" />
</svelte:head>

<div class="grid gap-4">
	<h1 class="font-bold text-2xl">Requests</h1>
	{#if error}
		<div class="text-sm text-destructive">
			Error: {error.message}
			<button class="ml-2 underline" onclick={load}>Retry</button>
		</div>
	{:else}
		<RequestList
			{requests}
			{loading}
			isAdmin={auth.isAdmin}
			onApprove={handleApprove}
			onDecline={handleDecline}
			onRetry={handleRetry}
			onDelete={handleDelete}
		/>
	{/if}
</div>
