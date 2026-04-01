<script lang="ts">
import { cn } from "$lib/utils"
import type { HTMLAttributes } from "svelte/elements"
import type { Snippet } from "svelte"

type Props = HTMLAttributes<HTMLParagraphElement> & {
  class?: string
  children?: Snippet
}

let { class: className, children, ...restProps }: Props = $props()
</script>

<p class={cn("text-sm leading-6 text-(--muted-foreground)", className)} {...restProps}>
  {@render children?.()}
</p>
