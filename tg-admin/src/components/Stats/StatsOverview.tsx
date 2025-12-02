import React, { useEffect, useState } from 'react'
import { Stack, Grid, Paper, Text, Group, SimpleGrid, Loader, Center, SegmentedControl, Progress, Badge } from '@mantine/core'
import { IconUsers, IconUserCheck, IconUserX, IconCoin, IconReceipt, IconTrendingUp, IconGift, IconTicket, IconCreditCard, IconFlask, IconLanguage } from '@tabler/icons-react'
import { AreaChart, BarChart } from '@mantine/charts'
import { http } from '@/lib/http'
import { PaymentIcon, getPaymentLabel } from '@/components/PaymentIcons'

interface UserStats {
  total: number
  active: number
  expired: number
  blocked: number
  blocked_by_user: number
  new_today: number
  new_this_week: number
  new_this_month: number
}

interface RevenueStats {
  today: number
  this_week: number
  this_month: number
  all_time: number
  avg_check: number
}

interface PaymentStats {
  total_count: number
  today_count: number
  by_currency: { currency: string; count: number; amount: number }[]
  by_payment_type: { type: string; count: number; amount: number }[]
}

interface ReferralStats {
  total_referrals: number
  active_referrers: number
  bonus_days_granted: number
  conversion_rate: number
}

interface PromoStats {
  total_promos: number
  active_promos: number
  total_usages: number
  bonus_days_granted: number
}

interface PlanStats {
  plan_id: number
  plan_name: string
  count: number
  amount: number
  percent: number
}

interface PeriodStats {
  months: number
  count: number
  amount: number
  percent: number
}

interface AutopayStats {
  enabled_users: number
  total_with_method: number
}

interface TrialStats {
  total_used: number
  converted_to_paid: number
  conversion_rate: number
}

interface LanguageStat {
  language: string
  count: number
  percent: number
}

interface StatsOverview {
  users: UserStats
  revenue: RevenueStats
  payments: PaymentStats
  referrals: ReferralStats
  promos: PromoStats
  plans: PlanStats[]
  periods: PeriodStats[]
  autopay: AutopayStats
  trial: TrialStats
  languages: LanguageStat[]
}

interface DailyGrowth {
  date: string
  count: number
}

interface DailyRevenue {
  date: string
  amount: number
  count: number
}

const StatCard: React.FC<{
  title: string
  value: string | number
  subtitle?: string
  icon: React.ReactNode
  color?: string
}> = ({ title, value, subtitle, icon, color = 'blue' }) => (
  <Paper p="md" radius="md" withBorder>
    <Group justify="space-between" mb="xs">
      <Text size="sm" c="dimmed" fw={500}>{title}</Text>
      <Text c={color}>{icon}</Text>
    </Group>
    <Text size="xl" fw={700}>{value}</Text>
    {subtitle && <Text size="xs" c="dimmed">{subtitle}</Text>}
  </Paper>
)

const formatNumber = (n: number): string => {
  return new Intl.NumberFormat('ru-RU').format(n)
}

const formatCurrency = (n: number): string => {
  return new Intl.NumberFormat('ru-RU', { minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(n) + ' ₽'
}

const StatsOverviewPage: React.FC = () => {
  const [overview, setOverview] = useState<StatsOverview | null>(null)
  const [userGrowth, setUserGrowth] = useState<DailyGrowth[]>([])
  const [dailyRevenue, setDailyRevenue] = useState<DailyRevenue[]>([])
  const [loading, setLoading] = useState(true)
  const [chartPeriod, setChartPeriod] = useState('30')

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [overviewData, growthData, revenueData] = await Promise.all([
          http.get('/api/stats/overview'),
          http.get(`/api/stats/users/daily?days=${chartPeriod}`),
          http.get(`/api/stats/revenue/daily?days=${chartPeriod}`)
        ])
        setOverview(overviewData)
        setUserGrowth(growthData || [])
        setDailyRevenue(revenueData || [])
      } catch (error) {
        console.error('Failed to fetch stats:', error)
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [chartPeriod])

  if (loading) {
    return (
      <Center h={200}>
        <Loader size="lg" />
      </Center>
    )
  }

  if (!overview) {
    return <Text c="red">Не удалось загрузить статистику</Text>
  }

  const { users, revenue, payments, referrals, promos, plans, periods, autopay, trial, languages } = overview

  return (
    <Stack gap="md">
      {/* User Stats */}
      <Text size="lg" fw={600}>Пользователи</Text>
      <SimpleGrid cols={{ base: 2, sm: 4 }}>
        <StatCard
          title="Всего"
          value={formatNumber(users.total)}
          icon={<IconUsers size={20} />}
          color="blue"
        />
        <StatCard
          title="Активные"
          value={formatNumber(users.active)}
          subtitle={`${((users.active / users.total) * 100).toFixed(1)}%`}
          icon={<IconUserCheck size={20} />}
          color="green"
        />
        <StatCard
          title="Неактивные"
          value={formatNumber(users.expired)}
          icon={<IconUserX size={20} />}
          color="orange"
        />
        <StatCard
          title="Новые сегодня"
          value={formatNumber(users.new_today)}
          subtitle={`За неделю: ${users.new_this_week}`}
          icon={<IconTrendingUp size={20} />}
          color="teal"
        />
        <StatCard
          title="Заблокировали бота"
          value={formatNumber(users.blocked_by_user)}
          subtitle={`${((users.blocked_by_user / users.total) * 100).toFixed(1)}%`}
          icon={<IconUserX size={20} />}
          color="red"
        />
      </SimpleGrid>

      {/* Revenue Stats */}
      <Text size="lg" fw={600} mt="md">Доходы</Text>
      <SimpleGrid cols={{ base: 2, sm: 4 }}>
        <StatCard
          title="Сегодня"
          value={formatCurrency(revenue.today)}
          icon={<IconCoin size={20} />}
          color="yellow"
        />
        <StatCard
          title="За неделю"
          value={formatCurrency(revenue.this_week)}
          icon={<IconCoin size={20} />}
          color="yellow"
        />
        <StatCard
          title="За месяц"
          value={formatCurrency(revenue.this_month)}
          icon={<IconCoin size={20} />}
          color="yellow"
        />
        <StatCard
          title="Средний чек"
          value={formatCurrency(revenue.avg_check)}
          subtitle={`Всего: ${formatCurrency(revenue.all_time)}`}
          icon={<IconReceipt size={20} />}
          color="grape"
        />
      </SimpleGrid>

      {/* Charts */}
      <Group justify="space-between" mt="md">
        <Text size="lg" fw={600}>Графики</Text>
        <SegmentedControl
          size="xs"
          value={chartPeriod}
          onChange={setChartPeriod}
          data={[
            { value: '7', label: '7 дней' },
            { value: '30', label: '30 дней' },
            { value: '90', label: '90 дней' }
          ]}
        />
      </Group>

      <Grid>
        <Grid.Col span={{ base: 12, md: 6 }}>
          <Paper p="md" radius="md" withBorder>
            <Text size="sm" fw={500} mb="md">Новые пользователи</Text>
            {userGrowth.length > 0 ? (
              <AreaChart
                h={200}
                data={userGrowth}
                dataKey="date"
                series={[{ name: 'count', color: 'blue.6', label: 'Пользователи' }]}
                curveType="monotone"
                withDots={false}
              />
            ) : (
              <Text c="dimmed" ta="center">Нет данных</Text>
            )}
          </Paper>
        </Grid.Col>
        <Grid.Col span={{ base: 12, md: 6 }}>
          <Paper p="md" radius="md" withBorder>
            <Text size="sm" fw={500} mb="md">Доходы</Text>
            {dailyRevenue.length > 0 ? (
              <BarChart
                h={200}
                data={dailyRevenue}
                dataKey="date"
                series={[{ name: 'amount', color: 'yellow.6', label: 'Сумма' }]}
              />
            ) : (
              <Text c="dimmed" ta="center">Нет данных</Text>
            )}
          </Paper>
        </Grid.Col>
      </Grid>

      {/* Payment breakdown */}
      {payments.by_currency && payments.by_currency.length > 0 && (
        <>
          <Text size="lg" fw={600} mt="md">По валютам</Text>
          <SimpleGrid cols={{ base: 2, sm: 3 }}>
            {payments.by_currency.map((item) => (
              <Paper key={item.currency} p="md" radius="md" withBorder>
                <Text size="sm" c="dimmed">{item.currency.toUpperCase()}</Text>
                <Text size="lg" fw={600}>{formatCurrency(item.amount)}</Text>
                <Text size="xs" c="dimmed">{item.count} платежей</Text>
              </Paper>
            ))}
          </SimpleGrid>
        </>
      )}

      {payments.by_payment_type && payments.by_payment_type.length > 0 && (
        <>
          <Text size="lg" fw={600} mt="md">По способам оплаты</Text>
          <SimpleGrid cols={{ base: 2, sm: 4 }}>
            {payments.by_payment_type.map((item) => (
              <Paper key={item.type} p="md" radius="md" withBorder>
                <Group gap="xs" mb="xs">
                  <PaymentIcon type={item.type as 'yookassa' | 'crypto' | 'telegram' | 'tribute'} size={20} />
                  <Text size="sm" c="dimmed">{getPaymentLabel(item.type as 'yookassa' | 'crypto' | 'telegram' | 'tribute')}</Text>
                </Group>
                <Text size="lg" fw={600}>{formatCurrency(item.amount)}</Text>
                <Text size="xs" c="dimmed">{item.count} платежей</Text>
              </Paper>
            ))}
          </SimpleGrid>
        </>
      )}

      {/* Referral Stats */}
      {referrals && (
        <>
          <Text size="lg" fw={600} mt="md">Реферальная программа</Text>
          <SimpleGrid cols={{ base: 2, sm: 4 }}>
            <StatCard
              title="Всего рефералов"
              value={formatNumber(referrals.total_referrals)}
              icon={<IconGift size={20} />}
              color="violet"
            />
            <StatCard
              title="Активных рефереров"
              value={formatNumber(referrals.active_referrers)}
              icon={<IconUsers size={20} />}
              color="violet"
            />
            <StatCard
              title="Бонусных дней выдано"
              value={formatNumber(referrals.bonus_days_granted)}
              icon={<IconGift size={20} />}
              color="violet"
            />
            <StatCard
              title="Конверсия"
              value={`${referrals.conversion_rate.toFixed(1)}%`}
              subtitle="рефералов в платящих"
              icon={<IconTrendingUp size={20} />}
              color="violet"
            />
          </SimpleGrid>
        </>
      )}

      {/* Promo Stats */}
      {promos && (
        <>
          <Text size="lg" fw={600} mt="md">Промокоды</Text>
          <SimpleGrid cols={{ base: 2, sm: 4 }}>
            <StatCard
              title="Всего промокодов"
              value={formatNumber(promos.total_promos)}
              subtitle={`Активных: ${promos.active_promos}`}
              icon={<IconTicket size={20} />}
              color="pink"
            />
            <StatCard
              title="Использований"
              value={formatNumber(promos.total_usages)}
              icon={<IconTicket size={20} />}
              color="pink"
            />
            <StatCard
              title="Бонусных дней"
              value={formatNumber(promos.bonus_days_granted)}
              subtitle="выдано через промокоды"
              icon={<IconGift size={20} />}
              color="pink"
            />
          </SimpleGrid>
        </>
      )}

      {/* Plans Stats */}
      {plans && plans.length > 0 && (
        <>
          <Text size="lg" fw={600} mt="md">По тарифам</Text>
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            {plans.map((plan) => (
              <Paper key={plan.plan_id} p="md" radius="md" withBorder>
                <Group justify="space-between" mb="xs">
                  <Text fw={500}>{plan.plan_name}</Text>
                  <Badge color="blue">{plan.percent.toFixed(1)}%</Badge>
                </Group>
                <Text size="lg" fw={600}>{formatCurrency(plan.amount)}</Text>
                <Text size="xs" c="dimmed">{plan.count} покупок</Text>
                <Progress value={plan.percent} size="sm" mt="xs" />
              </Paper>
            ))}
          </SimpleGrid>
        </>
      )}

      {/* Period Stats */}
      {periods && periods.length > 0 && (
        <>
          <Text size="lg" fw={600} mt="md">По периодам подписки</Text>
          <SimpleGrid cols={{ base: 2, sm: 4 }}>
            {periods.map((period) => (
              <Paper key={period.months} p="md" radius="md" withBorder>
                <Group justify="space-between" mb="xs">
                  <Text fw={500}>{period.months} мес</Text>
                  <Badge color="cyan">{period.percent.toFixed(1)}%</Badge>
                </Group>
                <Text size="lg" fw={600}>{formatCurrency(period.amount)}</Text>
                <Text size="xs" c="dimmed">{period.count} покупок</Text>
              </Paper>
            ))}
          </SimpleGrid>
        </>
      )}

      {/* Autopay & Trial Stats */}
      <Grid mt="md">
        {autopay && (
          <Grid.Col span={{ base: 12, sm: 6 }}>
            <Paper p="md" radius="md" withBorder>
              <Group gap="xs" mb="md">
                <IconCreditCard size={20} />
                <Text size="lg" fw={600}>Автоплатежи</Text>
              </Group>
              <SimpleGrid cols={2}>
                <div>
                  <Text size="xs" c="dimmed">С автоплатежом</Text>
                  <Text size="lg" fw={600}>{formatNumber(autopay.enabled_users)}</Text>
                </div>
                <div>
                  <Text size="xs" c="dimmed">С методом оплаты</Text>
                  <Text size="lg" fw={600}>{formatNumber(autopay.total_with_method)}</Text>
                </div>
              </SimpleGrid>
            </Paper>
          </Grid.Col>
        )}

        {trial && (
          <Grid.Col span={{ base: 12, sm: 6 }}>
            <Paper p="md" radius="md" withBorder>
              <Group gap="xs" mb="md">
                <IconFlask size={20} />
                <Text size="lg" fw={600}>Триал</Text>
              </Group>
              <SimpleGrid cols={3}>
                <div>
                  <Text size="xs" c="dimmed">Использовали</Text>
                  <Text size="lg" fw={600}>{formatNumber(trial.total_used)}</Text>
                </div>
                <div>
                  <Text size="xs" c="dimmed">Оплатили</Text>
                  <Text size="lg" fw={600}>{formatNumber(trial.converted_to_paid)}</Text>
                </div>
                <div>
                  <Text size="xs" c="dimmed">Конверсия</Text>
                  <Text size="lg" fw={600} c="green">{trial.conversion_rate.toFixed(1)}%</Text>
                </div>
              </SimpleGrid>
            </Paper>
          </Grid.Col>
        )}
      </Grid>

      {/* Language Stats */}
      {languages && languages.length > 0 && (
        <>
          <Text size="lg" fw={600} mt="md">По языкам</Text>
          <SimpleGrid cols={{ base: 2, sm: 4 }}>
            {languages.map((lang) => (
              <Paper key={lang.language} p="md" radius="md" withBorder>
                <Group justify="space-between" mb="xs">
                  <Group gap="xs">
                    <IconLanguage size={16} />
                    <Text fw={500}>{lang.language.toUpperCase()}</Text>
                  </Group>
                  <Badge color="gray">{lang.percent.toFixed(1)}%</Badge>
                </Group>
                <Text size="lg" fw={600}>{formatNumber(lang.count)}</Text>
              </Paper>
            ))}
          </SimpleGrid>
        </>
      )}
    </Stack>
  )
}

export default StatsOverviewPage
