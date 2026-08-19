<script lang="ts">
	import { Skeleton } from '$lib/components/ui/skeleton';
	import RequestRow from './request-row.svelte';

	let {
		requests = [],
		loading = false,
		isAdmin = false,
		onApprove,
		onDecline,
		onRetry,
		onDelete,
	}: {
		requests?: any[];
		loading?: boolean;
		isAdmin?: boolean;
		onApprove?: (id: number) => void;
		onDecline?: (id: number) => void;
		onRetry?: (id: number) => void;
		onDelete?: (id: number) => void;
	} = $props();
</script>

<div class="grid">
	{#if loading}
		{#each { length: 6 } as _, i (i)}
			<div class="flex items-center gap-3 border-b border-border py-2">
				<Skeleton class="h-16 w-11 rounded" />
				<Skeleton class="h-4 flex-1" />
			</div>
		{/each}
	{:else if requests.length === 0}
		<p class="text-muted-foreground text-sm py-4">No requests found.</p>
	{:else}
		{#each requests as request (request.id)}
			<RequestRow {request} {isAdmin} {onApprove} {onDecline} {onRetry} {onDelete} />
		{/each}
	{/if}
</div>
