<script lang="ts">
	import { t, hasKey } from '$lib/i18n/index.svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { page } from '$app/state';
	import { api, unwrap, errorText } from '$lib/api/client';
	import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';

	let username = $state('');
	let password = $state('');
	let error: string | null = $state(null);
	let submitting = $state(false);
	let sso = $state(false);
	let methodsLoaded = $state(false);

	$effect(() => {
		unwrap(api.GET('/auth/methods'))
			.then((m) => (sso = m.sso))
			.catch(() => {})
			.finally(() => (methodsLoaded = true));
	});

	// Single sign-on replaces the form. It comes back when sign-in through it was refused, or on
	// /login?password=1 (the way in if the provider is down).
	const refused = $derived(page.url.searchParams.get('sso') === 'refused');
	const reason = $derived(page.url.searchParams.get('reason') ?? 'provider');
	const showForm = $derived(!sso || refused || page.url.searchParams.get('password') === '1');
	const refusedText = $derived(hasKey(`login.sso_refused.${reason}`) ? t(`login.sso_refused.${reason}` as 'login.sso_refused.provider') : t('login.sso_refused.provider'));

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

{#if methodsLoaded}
	<Card class="w-full max-w-sm">
		<CardHeader>
			<CardTitle>{t('login.title')}</CardTitle>
			<CardDescription>{sso && !showForm ? t('login.sso_desc') : t('login.desc')}</CardDescription>
		</CardHeader>
		<CardContent class="grid gap-4">
			{#if sso}
				{#if refused}
					<p class="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-300" role="alert">{refusedText}</p>
				{/if}
				<Button href="/api/v1/auth/oidc/login" data-sveltekit-reload variant={showForm ? 'outline' : 'default'} class="w-full">{t('login.sso_button')}</Button>
				{#if showForm}<p class="text-center text-xs text-muted-foreground">{t('login.or_password')}</p>{/if}
			{/if}
			{#if showForm}
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
			{/if}
		</CardContent>
	</Card>
{/if}
