<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getPersonDetails, getPersonCombinedCredits, posterUrl } from '$lib/api/seerr';
	import { fromSeerrResult, type MediaItem } from '$lib/api/media';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import Carousel from '$lib/components/media/carousel.svelte';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';

	let personId = $derived(Number(page.params.id));

	let person: any = $state(null);
	let loading = $state(true);
	let error: Error | null = $state(null);
	let credits: MediaItem[] = $state([]);
	let creditsLoading = $state(true);

	async function load() {
		loading = true;
		error = null;
		try {
			person = await getPersonDetails(personId);
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}

		creditsLoading = true;
		try {
			const data = await getPersonCombinedCredits(personId);
			const all = [...(data.cast || []), ...(data.crew || [])];
			const seen = new Set<number>();
			credits = all
				.filter((c: any) => {
					const key = c.id;
					if (seen.has(key)) return false;
					seen.add(key);
					return true;
				})
				.sort((a: any, b: any) => (b.popularity || 0) - (a.popularity || 0))
				.map((c: any) => fromSeerrResult(c));
		} catch {
			credits = [];
		} finally {
			creditsLoading = false;
		}
	}

	$effect(() => {
		if (personId) load();
	});

	function goToCredit(item: MediaItem) {
		goto(`/media/${item.mediaType}/${item.tmdbId}`);
	}
</script>

<svelte:head>
	<meta name="description" content="TipsArr person details" />
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
			class="flex w-fit items-center gap-1.5 rounded-full border border-border px-3 py-1.5 text-sm hover:bg-accent"
			onclick={() => history.back()}
		>
			<ArrowLeftIcon class="size-4" />
			Back
		</button>

		<div class="flex flex-col gap-4 sm:flex-row">
			<div class="w-32 shrink-0 overflow-hidden rounded-lg bg-muted sm:w-44">
				{#if person.profilePath}
					<img src={posterUrl(person.profilePath)} alt={person.name} class="aspect-[2/3] w-full object-cover" />
				{:else}
					<div class="aspect-[2/3] w-full"></div>
				{/if}
			</div>
			<div class="grid gap-2">
				<h1 class="font-bold text-2xl md:text-3xl">{person.name}</h1>
				<div class="flex flex-wrap gap-x-3 gap-y-1 text-sm text-muted-foreground">
					{#if person.knownForDepartment}<span>{person.knownForDepartment}</span>{/if}
					{#if person.placeOfBirth}<span>· Born in {person.placeOfBirth}</span>{/if}
				</div>
				{#if person.biography}
					<p class="max-w-3xl text-sm leading-relaxed">{person.biography}</p>
				{/if}
			</div>
		</div>

		<Carousel
			title="Known for"
			items={credits}
			loading={creditsLoading}
			hasMore={false}
			onSelect={goToCredit}
		/>
	</div>
{/if}
