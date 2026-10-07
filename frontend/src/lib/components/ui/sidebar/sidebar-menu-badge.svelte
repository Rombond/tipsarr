<script lang="ts">
	import { cn, type WithElementRef } from "$lib/utils.js";
	import type { HTMLAttributes } from "svelte/elements";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLElement>> = $props();
</script>

<div
	bind:this={ref}
	data-slot="sidebar-menu-badge"
	data-sidebar="menu-badge"
	class={cn(
		"pointer-events-none absolute top-1/2 right-3 flex h-6 min-w-6 -translate-y-1/2 items-center justify-center rounded-full bg-primary px-2 text-sm leading-none font-semibold text-primary-foreground tabular-nums select-none group-data-[collapsible=icon]:hidden",
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>
