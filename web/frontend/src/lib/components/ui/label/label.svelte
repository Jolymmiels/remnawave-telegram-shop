<script lang="ts">
import { cn } from "$lib/utils"
import type { HTMLLabelAttributes } from "svelte/elements"
import type { Snippet } from "svelte"

type Props = HTMLLabelAttributes & {
  class?: string
  children?: Snippet
}

let { class: className, children, ...restProps }: Props = $props()
</script>

<label class={cn("text-sm font-medium text-(--foreground)", className)} {...restProps}>
  {@render children?.()}
</label>
