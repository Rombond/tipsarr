<script lang="ts">
	import { page } from '$app/state';
	import { api, unwrap, imageUrl, type Schemas } from '$lib/api/client';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';

	let collectionId = $derived(Number(page.params.id));

	let collection = $state<Schemas['CollectionDetail'] | null>(null);
	let loading = $state(true);
	let error = $state<Error | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			collection = await unwrap(api.GET('/collection/{id}', { params: { path: { id: collectionId } } }));
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (collectionId) load();
	});
</script>

<svelte:head>
	<title>{collection?.name ?? 'Collection'} · Tipsarr</title>
</svelte:head>

{#if loading}
	<div class="grid gap-4">
		<Skeleton class="h-56 w-full rounded-lg" />
		<Skeleton class="h-6 w-1/3" />
	</div>
{:else if error}
	<div class="text-sm text-destructive">
		Error: {error.message}
		<button class="ml-2 underline" onclick={load}>Retry</button>
	</div>
{:else if collection}
	<div class="-mx-4 -mt-4 md:-mx-6 md:-mt-6">
		<div class="relative h-56 w-full overflow-hidden md:h-80">
			{#if collection.backdropPath}
				<img src={imageUrl(collection.backdropPath, 'w1280')} alt="" class="h-full w-full object-cover" />
				<div class="absolute inset-0 bg-gradient-to-t from-background via-background/60 to-transparent"></div>
			{:else}
				<div class="h-full w-full bg-muted"></div>
			{/if}
			<button
				type="button"
				class="absolute top-4 left-4 z-10 flex cursor-pointer items-center gap-1.5 rounded-full bg-background/80 px-3 py-1.5 text-sm shadow hover:bg-background"
				onclick={() => history.back()}
			>
				<ArrowLeftIcon class="size-4" />
				Back
			</button>
			<div class="absolute right-0 bottom-0 left-0 p-4 md:p-6">
				<h1 class="font-bold text-2xl text-foreground drop-shadow md:text-3xl">{collection.name}</h1>
			</div>
		</div>

		<div class="grid gap-6 px-4 pb-6 md:px-6">
			{#if collection.overview}
				<p class="max-w-3xl text-sm leading-relaxed">{collection.overview}</p>
			{/if}

			<section class="grid gap-2">
				<h2 class="font-semibold text-lg">Movies in this collection</h2>
				<div class="flex flex-wrap gap-4">
					{#each collection.parts as item (item.tmdbId)}
						<MediaCard {item} />
					{/each}
				</div>
			</section>
		</div>
	</div>
{/if}
