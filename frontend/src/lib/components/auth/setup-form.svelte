<script lang="ts">
	import { api, unwrap } from '$lib/api/client';
	import { auth } from '$lib/stores/auth.svelte';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';

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
					body: { jellyfinUrl: jellyfinUrl.trim(), ...(tmdbApiKey.trim() ? { tmdbApiKey: tmdbApiKey.trim() } : {}) },
				}),
			);
			auth.markConfigured();
		} catch (err) {
			error = (err as Error).message;
		} finally {
			submitting = false;
		}
	}
</script>

<Card class="w-full max-w-sm">
	<CardHeader>
		<CardTitle>Set up Tipsarr</CardTitle>
		<CardDescription>
			Point Tipsarr at your Jellyfin server. The first Jellyfin administrator to sign in becomes the Tipsarr admin.
		</CardDescription>
	</CardHeader>
	<CardContent>
		<form class="grid gap-3" onsubmit={handleSubmit}>
			<Input placeholder="Jellyfin URL (http://host:8096)" type="url" bind:value={jellyfinUrl} required />
			<Input placeholder="TMDB API key or read token (optional now)" bind:value={tmdbApiKey} autocomplete="off" />
			{#if error}
				<p class="text-sm text-destructive">{error}</p>
			{/if}
			<Button type="submit" disabled={submitting}>
				{submitting ? 'Checking Jellyfin…' : 'Save'}
			</Button>
		</form>
	</CardContent>
</Card>
