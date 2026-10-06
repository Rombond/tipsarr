<script lang="ts">
	import { page } from '$app/state';
	import { api, unwrap, imageUrl, type Schemas } from '$lib/api/client';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';

	let personId = $derived(Number(page.params.id));

	let person = $state<Schemas['PersonDetail'] | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let expanded = $state(false);

	async function load() {
		loading = true;
		error = null;
		expanded = false;
		try {
			person = await unwrap(api.GET('/person/{id}', { params: { path: { id: personId } } }));
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (personId) load();
	});

	const longBio = $derived((person?.biography?.length ?? 0) > 650);
	const day = (d?: string) => (d ? new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' }) : '');
	function age(birth?: string, death?: string) {
		if (!birth) return null;
		const end = death ? new Date(death) : new Date();
		const b = new Date(birth);
		let a = end.getFullYear() - b.getFullYear();
		if (end < new Date(end.getFullYear(), b.getMonth(), b.getDate())) a--;
		return a;
	}
</script>

<svelte:head>
	<title>{person?.name ?? 'Person'} · Tipsarr</title>
</svelte:head>

{#if loading}
	<div class="grid gap-6 md:grid-cols-[16rem_minmax(0,1fr)]">
		<Skeleton class="aspect-[2/3] w-full max-w-64 rounded-xl" />
		<div class="grid content-start gap-3">
			<Skeleton class="h-9 w-64" />
			<Skeleton class="h-4 w-full" />
			<Skeleton class="h-4 w-full" />
			<Skeleton class="h-4 w-2/3" />
		</div>
	</div>
{:else if error}
	<div class="text-sm text-destructive">
		Error: {error}
		<button class="ml-2 underline" onclick={load}>Retry</button>
	</div>
{:else if person}
	<div class="grid grid-cols-[minmax(0,1fr)] gap-8">
		<button
			type="button"
			class="flex w-fit cursor-pointer items-center gap-1.5 rounded-full border border-border px-3 py-1.5 text-sm hover:bg-accent"
			onclick={() => history.back()}
		>
			<ArrowLeftIcon class="size-4" />
			Back
		</button>

		<h1 class="font-bold text-3xl md:hidden">{person.name}</h1>

		<!-- items-start: the photo keeps its own height instead of stretching to the biography -->
		<div class="grid items-start gap-6 md:grid-cols-[14rem_minmax(0,1fr)] lg:grid-cols-[16rem_minmax(0,1fr)] lg:gap-10">
			<aside class="grid gap-4 md:sticky md:top-24">
				<div class="aspect-[2/3] w-40 overflow-hidden rounded-xl bg-muted shadow-lg md:w-full md:max-w-64">
					{#if person.profilePath}
						<img src={imageUrl(person.profilePath, 'w500')} alt={person.name} class="h-full w-full object-cover" />
					{/if}
				</div>
				<dl class="grid max-w-64 gap-3 rounded-xl border border-border p-4 text-sm">
					{#if person.department}
						<div><dt class="text-xs text-muted-foreground">Known for</dt><dd>{person.department}</dd></div>
					{/if}
					{#if person.birthday}
						<div>
							<dt class="text-xs text-muted-foreground">{person.deathday ? 'Born' : 'Born (age)'}</dt>
							<dd>{day(person.birthday)}{#if !person.deathday}&nbsp;({age(person.birthday)}){/if}</dd>
						</div>
					{/if}
					{#if person.deathday}
						<div><dt class="text-xs text-muted-foreground">Died (age)</dt><dd>{day(person.deathday)}&nbsp;({age(person.birthday, person.deathday)})</dd></div>
					{/if}
					{#if person.birthplace}
						<div><dt class="text-xs text-muted-foreground">Place of birth</dt><dd>{person.birthplace}</dd></div>
					{/if}
				</dl>
			</aside>

			<div class="grid min-w-0 content-start gap-4">
				<h1 class="hidden font-bold text-4xl md:block">{person.name}</h1>
				{#if person.biography}
					<div>
						<h2 class="mb-1 font-semibold">Biography</h2>
						<p class="whitespace-pre-line text-[15px] leading-7 text-foreground/90 {longBio && !expanded ? 'line-clamp-[9]' : ''}">{person.biography}</p>
						{#if longBio}
							<button type="button" class="mt-2 cursor-pointer text-sm font-medium underline-offset-4 hover:underline" onclick={() => (expanded = !expanded)}>
								{expanded ? 'Show less' : 'Read more'}
							</button>
						{/if}
					</div>
				{:else}
					<p class="text-sm text-muted-foreground">No biography available.</p>
				{/if}
			</div>
		</div>

		{#if person.credits.length}
			<section class="grid gap-3">
				<h2 class="font-semibold text-lg">Known for <span class="text-sm font-normal text-muted-foreground">({person.credits.length})</span></h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
					{#each person.credits as item (`${item.type}:${item.tmdbId}`)}
						<MediaCard {item} fluid />
					{/each}
				</div>
			</section>
		{/if}
	</div>
{/if}
