<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { api, unwrap, errorText } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';

	let setupToken = $state('');
	let jellyfinUrl = $state('');
	let tmdbApiKey = $state('');
	let error: string | null = $state(null);
	let submitting = $state(false);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		submitting = true;
		try {
			await unwrap(
				api.POST('/setup', {
					body: { setupToken: setupToken.trim(), jellyfinUrl: jellyfinUrl.trim(), ...(tmdbApiKey.trim() ? { tmdbApiKey: tmdbApiKey.trim() } : {}) },
				}),
			);
			auth.markConfigured();
		} catch (err) {
			error = errorText(err);
		} finally {
			submitting = false;
		}
	}
</script>

<Card class="w-full max-w-sm">
	<CardHeader>
		<CardTitle>{t('setup.title')}</CardTitle>
		<CardDescription>
			{t('setup.desc')}
		</CardDescription>
	</CardHeader>
	<CardContent>
		<form class="grid gap-3" onsubmit={handleSubmit}>
			{#if auth.setupTokenRequired}
				<Input placeholder={t('setup.token')} bind:value={setupToken} autocomplete="off" required />
			{/if}
			<Input placeholder={t('setup.jellyfin_url')} type="url" bind:value={jellyfinUrl} required />
			<Input placeholder={t('setup.tmdb')} bind:value={tmdbApiKey} autocomplete="off" />
			{#if error}
				<p class="text-sm text-destructive">{error}</p>
			{/if}
			<Button type="submit" disabled={submitting}>
				{submitting ? t('setup.checking') : t('setup.submit')}
			</Button>
		</form>
	</CardContent>
</Card>
