<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { page } from '$app/state';
	import { api, unwrap, imageUrl, errorText, type Schemas } from '$lib/api/client';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import MediaCard from '$lib/components/media/media-card.svelte';
	import CollectionRequestDialog from '$lib/components/requests/collection-request-dialog.svelte';
	import { Button } from '$lib/components/ui/button';
	import { auth } from '$lib/stores/auth.svelte';
	import { onEvent } from '$lib/events.svelte';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';

	let collectionId = $derived(Number(page.params.id));

	let collection = $state<Schemas['CollectionDetail'] | null>(null);
	let loading = $state(true);
	let error = $state<Error | null>(null);

	let requesting = $state(false);

	// what the library holds, what is asked for, what can still be requested
	const parts = $derived(collection?.parts ?? []);
	const available = $derived(parts.filter((p) => p.availability === 'available').length);
	const requested = $derived(parts.filter((p) => p.availability !== 'available' && p.requestStatus).length);
	const missing = $derived(parts.filter((p) => p.availability !== 'available' && !p.requestStatus));

	async function load(spinner = true) {
		if (spinner) loading = true;
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

	// statuses follow requests and approvals without a reload (a whole saga sends a burst of events: one reload)
	$effect(() => {
		let timer: ReturnType<typeof setTimeout> | undefined;
		const soon = () => {
			clearTimeout(timer);
			timer = setTimeout(() => collection && load(false), 400);
		};
		const offs = [onEvent('request.updated', soon), onEvent('media.available', soon)];
		return () => {
			clearTimeout(timer);
			offs.forEach((off) => off());
		};
	});
</script>

<svelte:head>
	<title>{collection?.name ?? t('collection.fallback')} · Tipsarr</title>
</svelte:head>

{#if loading}
	<div class="grid gap-4">
		<Skeleton class="h-56 w-full rounded-lg" />
		<Skeleton class="h-6 w-1/3" />
	</div>
{:else if error}
	<div class="text-sm text-destructive">
		{t('common.error_prefix', { message: errorText(error) })}
		<button class="ml-2 underline" onclick={() => load()}>{t('common.retry')}</button>
	</div>
{:else if collection}
	<div class="-mx-4 -mt-20 md:-mx-8">
		<div class="relative h-56 w-full overflow-hidden md:h-80">
			{#if collection.backdropPath}
				<img src={imageUrl(collection.backdropPath, 'w1280')} alt="" class="h-full w-full object-cover" />
				<div class="absolute inset-0 bg-gradient-to-t from-background via-background/60 to-transparent"></div>
			{:else}
				<div class="h-full w-full bg-muted"></div>
			{/if}
			<button
				type="button"
				class="absolute top-20 left-4 z-10 flex cursor-pointer items-center gap-1.5 rounded-full bg-background/70 px-3 py-1.5 text-sm shadow backdrop-blur hover:bg-background md:left-8"
				onclick={() => history.back()}
			>
				<ArrowLeftIcon class="size-4" />
				{t('common.back')}
			</button>
			<div class="absolute right-0 bottom-0 left-0 p-4 md:px-8">
				<h1 class="font-bold text-3xl text-foreground drop-shadow md:text-4xl">{collection.name}</h1>
			</div>
		</div>

		<div class="grid gap-6 px-4 pb-6 md:px-8">
			{#if collection.overview}
				<p class="max-w-3xl text-sm leading-relaxed">{collection.overview}</p>
			{/if}

			<section class="grid gap-2">
				<div class="flex flex-wrap items-center justify-between gap-2">
					<div class="grid gap-0.5">
						<h2 class="font-semibold text-lg">{t('collection.movies')}</h2>
						<p class="text-sm text-muted-foreground">
							{t('collection.count_available', { count: available })} · {t('collection.count_requested', { count: requested })} · {t('collection.count_missing', { count: missing.length })}
						</p>
					</div>
					{#if auth.mayRequest('movie') && missing.length}
						<Button onclick={() => (requesting = true)}>
							<PlusIcon class="size-4" />
							{t('collection.request_missing', { count: missing.length })}
						</Button>
					{/if}
				</div>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(8.5rem,1fr))] gap-x-4 gap-y-6">
					{#each collection.parts as item (item.tmdbId)}
						<MediaCard {item} fluid />
					{/each}
				</div>
			</section>
		</div>
	</div>
{/if}

{#if collection}
	<CollectionRequestDialog bind:open={requesting} collectionId={collection.id} name={collection.name} parts={missing} onrequested={() => load(false)} />
{/if}
