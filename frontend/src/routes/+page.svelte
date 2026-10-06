<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.svelte';

	onMount(() => {
		goto(auth.status === 'authenticated' ? '/discover' : '/login');
	});
</script>
