<script lang="ts">
	import { api, unwrap, expectOk, errorText, type Schemas } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { t, i18n, LOCALES, type Locale } from '$lib/i18n/index.svelte';
	import { toast } from '$lib/toast.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import SimpleSelect from '$lib/components/ui/simple-select.svelte';
	import { RATING_SOURCES } from '$lib/ratings.svelte';

	// the stored value is a TMDB language tag (fr-FR) or '' for "follow the browser"
	let language = $state(auth.user?.language ? (LOCALES.find((l) => l.code === auth.user!.language.slice(0, 2))?.tag ?? '') : '');
	let region = $state(auth.user?.region ?? '');
	let ratingSource = $state(auth.user?.ratingSource || 'tmdb');
	let error = $state<string | null>(null);
	let saving = $state(false);
	let hidden = $state<Schemas['Item'][]>([]);

	$effect(() => {
		unwrap(api.GET('/blocklist'))
			.then((r) => (hidden = r))
			.catch(() => {});
	});

	const languageOptions = $derived([{ value: '', label: t('lang.auto') }, ...LOCALES.map((l) => ({ value: l.tag, label: l.name }))]);

	async function save(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		error = null;
		try {
			auth.user = await unwrap(api.PATCH('/me', { body: { region: region.trim().toUpperCase(), language, ratingSource: ratingSource === 'tmdb' ? '' : (ratingSource as 'imdb' | 'metacritic' | 'rottenTomatoes') } }));
			// the interface follows immediately
			if (language) i18n.set(LOCALES.find((l) => l.tag === language)!.code as Locale);
			else i18n.useBrowser();
			toast.success(t('profile.saved_titles'));
		} catch (err) {
			error = errorText(err);
		} finally {
			saving = false;
		}
	}

	async function unhide(item: Schemas['Item']) {
		try {
			await expectOk(api.DELETE('/blocklist/{type}/{id}', { params: { path: { type: item.type, id: item.tmdbId } } }));
			hidden = hidden.filter((i) => !(i.type === item.type && i.tmdbId === item.tmdbId));
		} catch (e) {
			error = errorText(e);
		}
	}
</script>

<div class="grid grid-cols-[minmax(0,1fr)] items-start gap-4 lg:grid-cols-2">
	<Card>
		<CardHeader>
			<CardTitle>{t('profile.preferences')}</CardTitle>
			<CardDescription>{t('profile.card_desc')}</CardDescription>
		</CardHeader>
		<CardContent>
			<form class="grid gap-4" onsubmit={save}>
				<div class="grid gap-1.5 text-sm">
					{t('profile.language')}
					<SimpleSelect label={t('profile.language')} value={language} options={languageOptions} onchange={(v) => (language = v)} class="w-full" />
				</div>
				<div class="grid gap-1.5 text-sm">
					{t('profile.rating_source')}
					<SimpleSelect label={t('profile.rating_source')} value={ratingSource} options={RATING_SOURCES} onchange={(v) => (ratingSource = v)} class="w-full" />
					<span class="text-xs text-muted-foreground">{t('profile.rating_source_hint')}</span>
				</div>
				<label class="grid gap-1.5 text-sm">
					{t('profile.region')}
					<Input placeholder={t('profile.region_placeholder')} bind:value={region} maxlength={2} />
				</label>
				{#if error}<p class="text-sm text-destructive">{error}</p>{/if}
				<Button type="submit" disabled={saving}>{saving ? t('common.saving') : t('common.save')}</Button>
			</form>
		</CardContent>
	</Card>

	<Card>
		<CardHeader>
			<CardTitle>{t('profile.hidden_title')}</CardTitle>
			<CardDescription>{t('profile.hidden_desc')}</CardDescription>
		</CardHeader>
		<CardContent class="grid gap-1.5 text-sm">
			{#each hidden as item (`${item.type}:${item.tmdbId}`)}
				<div class="flex items-center justify-between gap-2 rounded-md border border-border px-3 py-2">
					<a href="/media/{item.type}/{item.tmdbId}" class="truncate hover:underline">{item.title}</a>
					<Button size="sm" variant="ghost" onclick={() => unhide(item)}>{t('profile.unhide')}</Button>
				</div>
			{:else}
				<p class="text-muted-foreground">{t('profile.nothing_hidden')}</p>
			{/each}
		</CardContent>
	</Card>
</div>
