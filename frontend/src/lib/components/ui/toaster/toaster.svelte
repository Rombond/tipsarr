<script lang="ts">
	import { toast } from '$lib/toast.svelte';
	import CheckIcon from '@lucide/svelte/icons/circle-check';
	import AlertIcon from '@lucide/svelte/icons/circle-alert';
	import InfoIcon from '@lucide/svelte/icons/info';
	import XIcon from '@lucide/svelte/icons/x';
</script>

<div class="pointer-events-none fixed right-4 bottom-20 z-[100] grid w-[min(24rem,calc(100vw-2rem))] gap-2 md:bottom-4" aria-live="polite">
	{#each toast.items as t (t.id)}
		<div
			class="pointer-events-auto flex items-start gap-2 rounded-lg border border-border bg-popover p-3 text-sm text-popover-foreground shadow-lg"
			role="status"
		>
			{#if t.kind === 'success'}<CheckIcon class="mt-0.5 size-4 shrink-0 text-emerald-500" />
			{:else if t.kind === 'error'}<AlertIcon class="mt-0.5 size-4 shrink-0 text-destructive" />
			{:else}<InfoIcon class="mt-0.5 size-4 shrink-0 text-muted-foreground" />{/if}
			<span class="min-w-0 flex-1 break-words">{t.text}</span>
			<button type="button" class="cursor-pointer text-muted-foreground hover:text-foreground" aria-label="Dismiss" onclick={() => toast.dismiss(t.id)}>
				<XIcon class="size-4" />
			</button>
		</div>
	{/each}
</div>
