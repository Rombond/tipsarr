<script lang="ts">
	import { goto } from '$app/navigation';
	import { posterUrl } from '$lib/api/seerr';

	let { person }: { person: { id: number; name: string; profilePath?: string | null } } = $props();
</script>

<button
	type="button"
	class="w-40 shrink-0 snap-start text-left"
	onclick={() => goto(`/person/${person.id}`)}
>
	<div class="aspect-[2/3] w-full overflow-hidden rounded-lg bg-muted shadow-sm transition-shadow hover:shadow-md">
		{#if person.profilePath}
			<img src={posterUrl(person.profilePath)} alt={person.name} class="h-full w-full object-cover" loading="lazy" />
		{:else}
			<div class="flex h-full w-full items-center justify-center text-muted-foreground text-xs">No image</div>
		{/if}
	</div>
	<p class="mt-1.5 truncate text-sm font-medium">{person.name}</p>
</button>
