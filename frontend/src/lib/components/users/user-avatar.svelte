<script lang="ts">
	import { Avatar, AvatarFallback } from '$lib/components/ui/avatar';

	let { id, name, class: className = 'size-9' }: { id: string; name: string; class?: string } = $props();
	let failed = $state(false);

	$effect(() => {
		id;
		failed = false;
	});
</script>

<Avatar class={className}>
	{#if !failed}
		<img src="/api/v1/users/{id}/avatar" alt="" class="aspect-square size-full object-cover" onerror={() => (failed = true)} />
	{/if}
	{#if failed}<AvatarFallback>{(name || '?').slice(0, 2).toUpperCase()}</AvatarFallback>{/if}
</Avatar>
