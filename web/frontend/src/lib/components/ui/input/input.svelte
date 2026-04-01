<script lang="ts">
import { cn } from "$lib/utils"
import type { HTMLInputAttributes } from "svelte/elements"

type Props = HTMLInputAttributes & {
  class?: string
}

let { class: className, value = $bindable(), ...restProps }: Props = $props()
</script>

<input
  class={cn(
    "flex h-11 w-full rounded-md border border-(--input) bg-(--background) px-3 py-2 text-sm text-(--foreground) outline-none transition placeholder:text-(--muted-foreground) focus-visible:ring-2 focus-visible:ring-(--ring) focus-visible:ring-offset-2 focus-visible:ring-offset-(--background) disabled:cursor-not-allowed disabled:opacity-50",
    className
  )}
  bind:value
  {...restProps}
/>
