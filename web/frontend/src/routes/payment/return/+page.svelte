<script lang="ts">
import { Badge } from "$lib/components/ui/badge"
import { buttonVariants } from "$lib/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card"
import AppShell from "$lib/components/AppShell.svelte"
import { api } from "$lib/api"
import { cn } from "$lib/utils"
import { CheckCircle2, LoaderCircle, TriangleAlert } from "@lucide/svelte"
import { onMount } from "svelte"

type PurchaseStatusResponse = {
  id: number
  status: "new" | "pending" | "paid" | "cancel"
  subscriptionLink?: string | null
}

let status = $state<"checking" | "active" | "waiting" | "error">("checking")
let attempts = $state(0)
let message = $state("Проверяем активацию подписки...")

onMount(() => {
  let cancelled = false
  const purchaseId = new URL(window.location.href).searchParams.get("purchaseId")

  const run = async () => {
    if (!purchaseId) {
      status = "error"
      message = "Не удалось определить номер покупки после возврата из YooKassa."
      return
    }

    for (let index = 0; index < 12; index += 1) {
      if (cancelled) return

      attempts = index + 1

      try {
        const purchase = await api<PurchaseStatusResponse>(
          `/api/v1/purchases/status?id=${purchaseId}`
        )

        if (purchase.status === "paid" && purchase.subscriptionLink) {
          status = "active"
          message = "Подписка активирована. Перенаправляем в кабинет..."
          setTimeout(() => {
            if (!cancelled) {
              window.location.href = "/cabinet"
            }
          }, 1200)
          return
        }

        if (purchase.status === "cancel") {
          status = "error"
          message = "Платёж был отменён. Можно вернуться к тарифам и создать новый заказ."
          return
        }

        status = "waiting"
        message = "Платёж найден, ждём подтверждение и выдачу подписки."
      } catch {
        status = "error"
        message = "Не удалось проверить статус автоматически. Можно открыть кабинет вручную."
        return
      }

      await new Promise((resolve) => setTimeout(resolve, 3000))
    }

    if (!cancelled) {
      status = "waiting"
      message = "Автоматическая проверка завершена. Если платёж уже проведён, обновите кабинет."
    }
  }

  run()

  return () => {
    cancelled = true
  }
})
</script>

<svelte:head>
  <title>Возврат из оплаты</title>
</svelte:head>

<AppShell title="Оплата wowblvck VPN" subtitle="Подтверждаем платёж и подготавливаем доступ">
  <section class="mx-auto max-w-3xl">
    <Card>
      <CardHeader>
        <div class="flex items-center gap-3">
          {#if status === "active"}
            <CheckCircle2 class="size-6" />
          {:else if status === "error"}
            <TriangleAlert class="size-6" />
          {:else}
            <LoaderCircle class="size-6 animate-spin" />
          {/if}
          <div>
            <CardTitle>Платёж принят в обработку</CardTitle>
            <CardDescription>Обычно подтверждение занимает всего несколько секунд.</CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent class="space-y-5">
        <div class="flex flex-wrap gap-2">
          <Badge variant={status === "active" ? "success" : status === "error" ? "danger" : "warning"}>
            {status === "active" ? "подписка активирована" : status === "error" ? "ошибка проверки" : "идёт обработка"}
          </Badge>
          <Badge>проверка #{attempts}</Badge>
        </div>

        <div class="rounded-lg border border-(--border) bg-(--muted) p-4 text-sm leading-7 text-(--muted-foreground)">
          {message}
        </div>

        <div class="flex flex-wrap gap-3">
          <a class={buttonVariants({ variant: "default", size: "lg" })} href="/cabinet">Открыть кабинет</a>
          <a class={cn(buttonVariants({ variant: "secondary", size: "lg" }))} href="/plans">
            Вернуться к тарифам
          </a>
        </div>
      </CardContent>
    </Card>
  </section>
</AppShell>
