<script lang="ts">
import { cn } from "$lib/utils"
import type { HTMLAttributes } from "svelte/elements"
import type { Snippet } from "svelte"

type Props = HTMLAttributes<HTMLDivElement> & {
  class?: string
  children?: Snippet
}

let { class: className, children, ...restProps }: Props = $props()
</script>

<div class={cn("text-lg font-semibold tracking-tight text-(--foreground)", className)} {...restProps}>
  {@render children?.()}
</div>
