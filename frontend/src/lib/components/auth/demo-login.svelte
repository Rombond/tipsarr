<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { errorText, ApiError } from '$lib/api/client';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';

	let error: string | null = $state(null);
	let busy = $state(false);

	async function enter(role: 'admin' | 'user') {
		error = null;
		busy = true;
		try {
			await auth.demoLogin(role);
		} catch (e) {
			if (e instanceof ApiError && e.code === 'demo_expired') {
				location.reload();
				return;
			}
			error = errorText(e);
		} finally {
			busy = false;
		}
	}
</script>

<Card class="w-full max-w-sm">
	<CardHeader>
		<CardTitle>{t('login.demo_title')}</CardTitle>
		<CardDescription>{t('login.demo_desc')}</CardDescription>
	</CardHeader>
	<CardContent class="grid gap-3">
		<Button disabled={busy} onclick={() => enter('admin')}>{t('login.demo_admin')}</Button>
		<Button variant="outline" disabled={busy} onclick={() => enter('user')}>{t('login.demo_user')}</Button>
		{#if error}
			<p class="text-sm text-destructive">{error}</p>
		{/if}
	</CardContent>
</Card>
