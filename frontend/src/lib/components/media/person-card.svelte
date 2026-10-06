<script lang="ts">
	import { goto } from '$app/navigation';
	import { imageUrl } from '$lib/api/client';

	let { person, fluid = false }: { person: { id: number; name: string; profilePath?: string }; fluid?: boolean } = $props();

	const photo = $derived(imageUrl(person.profilePath, 'w185'));
</script>

<button
	type="button"
	class={fluid ? 'min-w-0 cursor-pointer text-left' : 'w-40 shrink-0 cursor-pointer snap-start text-left'}
	onclick={() => goto(`/person/${person.id}`)}
>
	<div class="aspect-[2/3] w-full overflow-hidden rounded-lg bg-muted shadow-sm transition-shadow hover:shadow-md">
		{#if photo}
			<img src={photo} alt={person.name} class="h-full w-full object-cover" loading="lazy" />
		{:else}
			<div class="flex h-full w-full items-center justify-center text-muted-foreground text-xs">No image</div>
		{/if}
	</div>
	<p class="mt-1.5 truncate text-sm font-medium">{person.name}</p>
</button>
