<script lang="ts">
import { CreditCard, LayoutDashboard, LogIn, Sparkles } from "@lucide/svelte";
import { page } from "$app/state";
import { Badge } from "$lib/components/ui/badge";
import { Button, buttonVariants } from "$lib/components/ui/button";
import { Separator } from "$lib/components/ui/separator";
import { cn } from "$lib/utils";

let {
	title = "wowblvck VPN",
	subtitle = "Приватный доступ в интернет на ваших устройствах",
	children,
} = $props<{
	title?: string;
	subtitle?: string;
	children?: import("svelte").Snippet;
}>();
</script>

<div class="min-h-screen px-3 py-4 md:px-6 md:py-6">
  <div class="shell-card mx-auto flex max-w-7xl flex-col overflow-hidden">
    <header class="border-b border-(--border) px-5 py-5 md:px-7">
      <div class="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
        <div class="space-y-4">
          <div class="flex flex-wrap items-center gap-3">
            <Badge>приватный доступ</Badge>
            <Badge variant="success">стабильное соединение</Badge>
            <Badge variant="default">несколько устройств</Badge>
          </div>
          <div class="space-y-2">
            <div class="flex items-center gap-3">
              <div class="flex items-center gap-3">
                <img
                  alt="wowblvck VPN"
                  class="h-12 w-auto rounded-md border border-(--border) bg-[#08111F] p-1.5 shadow-sm"
                  src="/logo-wordmark.svg"
                />
                <div class="text-xl font-semibold tracking-tight text-(--foreground)">{title}</div>
              </div>
            </div>
            <p class="max-w-2xl text-sm leading-6 text-(--muted-foreground)">{subtitle}</p>
          </div>
        </div>

        <div class="flex flex-col gap-4 md:items-end">
          <nav class="flex flex-wrap gap-2">
            <a class={cn(buttonVariants({ variant: "ghost", size: "sm" }), "text-(--foreground)")} href="/">
              <Sparkles class="size-4" />
              Главная
            </a>
            <a class={cn(buttonVariants({ variant: "ghost", size: "sm" }), "text-(--foreground)")} href="/plans">
              <CreditCard class="size-4" />
              Тарифы
            </a>
            {#if page.data.authenticated}
              <a class={cn(buttonVariants({ variant: "ghost", size: "sm" }), "text-(--foreground)")} href="/cabinet">
                <LayoutDashboard class="size-4" />
                Кабинет
              </a>
            {:else}
              <a class={buttonVariants({ variant: "secondary", size: "sm" })} href="/login">
                <LogIn class="size-4" />
                Вход
              </a>
            {/if}
          </nav>
          <div class="hidden w-full max-w-md lg:block">
            <Separator />
          </div>
        </div>
      </div>
    </header>

    <main class="px-5 py-6 md:px-7 md:py-8">
      {@render children?.()}
    </main>
  </div>
</div>
