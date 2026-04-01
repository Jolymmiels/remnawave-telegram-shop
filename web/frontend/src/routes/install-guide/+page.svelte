<script lang="ts">
import { Badge } from "$lib/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card"
import AppShell from "$lib/components/AppShell.svelte"
import { Download, Smartphone, TabletSmartphone } from "@lucide/svelte"

const apps = [
  { name: "iOS", href: "https://apps.apple.com/ru/app/happ-proxy-utility-plus/id6746188973" },
  { name: "Android", href: "https://play.google.com/store/apps/details?id=com.happproxy" },
  {
    name: "Windows",
    href: "https://github.com/Happ-proxy/happ-desktop/releases/latest/download/setup-Happ.x64.exe"
  },
  { name: "macOS", href: "https://apps.apple.com/ru/app/happ-proxy-utility-plus/id6746188973" },
  {
    name: "Linux",
    href: "https://github.com/Happ-proxy/happ-desktop/releases/latest/download/Happ.linux.x64.deb"
  }
]
</script>

<svelte:head>
  <title>Инструкция по установке</title>
</svelte:head>

<AppShell title="Подключение к wowblvck VPN" subtitle="Установите приложение и добавьте ссылку подписки на своём устройстве">
  <section class="space-y-6">
    <Card>
      <CardHeader>
        <div class="flex items-center gap-3">
          <TabletSmartphone class="size-6" />
          <div>
            <CardTitle>Как подключиться</CardTitle>
            <CardDescription>Несколько шагов, и wowblvck VPN будет готов к использованию.</CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent class="grid gap-3 md:grid-cols-5">
        <div class="rounded-lg border border-(--border) bg-(--muted) p-4 text-sm text-(--muted-foreground)">1. Скачайте клиент.</div>
        <div class="rounded-lg border border-(--border) bg-(--muted) p-4 text-sm text-(--muted-foreground)">2. Откройте приложение.</div>
        <div class="rounded-lg border border-(--border) bg-(--muted) p-4 text-sm text-(--muted-foreground)">3. Добавьте конфигурацию.</div>
        <div class="rounded-lg border border-(--border) bg-(--muted) p-4 text-sm text-(--muted-foreground)">4. Вставьте ссылку подписки.</div>
        <div class="rounded-lg border border-(--border) bg-(--muted) p-4 text-sm text-(--muted-foreground)">5. Подключитесь к VPN.</div>
      </CardContent>
    </Card>

    <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {#each apps as app}
        <a href={app.href} target="_blank" rel="noreferrer">
          <Card class="h-full transition hover:bg-(--muted)">
            <CardHeader>
              <div class="flex items-center justify-between gap-3">
                <Badge>{app.name}</Badge>
                <Smartphone class="size-4 text-(--muted-foreground)" />
              </div>
              <CardTitle class="text-2xl">{app.name}</CardTitle>
              <CardDescription>Открыть загрузку клиента Happ Proxy.</CardDescription>
            </CardHeader>
            <CardContent>
              <div class="flex items-center gap-2 text-sm text-(--foreground)">
                <Download class="size-4" />
                Скачать приложение
              </div>
            </CardContent>
          </Card>
        </a>
      {/each}
    </section>
  </section>
</AppShell>
