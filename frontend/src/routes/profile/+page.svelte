<script lang="ts">
	import { t } from '$lib/i18n/index.svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import ProfileView from '$lib/components/users/profile-view.svelte';
</script>

<svelte:head>
	<title>{t('profile.title')} · Tipsarr</title>
</svelte:head>

{#if auth.user}<ProfileView userId={auth.user.id} self />{/if}
