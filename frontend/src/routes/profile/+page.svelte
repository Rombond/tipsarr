<script lang="ts">
	import { api, unwrap, expectOk, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';

	let region = $state(auth.user?.region ?? '');
	let language = $state(auth.user?.language ?? '');
	let message = $state<string | null>(null);
	let error = $state<string | null>(null);
	let saving = $state(false);
	let hidden = $state<Schemas['Item'][]>([]);

	$effect(() => {
		unwrap(api.GET('/blocklist'))
			.then((r) => (hidden = r))
			.catch(() => {});
	});

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		error = message = null;
		try {
			auth.user = await unwrap(api.PATCH('/me', { body: { region: region.trim().toUpperCase(), language: language.trim() } }));
			message = 'Saved. New language applies to titles you open from now on.';
		} catch (err) {
			error = (err as Error).message;
		} finally {
			saving = false;
		}
	}

	async function unhide(item: Schemas['Item']) {
		try {
			await expectOk(api.DELETE('/blocklist/{type}/{id}', { params: { path: { type: item.type, id: item.tmdbId } } }));
			hidden = hidden.filter((i) => !(i.type === item.type && i.tmdbId === item.tmdbId));
		} catch (e) {
			error = (e as Error).message;
		}
	}
</script>

<svelte:head>
	<title>Profile · Tipsarr</title>
</svelte:head>

<div class="grid grid-cols-[minmax(0,1fr)] gap-4 lg:grid-cols-2">
	<h1 class="font-bold text-3xl lg:col-span-2">Profile</h1>
	<Card>
		<CardHeader>
			<CardTitle>{auth.username}</CardTitle>
			<CardDescription>Region and language shape TMDB titles and the default box-office region.</CardDescription>
		</CardHeader>
		<CardContent>
			<form class="grid gap-3" onsubmit={save}>
				<Input placeholder="Region, e.g. FR (empty = none)" bind:value={region} maxlength={2} />
				<Input placeholder="Language, e.g. fr-FR (empty = English)" bind:value={language} />
				{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
				{#if message}<p class="text-sm text-muted-foreground">{message}</p>{/if}
				<Button type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
			</form>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>Hidden from my suggestions</CardTitle>
			<CardDescription>Titles you marked "not interested".</CardDescription>
		</CardHeader>
		<CardContent class="grid gap-1.5 text-sm">
			{#each hidden as item (`${item.type}:${item.tmdbId}`)}
				<div class="flex items-center justify-between gap-2 rounded-md border border-border px-3 py-2">
					<a href="/media/{item.type}/{item.tmdbId}" class="truncate hover:underline">{item.title}</a>
					<Button size="sm" variant="ghost" onclick={() => unhide(item)}>Unhide</Button>
				</div>
			{:else}
				<p class="text-muted-foreground">Nothing hidden.</p>
			{/each}
		</CardContent>
	</Card>
</div>
