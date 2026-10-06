<script lang="ts">
	import { page } from '$app/state';
	import { api, unwrap, imageUrl, type Schemas } from '$lib/api/client';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import Carousel from '$lib/components/media/carousel.svelte';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';

	let personId = $derived(Number(page.params.id));

	let person = $state<Schemas['PersonDetail'] | null>(null);
	let loading = $state(true);
	let error = $state<Error | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			person = await unwrap(api.GET('/person/{id}', { params: { path: { id: personId } } }));
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (personId) load();
	});
</script>

<svelte:head>
	<title>{person?.name ?? 'Person'} · Tipsarr</title>
</svelte:head>

{#if loading}
	<div class="grid gap-4">
		<Skeleton class="h-48 w-32 rounded-lg" />
		<Skeleton class="h-6 w-1/3" />
		<Skeleton class="h-4 w-2/3" />
	</div>
{:else if error}
	<div class="text-sm text-destructive">
		Error: {error.message}
		<button class="ml-2 underline" onclick={load}>Retry</button>
	</div>
{:else if person}
	<div class="grid gap-6">
		<button
			type="button"
			class="flex w-fit cursor-pointer items-center gap-1.5 rounded-full border border-border px-3 py-1.5 text-sm hover:bg-accent"
			onclick={() => history.back()}
		>
			<ArrowLeftIcon class="size-4" />
			Back
		</button>

		<div class="flex flex-col gap-4 sm:flex-row">
			<div class="w-32 shrink-0 overflow-hidden rounded-lg bg-muted sm:w-44">
				{#if person.profilePath}
					<img src={imageUrl(person.profilePath, 'w342')} alt={person.name} class="aspect-[2/3] w-full object-cover" />
				{:else}
					<div class="aspect-[2/3] w-full"></div>
				{/if}
			</div>
			<div class="grid gap-2">
				<h1 class="font-bold text-2xl md:text-3xl">{person.name}</h1>
				<div class="flex flex-wrap gap-x-3 gap-y-1 text-sm text-muted-foreground">
					{#if person.department}<span>{person.department}</span>{/if}
					{#if person.birthplace}<span>· Born in {person.birthplace}</span>{/if}
				</div>
				{#if person.biography}
					<p class="max-w-3xl text-sm leading-relaxed">{person.biography}</p>
				{/if}
			</div>
		</div>

		<Carousel title="Known for" items={person.credits} />
	</div>
{/if}
