<script lang="ts">
import { Badge } from "$lib/components/ui/badge"
import { Button, buttonVariants } from "$lib/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card"
import { Separator } from "$lib/components/ui/separator"
import AppShell from "$lib/components/AppShell.svelte"
import { cn } from "$lib/utils"
import {
  BadgeCheck,
  Copy,
  ExternalLink,
  RefreshCcw,
  Rocket,
  Shield,
  WandSparkles
} from "@lucide/svelte"

type SessionResponse = {
  authenticated: boolean
  customer?: {
    id: number
    telegramId?: number | null
    login?: string | null
    language?: string
    subscriptionLink?: string | null
    expireAt?: string | null
  }
  purchase?: {
    id: number
    status: "new" | "pending" | "paid" | "cancel"
    invoiceType: string
    amount: number
    currency: string
    months: number
    createdAt: string
    paidAt?: string | null
    paymentUrl?: string | null
  } | null
  purchases?: Array<{
    id: number
    status: "new" | "pending" | "paid" | "cancel"
    invoiceType: string
    amount: number
    currency: string
    months: number
    createdAt: string
    paidAt?: string | null
    paymentUrl?: string | null
  }>
  trialAvailable?: boolean
}

let { data, form } = $props<{
  data: SessionResponse
  form?: {
    message?: string
  }
}>()

let copyState = $state<"idle" | "copied" | "error">("idle")

function purchaseVariant(
  status: SessionResponse["purchase"] extends infer P
    ? P extends { status: infer S }
      ? S
      : never
    : never
) {
  if (status === "paid") return "success"
  if (status === "cancel") return "danger"
  return "warning"
}

function purchaseStatusLabel(status: "new" | "pending" | "paid" | "cancel") {
  if (status === "paid") return "оплачено"
  if (status === "cancel") return "отменено"
  return "ожидает оплаты"
}

async function copySubscriptionLink(link: string) {
  try {
    await navigator.clipboard.writeText(link)
    copyState = "copied"
    setTimeout(() => {
      copyState = "idle"
    }, 2000)
  } catch {
    copyState = "error"
    setTimeout(() => {
      copyState = "idle"
    }, 2500)
  }
}
</script>

<svelte:head>
  <title>Кабинет</title>
</svelte:head>

<AppShell title="Личный кабинет wowblvck VPN" subtitle="Подписка, заказы и подключение устройств в одном месте">
  <section class="space-y-6">
    {#if !data.authenticated}
      <Card class="mx-auto max-w-2xl">
        <CardHeader>
          <CardTitle>Нужен вход</CardTitle>
          <CardDescription>Для доступа к кабинету сначала выполните вход или регистрацию.</CardDescription>
        </CardHeader>
        <CardContent>
          <a class={buttonVariants({ variant: "default", size: "lg" })} href="/login">Перейти ко входу</a>
        </CardContent>
      </Card>
    {:else}
      <div class="grid gap-5 xl:grid-cols-[1.15fr_0.85fr]">
        <Card>
          <CardHeader>
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="space-y-3">
                <div class="flex items-center gap-3">
                  <div class="flex size-12 items-center justify-center rounded-lg border border-(--border) bg-(--muted)">
                    <Shield class="size-5" />
                  </div>
                  <div>
                    <CardTitle class="text-3xl">{data.customer?.login ?? "Пользователь"}</CardTitle>
                    <CardDescription>Ваш персональный кабинет wowblvck VPN</CardDescription>
                  </div>
                </div>
                <div class="flex flex-wrap gap-2">
                  <Badge variant={data.customer?.subscriptionLink ? "success" : "warning"}>
                    {data.customer?.subscriptionLink ? "подписка активна" : "подписка не активирована"}
                  </Badge>
                  <Badge variant={data.customer?.telegramId ? "success" : "warning"}>
                    {data.customer?.telegramId ? "telegram привязан" : "telegram не привязан"}
                  </Badge>
                  {#if data.purchases?.length}
                    <Badge variant={purchaseVariant(data.purchases[0].status)}>
                      {purchaseStatusLabel(data.purchases[0].status)}
                    </Badge>
                  {/if}
                </div>
              </div>
              <form method="POST" action="?/logout">
                <Button variant="secondary" type="submit">Выйти</Button>
              </form>
            </div>
          </CardHeader>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Быстрые действия</CardTitle>
            <CardDescription>Продление доступа, проверка статуса и активация пробного периода.</CardDescription>
          </CardHeader>
          <CardContent class="flex flex-wrap gap-3">
            <form method="POST" action="?/refresh">
              <Button variant="secondary" type="submit">
                <RefreshCcw class="size-4" />
                Проверить статус
              </Button>
            </form>

            <a class={buttonVariants({ variant: "default" })} href="/plans">
              <Rocket class="size-4" />
              Купить подписку
            </a>

            {#if data.trialAvailable}
              <form method="POST" action="?/activateTrial">
                <Button variant="outline" type="submit">
                  <WandSparkles class="size-4" />
                  Активировать пробный период
                </Button>
              </form>
            {/if}

            {#if !data.customer?.telegramId}
              <form method="POST" action="?/linkTelegram">
                <Button variant="outline" type="submit">
                  <ExternalLink class="size-4" />
                  Привязать Telegram
                </Button>
              </form>
            {/if}
          </CardContent>
        </Card>
      </div>

      <div class="grid gap-5 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Подписка</CardTitle>
            <CardDescription>Используйте ссылку подписки для подключения приложений на ваших устройствах.</CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            {#if data.customer?.subscriptionLink}
              <div class="rounded-lg border border-(--border) bg-(--muted) p-4">
                <div class="flex items-center gap-2 text-sm font-medium text-(--foreground)">
                  <BadgeCheck class="size-4" />
                  Подписка активна
                </div>
                <button
                  class="mt-3 w-full break-all rounded-md border border-(--border) px-3 py-3 text-left text-sm leading-6 text-(--foreground) transition hover:bg-(--accent)"
                  onclick={() => copySubscriptionLink(data.customer!.subscriptionLink!)}
                  type="button"
                >
                  {data.customer.subscriptionLink}
                </button>
                <div class="mt-3 flex flex-wrap gap-2">
                  <Button
                    variant="secondary"
                    type="button"
                    onclick={() => copySubscriptionLink(data.customer!.subscriptionLink!)}
                  >
                    <Copy class="size-4" />
                    {copyState === "copied"
                      ? "Скопировано"
                      : copyState === "error"
                        ? "Не удалось скопировать"
                        : "Скопировать ссылку"}
                  </Button>
                </div>
                {#if data.customer.expireAt}
                  <div class="mt-3 text-sm text-(--muted-foreground)">
                    Действует до: {new Date(data.customer.expireAt).toLocaleString("ru-RU")}
                  </div>
                {/if}
              </div>
            {:else}
              <div class="rounded-lg border border-(--border) bg-(--muted) p-4 text-sm leading-6 text-(--muted-foreground)">
                Подписка ещё не активирована. Выберите тариф или начните с пробного периода.
              </div>
            {/if}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Подключение</CardTitle>
            <CardDescription>Откройте инструкцию и настройте wowblvck VPN на нужной платформе.</CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="rounded-lg border border-(--border) bg-(--muted) p-4 text-sm leading-6 text-(--muted-foreground)">
              В инструкции доступны приложения для iOS, Android, Windows, macOS и Linux.
            </div>
            <a class={buttonVariants({ variant: "secondary" })} href="/install-guide">
              <ExternalLink class="size-4" />
              Открыть инструкцию
            </a>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>История покупок</CardTitle>
          <CardDescription>Здесь отображаются все заказы и их текущие статусы.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
          {#if data.purchases?.length}
            <div class="space-y-4">
              <div class="flex flex-wrap gap-3">
                <form method="POST" action="?/refresh">
                  <Button variant="secondary" type="submit">
                    <RefreshCcw class="size-4" />
                    Обновить статусы
                  </Button>
                </form>
              </div>

              {#each data.purchases as purchase}
                <div class="rounded-xl border border-(--border) bg-(--muted) p-4">
                  <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div class="space-y-3">
                      <div class="text-xl font-semibold text-(--foreground)">
                        {purchase.months} мес. · {purchase.amount} {purchase.currency}
                      </div>
                      <div class="flex flex-wrap gap-2">
                        <Badge variant={purchaseVariant(purchase.status)}>
                          {purchaseStatusLabel(purchase.status)}
                        </Badge>
                        <Badge>{purchase.invoiceType === "yookassa" ? "YooKassa" : purchase.invoiceType}</Badge>
                      </div>
                      <Separator class="max-w-sm" />
                      <div class="space-y-2 text-sm text-(--muted-foreground)">
                        <div>Заказ #{purchase.id}</div>
                        <div>Создана: {new Date(purchase.createdAt).toLocaleString("ru-RU")}</div>
                        {#if purchase.paidAt}
                          <div>Оплачена: {new Date(purchase.paidAt).toLocaleString("ru-RU")}</div>
                        {/if}
                      </div>
                    </div>

                    {#if purchase.paymentUrl && purchase.status !== "paid" && purchase.status !== "cancel"}
                      <a
                        class={cn(buttonVariants({ variant: "default" }))}
                        href={purchase.paymentUrl}
                        target="_blank"
                        rel="noreferrer"
                      >
                        <ExternalLink class="size-4" />
                        Открыть оплату
                      </a>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          {:else}
            <div class="rounded-lg border border-(--border) bg-(--muted) p-4 text-sm text-(--muted-foreground)">
              Покупок пока не было.
            </div>
          {/if}
        </CardContent>
      </Card>

      {#if form?.message}
        <div class="rounded-lg border border-(--border) bg-(--muted) px-4 py-3 text-sm text-(--foreground)">
          {form.message}
        </div>
      {/if}
    {/if}
  </section>
</AppShell>
