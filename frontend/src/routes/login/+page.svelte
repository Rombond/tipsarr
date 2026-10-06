<script lang="ts">
	import { auth } from '$lib/stores/auth.svelte';
	import LoginForm from '$lib/components/auth/login-form.svelte';
	import SetupForm from '$lib/components/auth/setup-form.svelte';
</script>

<div class="flex min-h-svh items-center justify-center px-4">
	{#if auth.configured === false}
		<SetupForm />
	{:else}
		<LoginForm />
	{/if}
</div>
