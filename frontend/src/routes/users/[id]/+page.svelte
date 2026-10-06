<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';
	import ProfileView from '$lib/components/users/profile-view.svelte';

	const id = $derived(page.params.id ?? '');

	// your own profile lives at /profile (with the settings tab)
	$effect(() => {
		if (id && id === auth.user?.id) goto('/profile', { replaceState: true });
	});
</script>

<svelte:head>
	<title>{t('profile.title')} · Tipsarr</title>
</svelte:head>

{#if id && id !== auth.user?.id}
	{#key id}<ProfileView userId={id} />{/key}
{/if}
