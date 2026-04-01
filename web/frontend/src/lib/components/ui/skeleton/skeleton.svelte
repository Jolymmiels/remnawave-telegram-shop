<script lang="ts">
import { cn } from "$lib/utils"
import type { HTMLAttributes } from "svelte/elements"

type Props = HTMLAttributes<HTMLDivElement> & {
  class?: string
}

let { class: className, ...restProps }: Props = $props()
</script>

<div class={cn("animate-pulse rounded-md bg-(--secondary)", className)} {...restProps}></div>
