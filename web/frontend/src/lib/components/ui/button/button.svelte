<script lang="ts">
import { cn } from "$lib/utils"
import type { Snippet } from "svelte"
import type { HTMLButtonAttributes } from "svelte/elements"
import type { VariantProps } from "tailwind-variants"
// biome-ignore lint/style/useImportType: used at runtime in the template class expression
import { buttonVariants } from "./button-variants"

type Props = HTMLButtonAttributes &
  VariantProps<typeof buttonVariants> & {
    class?: string
    children?: Snippet
  }

let {
  class: className,
  variant = "default",
  size = "default",
  children,
  type = "button",
  ...restProps
}: Props = $props()
</script>

<button class={cn(buttonVariants({ variant, size }), className)} {type} {...restProps}>
  {@render children?.()}
</button>
