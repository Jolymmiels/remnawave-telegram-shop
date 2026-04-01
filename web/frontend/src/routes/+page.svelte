<script lang="ts">
import { page } from "$app/state"
import { Badge } from "$lib/components/ui/badge"
import { buttonVariants } from "$lib/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card"
import { Separator } from "$lib/components/ui/separator"
import AppShell from "$lib/components/AppShell.svelte"
import { cn } from "$lib/utils"
import { ArrowRight, LockKeyhole, ShieldCheck, Wallet } from "@lucide/svelte"

let { data } = $props<{
  data: {
    plans: Array<{ months: number; price: number; currency: string }>
    trialDays: number
  }
}>()

const features = [
  {
    icon: ShieldCheck,
    title: "Приватность",
    text: "Защищённый доступ в сеть для повседневных задач, поездок и работы."
  },
  {
    icon: Wallet,
    title: "Гибкие тарифы",
    text: "Выберите удобный срок подписки и продлевайте доступ без лишних действий."
  },
  {
    icon: LockKeyhole,
    title: "Быстрый запуск",
    text: "Подключайте смартфон, ноутбук и настольный компьютер с одной подпиской."
  }
]
</script>

<svelte:head>
  <title>wowblvck VPN</title>
</svelte:head>

<AppShell title="wowblvck VPN" subtitle="Приватный интернет-доступ для работы, поездок и ежедневного использования">
  <section class="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
    <Card class="overflow-hidden">
      <CardHeader class="gap-5">
        <div class="flex flex-wrap items-center gap-3">
          <Badge>wowblvck VPN</Badge>
          <Badge variant="success">приватность</Badge>
          <Badge variant="default">скорость</Badge>
        </div>
        <CardTitle class="max-w-[14ch] text-4xl leading-[1.02] md:text-6xl">
          Спокойный и стабильный доступ к сети без лишнего шума.
        </CardTitle>
        <CardDescription class="max-w-2xl text-base leading-7">
          Оформите подписку, откройте доступ и используйте wowblvck VPN на тех устройствах,
          которые нужны вам каждый день.
        </CardDescription>
      </CardHeader>
      <CardContent class="space-y-6">
        <div class="grid gap-4 md:grid-cols-3">
          {#each features as feature}
            <div class="rounded-xl border border-(--border) bg-(--muted) p-4">
              <feature.icon class="mb-4 size-5" />
              <div class="text-sm font-semibold text-(--foreground)">{feature.title}</div>
              <p class="mt-2 text-sm leading-6 text-(--muted-foreground)">{feature.text}</p>
            </div>
          {/each}
        </div>

        <Separator />

        <div class="flex flex-wrap gap-3">
          <a class={buttonVariants({ variant: "default", size: "lg" })} href="/plans">
            Открыть тарифы
            <ArrowRight class="size-4" />
          </a>
          <a
            class={cn(buttonVariants({ variant: "secondary", size: "lg" }))}
            href={page.data.authenticated ? "/cabinet" : "/login"}
          >
            Перейти в кабинет
          </a>
        </div>
      </CardContent>
    </Card>

    <div class="grid gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Доступные тарифы</CardTitle>
          <CardDescription>Выберите срок подписки, который подходит именно вам.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-3">
          {#if data.plans.length === 0}
            <div class="rounded-lg border border-(--border) bg-(--muted) p-4 text-sm text-(--muted-foreground)">
              Тарифы пока не загружены. Попробуйте обновить страницу позже.
            </div>
          {:else}
            {#each data.plans as plan}
              <div class="flex items-center justify-between rounded-lg border border-(--border) bg-(--muted) p-4">
                <div>
                  <div class="text-sm font-semibold text-(--foreground)">{plan.months} мес.</div>
                </div>
                <div class="text-right text-sm font-medium text-(--foreground)">
                  {plan.price} {plan.currency}
                </div>
              </div>
            {/each}
          {/if}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Пробный период</CardTitle>
          <CardDescription>
            {#if data.trialDays > 0}
              Начните с пробного периода на {data.trialDays} дн. и оцените сервис в работе.
            {:else}
              Сейчас доступ открывается сразу после оформления подписки.
            {/if}
          </CardDescription>
        </CardHeader>
      </Card>
    </div>
  </section>
</AppShell>
