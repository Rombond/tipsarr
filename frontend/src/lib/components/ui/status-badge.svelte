<script lang="ts">
	import { STATUS, type StatusKey } from '$lib/status';
	import type { Snippet } from 'svelte';

	let { status, children, class: className = '' }: { status: StatusKey; children: Snippet; class?: string } = $props();
	const s = $derived(STATUS[status]);
</script>

<span class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-semibold {s.soft} {className}">
	<s.icon class="size-3" strokeWidth={2.5} />
	{@render children()}
</span>
