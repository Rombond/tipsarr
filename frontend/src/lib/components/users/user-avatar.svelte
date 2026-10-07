<script lang="ts">
	import { Avatar, AvatarFallback } from '$lib/components/ui/avatar';
	import { auth } from '$lib/stores/auth.svelte';

	let { id, name, class: className = 'size-9' }: { id: string; name: string; class?: string } = $props();
	// the photo is probed first: a missing one (404) must never show a broken-image icon
	let loaded = $state(false);

	$effect(() => {
		const src = `/api/v1/users/${id}/avatar?v=${auth.avatarV}`;
		loaded = false;
		const probe = new Image();
		probe.onload = () => (loaded = true);
		probe.src = src;
		return () => {
			probe.onload = null;
		};
	});
</script>

<Avatar class={className}>
	{#if loaded}
		<img src="/api/v1/users/{id}/avatar?v={auth.avatarV}" alt="" class="aspect-square size-full object-cover" />
	{:else}
		<AvatarFallback>{(name || '?').slice(0, 2).toUpperCase()}</AvatarFallback>
	{/if}
</Avatar>
