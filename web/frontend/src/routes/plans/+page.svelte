<script lang="ts">
import { ArrowRight, CreditCard } from "@lucide/svelte";
import AppShell from "$lib/components/AppShell.svelte";
import { Badge } from "$lib/components/ui/badge";
import { Button } from "$lib/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardFooter,
	CardHeader,
	CardTitle,
} from "$lib/components/ui/card";

let { data, form } = $props<{
	data: {
		plans: Array<{ months: number; price: number; currency: string }>;
		authenticated: boolean;
	};
	form?: {
		message?: string;
	};
}>();
</script>

<svelte:head>
  <title>Тарифы</title>
</svelte:head>

<AppShell title="Тарифы wowblvck VPN" subtitle="Выберите удобный срок подписки и оформите доступ">
  <section class="space-y-6">
    {#if form?.message}
      <div class="rounded-lg border border-(--border) bg-(--muted) px-4 py-3 text-sm text-(--foreground)">
        {form.message}
      </div>
    {/if}

    <div class="flex flex-wrap items-center gap-3">
      <Badge variant="success">понятные цены</Badge>
      <Badge>instant access</Badge>
      <Badge variant="default">{data.authenticated ? "готово к оформлению" : "требуется вход"}</Badge>
    </div>

    <section class="grid gap-5 md:grid-cols-2 xl:grid-cols-4">
      {#each data.plans as plan}
        <Card class="flex h-full flex-col">
          <CardHeader>
            <CardTitle class="text-4xl">{plan.months} мес.</CardTitle>
          </CardHeader>
          <CardContent class="flex-1">
            <div class="flex items-end gap-2">
              <span class="text-3xl font-semibold text-(--foreground)">{plan.price}</span>
              <span class="pb-1 text-sm text-(--muted-foreground)">{plan.currency}</span>
            </div>
          </CardContent>
          <CardFooter class="pt-0">
            <form class="w-full" method="POST" action="?/checkout">
              <input type="hidden" name="months" value={plan.months} />
              <Button type="submit" class="w-full">
                <CreditCard class="size-4" />
                {data.authenticated ? "Оплатить" : "Войти и продолжить"}
                <ArrowRight class="size-4" />
              </Button>
            </form>
          </CardFooter>
        </Card>
      {/each}
    </section>
  </section>
</AppShell>
