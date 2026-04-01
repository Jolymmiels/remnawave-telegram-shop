<script lang="ts">
import { cn } from "$lib/utils"
import { tv, type VariantProps } from "tailwind-variants"
import type { HTMLAttributes } from "svelte/elements"
import type { Snippet } from "svelte"

const badgeVariants = tv({
  base: "inline-flex items-center rounded-full border px-2.5 py-1 text-[11px] font-medium uppercase tracking-[0.22em]",
  variants: {
    variant: {
      default: "border-(--border) bg-(--secondary) text-(--muted-foreground)",
      success: "border-(--border) bg-(--secondary) text-(--muted-foreground)",
      warning: "border-(--border) bg-(--secondary) text-(--muted-foreground)",
      danger: "border-(--border) bg-(--secondary) text-(--muted-foreground)"
    }
  },
  defaultVariants: {
    variant: "default"
  }
})

type Props = HTMLAttributes<HTMLDivElement> &
  VariantProps<typeof badgeVariants> & {
    class?: string
    children?: Snippet
  }

let { class: className, variant = "default", children, ...restProps }: Props = $props()
</script>

<div class={cn(badgeVariants({ variant }), className)} {...restProps}>
  {@render children?.()}
</div>
