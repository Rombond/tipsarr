<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { errorText } from '$lib/api/client';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';

	let username = $state('');
	let password = $state('');
	let error: string | null = $state(null);
	let submitting = $state(false);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		submitting = true;
		try {
			await auth.login(username, password);
		} catch (err) {
			error = errorText(err);
		} finally {
			submitting = false;
		}
	}
</script>

<Card class="w-full max-w-sm">
	<CardHeader>
		<CardTitle>{t('login.title')}</CardTitle>
		<CardDescription>{t('login.desc')}</CardDescription>
	</CardHeader>
	<CardContent>
		<form class="grid gap-3" onsubmit={handleSubmit}>
			<Input placeholder={t('login.username')} bind:value={username} autocomplete="username" required />
			<Input placeholder={t('login.password')} type="password" bind:value={password} autocomplete="current-password" required />
			{#if error}
				<p class="text-sm text-destructive">{error}</p>
			{/if}
			<Button type="submit" disabled={submitting}>
				{submitting ? t('login.submitting') : t('login.submit')}
			</Button>
		</form>
	</CardContent>
</Card>
