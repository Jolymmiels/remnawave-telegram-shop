<script lang="ts">
import { Button } from "$lib/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card"
import { Input } from "$lib/components/ui/input"
import { Label } from "$lib/components/ui/label"
import { Separator } from "$lib/components/ui/separator"
import AppShell from "$lib/components/AppShell.svelte"
import { KeyRound, ShieldEllipsis } from "@lucide/svelte"

let { form } = $props<{
  form?: {
    mode?: "login" | "register"
    message?: string
    values?: {
      login?: string
    }
  }
}>()

let mode = $state<"login" | "register">("login")
let login = $state("")

$effect(() => {
  mode = form?.mode ?? "login"
  login = form?.values?.login ?? ""
})
</script>

<svelte:head>
  <title>Вход</title>
</svelte:head>

<AppShell title="Вход в wowblvck VPN" subtitle="Управляйте подпиской и устройствами в одном кабинете">
  <section class="mx-auto grid max-w-5xl gap-6 lg:grid-cols-[0.9fr_1.1fr]">
    <Card>
      <CardHeader>
        <div class="flex size-12 items-center justify-center rounded-lg border border-(--border) bg-(--muted)">
          <ShieldEllipsis class="size-5" />
        </div>
        <CardTitle class="text-3xl">Личный кабинет wowblvck VPN</CardTitle>
        <CardDescription>
          Войдите в аккаунт или создайте новый, чтобы сразу перейти к подписке и настройкам.
        </CardDescription>
      </CardHeader>
      <CardContent class="space-y-4 text-sm leading-6 text-(--muted-foreground)">
        <div>В кабинете доступны тарифы, статус подписки и ссылка для подключения устройств.</div>
        <Separator />
        <div>Если аккаунта ещё нет, зарегистрируйтесь на этой странице и начните пользоваться wowblvck VPN.</div>
      </CardContent>
    </Card>

    <Card>
      <CardHeader>
        <CardTitle>{mode === "login" ? "Вход" : "Регистрация"}</CardTitle>
        <CardDescription>
          {mode === "login"
            ? "Введите логин и пароль, чтобы открыть кабинет."
            : "Создайте аккаунт, чтобы оформить подписку и получить доступ."}
        </CardDescription>
      </CardHeader>
      <CardContent class="space-y-6">
        <div class="grid grid-cols-2 gap-3">
          <Button type="button" variant={mode === "login" ? "default" : "secondary"} onclick={() => (mode = "login")}>
            Вход
          </Button>
          <Button type="button" variant={mode === "register" ? "default" : "secondary"} onclick={() => (mode = "register")}>
            Регистрация
          </Button>
        </div>

        <form class="space-y-5" method="POST">
          <div class="space-y-2">
            <Label for="login">Логин</Label>
            <Input bind:value={login} id="login" name="login" autocomplete="username" required placeholder="Введите логин" />
          </div>

          <div class="space-y-2">
            <Label for="password">Пароль</Label>
            <Input
              id="password"
              name="password"
              type="password"
              autocomplete={mode === "login" ? "current-password" : "new-password"}
              required
              placeholder="••••••••"
            />
          </div>

          {#if form?.message}
            <div class="rounded-lg border border-(--border) bg-(--muted) px-4 py-3 text-sm text-(--foreground)">
              {form.message}
            </div>
          {/if}

          {#if mode === "login"}
            <Button type="submit" class="w-full" size="lg" formaction="?/login">
              <KeyRound class="size-4" />
              Войти в кабинет
            </Button>
          {:else}
            <Button type="submit" class="w-full" size="lg" formaction="?/register">
              <KeyRound class="size-4" />
              Создать аккаунт
            </Button>
          {/if}
        </form>
      </CardContent>
    </Card>
  </section>
</AppShell>
