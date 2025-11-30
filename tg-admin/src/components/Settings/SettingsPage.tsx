import React, { useState, useEffect, useMemo } from 'react'
import { DndContext, closestCenter, DragEndEvent } from '@dnd-kit/core'
import { SortableContext, useSortable, rectSortingStrategy, arrayMove } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import {
  Stack,
  Title,
  Paper,
  TextInput,
  NumberInput,
  Switch,
  Button,
  Group,
  Text,
  LoadingOverlay,
  Select,
  Accordion,
  SimpleGrid,
  Box,
  ActionIcon,
  Badge,
  Modal,
  Divider,
  SegmentedControl,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { IconPlus, IconTrash, IconStar, IconEdit } from '@tabler/icons-react'
import { httpGet, httpPut, httpPost, httpDelete } from '@/lib/http'

interface Squad {
  uuid: string
  name: string
}

interface SquadsResponse {
  internal_squads: Squad[]
  external_squads: Squad[]
}

interface Settings {
  [key: string]: string
}

interface Plan {
  id: number
  name: string
  price_1: number
  price_3: number
  price_6: number
  price_12: number
  traffic_limit: number
  device_limit: number | null
  internal_squads: string
  external_squad_uuid: string
  remnawave_tag: string
  tribute_url: string
  is_active: boolean
  is_default: boolean
}

interface PlansResponse {
  plans: Plan[]
}

interface LinkButton {
  id: string
  label: string
  urlKey: string
}

const LINK_BUTTONS: LinkButton[] = [
  { id: 'server_status', label: 'Status', urlKey: 'server_status_url' },
  { id: 'support', label: 'Support', urlKey: 'support_url' },
  { id: 'feedback', label: 'Feedback', urlKey: 'feedback_url' },
  { id: 'channel', label: 'Channel', urlKey: 'channel_url' },
]

function SortableButton({ id, label, disabled }: { id: string; label: string; disabled: boolean }) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id })
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: disabled ? 0.4 : 1,
  }
  return (
    <Button
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      size="compact-xs"
      variant="light"
      fullWidth
      disabled={disabled}
    >
      {label}
    </Button>
  )
}

const defaultPlan: Omit<Plan, 'id' | 'is_default'> = {
  name: '',
  price_1: 0,
  price_3: 0,
  price_6: 0,
  price_12: 0,
  traffic_limit: 0,
  device_limit: null,
  internal_squads: '',
  external_squad_uuid: '',
  remnawave_tag: '',
  tribute_url: '',
  is_active: true,
}

const SettingsPage: React.FC = () => {
  const [settings, setSettings] = useState<Settings>({})
  const [internalSquads, setInternalSquads] = useState<Squad[]>([])
  const [externalSquads, setExternalSquads] = useState<Squad[]>([])
  const [plans, setPlans] = useState<Plan[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  
  const [planModalOpen, setPlanModalOpen] = useState(false)
  const [editingPlan, setEditingPlan] = useState<Plan | null>(null)
  const [planForm, setPlanForm] = useState<Omit<Plan, 'id' | 'is_default'>>(defaultPlan)
  const [savingPlan, setSavingPlan] = useState(false)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [settingsRes, squadsRes, plansRes] = await Promise.all([
        httpGet<{ settings: Settings }>('/api/settings'),
        httpGet<SquadsResponse>('/api/squads'),
        httpGet<PlansResponse>('/api/plans'),
      ])
      setSettings(settingsRes.settings || {})
      setInternalSquads(squadsRes.internal_squads || [])
      setExternalSquads(squadsRes.external_squads || [])
      setPlans(plansRes.plans || [])
    } catch (error) {
      notifications.show({
        title: 'Ошибка',
        message: 'Не удалось загрузить настройки',
        color: 'red',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      await httpPut('/api/settings', { settings })
      notifications.show({
        title: 'Успешно',
        message: 'Настройки сохранены',
        color: 'green',
      })
    } catch (error) {
      notifications.show({
        title: 'Ошибка',
        message: 'Не удалось сохранить настройки',
        color: 'red',
      })
    } finally {
      setSaving(false)
    }
  }

  const updateSetting = (key: string, value: string | number | boolean) => {
    setSettings(prev => ({
      ...prev,
      [key]: String(value),
    }))
  }

  const openCreatePlan = () => {
    setEditingPlan(null)
    setPlanForm(defaultPlan)
    setPlanModalOpen(true)
  }

  const openEditPlan = (plan: Plan) => {
    setEditingPlan(plan)
    setPlanForm({
      name: plan.name,
      price_1: plan.price_1,
      price_3: plan.price_3,
      price_6: plan.price_6,
      price_12: plan.price_12,
      traffic_limit: plan.traffic_limit,
      device_limit: plan.device_limit,
      internal_squads: plan.internal_squads || '',
      external_squad_uuid: plan.external_squad_uuid || '',
      remnawave_tag: plan.remnawave_tag || '',
      tribute_url: plan.tribute_url || '',
      is_active: plan.is_active,
    })
    setPlanModalOpen(true)
  }

  const handleSavePlan = async () => {
    if (!planForm.name.trim()) {
      notifications.show({ title: 'Ошибка', message: 'Введите название тарифа', color: 'red' })
      return
    }
    
    setSavingPlan(true)
    try {
      if (editingPlan) {
        await httpPut(`/api/plans/${editingPlan.id}`, planForm)
      } else {
        await httpPost('/api/plans', planForm)
      }
      await loadData()
      setPlanModalOpen(false)
      notifications.show({ title: 'Успешно', message: editingPlan ? 'Тариф обновлен' : 'Тариф создан', color: 'green' })
    } catch (error) {
      notifications.show({ title: 'Ошибка', message: 'Не удалось сохранить тариф', color: 'red' })
    } finally {
      setSavingPlan(false)
    }
  }

  const handleDeletePlan = async (plan: Plan) => {
    if (plan.is_default) {
      notifications.show({ title: 'Ошибка', message: 'Нельзя удалить тариф по умолчанию', color: 'red' })
      return
    }
    
    // Check for associated purchases
    try {
      const res = await httpGet<{ count: number }>(`/api/plans/${plan.id}/purchases`)
      if (res.count > 0) {
        const confirmed = confirm(
          `ВНИМАНИЕ! К тарифу "${plan.name}" привязано ${res.count} покупок.\n\n` +
          `При удалении тарифа эти покупки потеряют связь с тарифом (plan_id станет NULL).\n\n` +
          `Рекомендуется деактивировать тариф вместо удаления.\n\n` +
          `Вы уверены, что хотите удалить тариф?`
        )
        if (!confirmed) return
      } else {
        if (!confirm(`Удалить тариф "${plan.name}"?`)) return
      }
    } catch {
      if (!confirm(`Удалить тариф "${plan.name}"?`)) return
    }
    
    try {
      await httpDelete(`/api/plans/${plan.id}`)
      await loadData()
      notifications.show({ title: 'Успешно', message: 'Тариф удален', color: 'green' })
    } catch (error) {
      notifications.show({ title: 'Ошибка', message: 'Не удалось удалить тариф', color: 'red' })
    }
  }

  const handleSetDefaultPlan = async (plan: Plan) => {
    try {
      await httpPost(`/api/plans/${plan.id}/default`, {})
      await loadData()
      notifications.show({ title: 'Успешно', message: `"${plan.name}" установлен по умолчанию`, color: 'green' })
    } catch (error) {
      notifications.show({ title: 'Ошибка', message: 'Не удалось установить тариф по умолчанию', color: 'red' })
    }
  }

  const getStarsPrice = (rubPrice: number): number => {
    const rate = Number(settings.stars_exchange_rate) || 1.5
    return Math.round(rubPrice * rate)
  }

  const internalSquadOptions = internalSquads.map(s => ({
    value: s.uuid,
    label: s.name,
  }))

  const externalSquadOptions = [
    { value: '', label: 'Не выбран' },
    ...externalSquads.map(s => ({
      value: s.uuid,
      label: s.name,
    })),
  ]

  const getSelectedSquads = (key: string): string[] => {
    const value = settings[key]
    if (!value) return []
    return value.split(',').filter(Boolean)
  }

  const setSelectedSquads = (key: string, values: string[]) => {
    updateSetting(key, values.join(','))
  }

  const getPlanSquads = (squadsStr: string): string[] => {
    if (!squadsStr) return []
    return squadsStr.split(',').filter(Boolean)
  }

  const setPlanSquads = (values: string[]) => {
    setPlanForm(prev => ({ ...prev, internal_squads: values.join(',') }))
  }

  const FieldLabel: React.FC<{ children: React.ReactNode }> = ({ children }) => (
    <Text size="sm" fw={500} mb={4}>{children}</Text>
  )

  return (
    <Stack gap="md" pos="relative">
      <LoadingOverlay visible={loading} />
      
      <Title order={3}>Настройки</Title>

      <Accordion defaultValue={['plans']} multiple variant="separated">
        {/* Plans Section */}
        <Accordion.Item value="plans">
          <Accordion.Control>Тарифы</Accordion.Control>
          <Accordion.Panel>
            <Stack gap="md">
              <Group justify="space-between">
                <Group gap="xs">
                  <NumberInput
                    label="Курс Stars"
                    description="RUB × курс = Stars"
                    size="xs"
                    value={Number(settings.stars_exchange_rate) || 1.5}
                    onChange={v => updateSetting('stars_exchange_rate', v || 1.5)}
                    min={0.1}
                    step={0.1}
                    decimalScale={2}
                    w={140}
                  />
                </Group>
                <Button size="xs" leftSection={<IconPlus size={14} />} onClick={openCreatePlan}>
                  Добавить тариф
                </Button>
              </Group>

              {plans.length === 0 ? (
                <Text c="dimmed" ta="center" py="xl">Нет тарифов</Text>
              ) : (
                <SimpleGrid cols={{ base: 1, md: 2 }} spacing="sm">
                  {plans.map(plan => (
                    <Paper key={plan.id} p="sm" withBorder>
                      <Group justify="space-between" mb="xs">
                        <Group gap="xs">
                          <Text fw={600}>{plan.name}</Text>
                          {plan.is_default && <Badge size="xs" color="blue">По умолчанию</Badge>}
                          {!plan.is_active && <Badge size="xs" color="gray">Неактивен</Badge>}
                        </Group>
                        <Group gap={4}>
                          <ActionIcon variant="subtle" size="sm" onClick={() => openEditPlan(plan)}>
                            <IconEdit size={14} />
                          </ActionIcon>
                          <ActionIcon 
                            variant="subtle" 
                            size="sm" 
                            color="red" 
                            onClick={() => handleDeletePlan(plan)}
                            disabled={plan.is_default}
                          >
                            <IconTrash size={14} />
                          </ActionIcon>
                        </Group>
                      </Group>
                      
                      <SimpleGrid cols={4} spacing={4}>
                        <Box>
                          <Text size="xs" c="dimmed">1 мес</Text>
                          <Text size="sm">{plan.price_1} ₽</Text>
                          <Group gap={2}>
                            <IconStar size={10} color="orange" />
                            <Text size="xs" c="orange">{getStarsPrice(plan.price_1)}</Text>
                          </Group>
                        </Box>
                        <Box>
                          <Text size="xs" c="dimmed">3 мес</Text>
                          <Text size="sm">{plan.price_3} ₽</Text>
                          <Group gap={2}>
                            <IconStar size={10} color="orange" />
                            <Text size="xs" c="orange">{getStarsPrice(plan.price_3)}</Text>
                          </Group>
                        </Box>
                        <Box>
                          <Text size="xs" c="dimmed">6 мес</Text>
                          <Text size="sm">{plan.price_6} ₽</Text>
                          <Group gap={2}>
                            <IconStar size={10} color="orange" />
                            <Text size="xs" c="orange">{getStarsPrice(plan.price_6)}</Text>
                          </Group>
                        </Box>
                        <Box>
                          <Text size="xs" c="dimmed">12 мес</Text>
                          <Text size="sm">{plan.price_12} ₽</Text>
                          <Group gap={2}>
                            <IconStar size={10} color="orange" />
                            <Text size="xs" c="orange">{getStarsPrice(plan.price_12)}</Text>
                          </Group>
                        </Box>
                      </SimpleGrid>

                      <Divider my="xs" />
                      
                      <Stack gap={4}>
                        <Text size="xs" c="dimmed">
                          Трафик: {plan.traffic_limit > 0 ? `${plan.traffic_limit} ГБ` : '∞'} 
                          {plan.device_limit && ` • Устройств: ${plan.device_limit}`}
                        </Text>
                        {(plan.internal_squads || plan.external_squad_uuid) && (
                          <Text size="xs" c="dimmed">
                            Squads: {getPlanSquads(plan.internal_squads).length > 0 && `${getPlanSquads(plan.internal_squads).length} internal`}
                            {plan.external_squad_uuid && (getPlanSquads(plan.internal_squads).length > 0 ? ' + 1 external' : '1 external')}
                          </Text>
                        )}
                        {!plan.is_default && (
                          <Button size="compact-xs" variant="light" onClick={() => handleSetDefaultPlan(plan)} mt={4}>
                            Сделать по умолчанию
                          </Button>
                        )}
                      </Stack>
                    </Paper>
                  ))}
                </SimpleGrid>
              )}
            </Stack>
          </Accordion.Panel>
        </Accordion.Item>

        {/* Trial Section */}
        <Accordion.Item value="trial">
          <Accordion.Control>Триал</Accordion.Control>
          <Accordion.Panel>
            <Stack gap="sm">
              <Switch
                label="Включить триал"
                size="xs"
                checked={settings.trial_enabled === 'true'}
                onChange={e => updateSetting('trial_enabled', e.currentTarget.checked)}
              />
              <SimpleGrid cols={3} spacing="xs">
                <NumberInput
                  label="Дни"
                  size="xs"
                  value={Number(settings.trial_days) || 0}
                  onChange={v => updateSetting('trial_days', v || 0)}
                  min={0}
                />
                <NumberInput
                  label="Трафик (ГБ)"
                  size="xs"
                  value={Number(settings.trial_traffic_limit) || 0}
                  onChange={v => updateSetting('trial_traffic_limit', v || 0)}
                  min={0}
                />
                <NumberInput
                  label="Устройства"
                  size="xs"
                  value={Number(settings.trial_device_limit) || 0}
                  onChange={v => updateSetting('trial_device_limit', v || 0)}
                  min={0}
                />
              </SimpleGrid>
              
              <TextInput
                label="Тег в Remnawave"
                size="xs"
                value={settings.trial_remnawave_tag || ''}
                onChange={e => updateSetting('trial_remnawave_tag', e.target.value.toUpperCase())}
              />

              <Box>
                <FieldLabel>Internal Squads</FieldLabel>
                <Stack gap={4}>
                  {internalSquadOptions.map(opt => (
                    <Switch
                      key={opt.value}
                      label={opt.label}
                      size="xs"
                      checked={getSelectedSquads('trial_internal_squads').includes(opt.value)}
                      onChange={e => {
                        const current = getSelectedSquads('trial_internal_squads')
                        if (e.currentTarget.checked) {
                          setSelectedSquads('trial_internal_squads', [...current, opt.value])
                        } else {
                          setSelectedSquads('trial_internal_squads', current.filter(v => v !== opt.value))
                        }
                      }}
                    />
                  ))}
                </Stack>
              </Box>

              <Select
                label="External Squad"
                size="xs"
                data={externalSquadOptions}
                value={settings.trial_external_squad_uuid || ''}
                onChange={v => updateSetting('trial_external_squad_uuid', v || '')}
                clearable
              />
            </Stack>
          </Accordion.Panel>
        </Accordion.Item>

        {/* Payments Section */}
        <Accordion.Item value="payments">
          <Accordion.Control>Платежи</Accordion.Control>
          <Accordion.Panel>
            <Stack gap="md">
              {/* General Payment Settings */}
              <TextInput
                label="URL возврата после оплаты"
                description="Ссылка на бота, куда пользователь вернется после оплаты (например: https://t.me/your_bot)"
                size="xs"
                value={settings.payment_return_url || ''}
                onChange={e => updateSetting('payment_return_url', e.target.value)}
                placeholder="https://t.me/your_bot"
              />
              {/* Telegram Stars */}
              <Paper p="xs" withBorder>
                <Group justify="space-between" mb="xs">
                  <Text fw={600} size="sm">Telegram Stars</Text>
                  <Switch
                    size="xs"
                    checked={settings.telegram_stars_enabled === 'true'}
                    onChange={e => updateSetting('telegram_stars_enabled', e.currentTarget.checked)}
                  />
                </Group>
                <Switch
                  label="Требовать оплаченную покупку"
                  size="xs"
                  checked={settings.require_paid_purchase_for_stars === 'true'}
                  onChange={e => updateSetting('require_paid_purchase_for_stars', e.currentTarget.checked)}
                />
              </Paper>

              {/* CryptoPay */}
              <Paper p="xs" withBorder>
                <Group justify="space-between" mb="xs">
                  <Text fw={600} size="sm">CryptoPay</Text>
                  <Switch
                    size="xs"
                    checked={settings.crypto_pay_enabled === 'true'}
                    onChange={e => updateSetting('crypto_pay_enabled', e.currentTarget.checked)}
                  />
                </Group>
                <Stack gap="xs">
                  <TextInput
                    label="Token"
                    size="xs"
                    value={settings.crypto_pay_token || ''}
                    onChange={e => updateSetting('crypto_pay_token', e.target.value)}
                    type="password"
                  />
                  <TextInput
                    label="URL"
                    size="xs"
                    value={settings.crypto_pay_url || ''}
                    onChange={e => updateSetting('crypto_pay_url', e.target.value)}
                  />
                </Stack>
              </Paper>

              {/* YooKassa */}
              <Paper p="xs" withBorder>
                <Group justify="space-between" mb="xs">
                  <Text fw={600} size="sm">YooKassa</Text>
                  <Switch
                    size="xs"
                    checked={settings.yookasa_enabled === 'true'}
                    onChange={e => updateSetting('yookasa_enabled', e.currentTarget.checked)}
                  />
                </Group>
                <Stack gap="xs">
                  <SimpleGrid cols={2} spacing="xs">
                    <TextInput
                      label="Shop ID"
                      size="xs"
                      value={settings.yookasa_shop_id || ''}
                      onChange={e => updateSetting('yookasa_shop_id', e.target.value)}
                    />
                    <TextInput
                      label="Email"
                      size="xs"
                      value={settings.yookasa_email || ''}
                      onChange={e => updateSetting('yookasa_email', e.target.value)}
                    />
                  </SimpleGrid>
                  <TextInput
                    label="Secret Key"
                    size="xs"
                    value={settings.yookasa_secret_key || ''}
                    onChange={e => updateSetting('yookasa_secret_key', e.target.value)}
                    type="password"
                  />
                  <TextInput
                    label="URL"
                    size="xs"
                    value={settings.yookasa_url || ''}
                    onChange={e => updateSetting('yookasa_url', e.target.value)}
                  />
                </Stack>
              </Paper>

              {/* Tribute */}
              <Paper p="xs" withBorder>
                <Group justify="space-between" mb="xs">
                  <Text fw={600} size="sm">Tribute</Text>
                  <Switch
                    size="xs"
                    checked={settings.tribute_enabled === 'true'}
                    onChange={e => updateSetting('tribute_enabled', e.currentTarget.checked)}
                  />
                </Group>
                <Stack gap="xs">
                  <TextInput
                    label="Webhook URL"
                    size="xs"
                    value={settings.tribute_webhook_url || ''}
                    onChange={e => updateSetting('tribute_webhook_url', e.target.value)}
                  />
                  <TextInput
                    label="API Key"
                    size="xs"
                    value={settings.tribute_api_key || ''}
                    onChange={e => updateSetting('tribute_api_key', e.target.value)}
                    type="password"
                  />
                </Stack>
              </Paper>

              {/* Recurring Payments (Autopay) */}
              <Paper p="xs" withBorder>
                <Group justify="space-between" mb="xs">
                  <Text fw={600} size="sm">Автоплатежи (YooKassa)</Text>
                  <Switch
                    size="xs"
                    checked={settings.recurring_payments_enabled === 'true'}
                    onChange={e => updateSetting('recurring_payments_enabled', e.currentTarget.checked)}
                  />
                </Group>
                <Text size="xs" c="dimmed" mb="xs">
                  Автоматическое продление подписки для пользователей с сохраненным способом оплаты
                </Text>
                <Stack gap="xs">
                  <SimpleGrid cols={2} spacing="xs">
                    <NumberInput
                      label="Списание за (дней)"
                      description="За сколько дней до окончания списывать"
                      size="xs"
                      value={Number(settings.recurring_days_before) || 1}
                      onChange={v => updateSetting('recurring_days_before', v || 1)}
                      min={1}
                      max={30}
                      disabled={settings.recurring_payments_enabled !== 'true'}
                    />
                    <NumberInput
                      label="Уведомление за (дней)"
                      description="За сколько дней предупредить"
                      size="xs"
                      value={Number(settings.recurring_notify_days_before) || 3}
                      onChange={v => updateSetting('recurring_notify_days_before', v || 3)}
                      min={1}
                      max={30}
                      disabled={settings.recurring_payments_enabled !== 'true'}
                    />
                  </SimpleGrid>
                  <Text size="xs" c="dimmed">
                    Пользователи получат уведомление за {settings.recurring_notify_days_before || 3} дн. до окончания подписки. 
                    Автоплатеж будет выполнен за {settings.recurring_days_before || 1} дн. до окончания.
                  </Text>
                </Stack>
              </Paper>
            </Stack>
          </Accordion.Panel>
        </Accordion.Item>

        {/* Links Section */}
        <Accordion.Item value="links">
          <Accordion.Control>Ссылки</Accordion.Control>
          <Accordion.Panel>
            <Stack gap="xs">
              <TextInput
                label="Mini App URL"
                size="xs"
                value={settings.mini_app_url || ''}
                onChange={e => updateSetting('mini_app_url', e.target.value)}
              />
              <TextInput
                label="Server Status"
                size="xs"
                value={settings.server_status_url || ''}
                onChange={e => updateSetting('server_status_url', e.target.value)}
              />
              <TextInput
                label="Support"
                size="xs"
                value={settings.support_url || ''}
                onChange={e => updateSetting('support_url', e.target.value)}
              />
              <TextInput
                label="Feedback"
                size="xs"
                value={settings.feedback_url || ''}
                onChange={e => updateSetting('feedback_url', e.target.value)}
              />
              <TextInput
                label="Channel"
                size="xs"
                value={settings.channel_url || ''}
                onChange={e => updateSetting('channel_url', e.target.value)}
              />
              <Box>
                <Text size="xs" fw={500} mb={4}>Расположение кнопок (перетащите для изменения порядка)</Text>
                <SegmentedControl
                  size="xs"
                  fullWidth
                  data={[
                    { value: '4x1', label: '4x1' },
                    { value: '2x2', label: '2x2' },
                    { value: '1x4', label: '1x4' },
                  ]}
                  value={settings.link_buttons_layout || '4x1'}
                  onChange={v => updateSetting('link_buttons_layout', v)}
                />
                <DndContext
                  collisionDetection={closestCenter}
                  onDragEnd={(event: DragEndEvent) => {
                    const { active, over } = event
                    if (over && active.id !== over.id) {
                      const order = settings.link_buttons_order 
                        ? JSON.parse(settings.link_buttons_order) 
                        : LINK_BUTTONS.map(b => b.id)
                      const oldIndex = order.indexOf(active.id)
                      const newIndex = order.indexOf(over.id)
                      const newOrder = arrayMove(order, oldIndex, newIndex)
                      updateSetting('link_buttons_order', JSON.stringify(newOrder))
                    }
                  }}
                >
                  <SortableContext
                    items={settings.link_buttons_order 
                      ? JSON.parse(settings.link_buttons_order) 
                      : LINK_BUTTONS.map(b => b.id)}
                    strategy={rectSortingStrategy}
                  >
                    <SimpleGrid 
                      cols={settings.link_buttons_layout === '1x4' ? 4 : settings.link_buttons_layout === '2x2' ? 2 : 1} 
                      spacing={4} 
                      mt="xs"
                    >
                      {(settings.link_buttons_order 
                        ? JSON.parse(settings.link_buttons_order) 
                        : LINK_BUTTONS.map(b => b.id)
                      ).map((id: string) => {
                        const btn = LINK_BUTTONS.find(b => b.id === id)
                        if (!btn) return null
                        return (
                          <SortableButton
                            key={btn.id}
                            id={btn.id}
                            label={btn.label}
                            disabled={!settings[btn.urlKey]}
                          />
                        )
                      })}
                    </SimpleGrid>
                  </SortableContext>
                </DndContext>
              </Box>
            </Stack>
          </Accordion.Panel>
        </Accordion.Item>

        {/* Referral Section */}
        <Accordion.Item value="referral">
          <Accordion.Control>Реферальная программа</Accordion.Control>
          <Accordion.Panel>
            <Stack gap="md">
              <Switch
                label="Включить реферальную программу"
                size="xs"
                checked={settings.referral_enabled === 'true'}
                onChange={e => updateSetting('referral_enabled', e.currentTarget.checked)}
              />

              {/* Base bonuses */}
              <Paper p="xs" withBorder>
                <Text fw={600} size="sm" mb="xs">Базовые бонусы</Text>
                <SimpleGrid cols={2} spacing="xs" verticalSpacing="xs">
                  <NumberInput
                    label="Рефереру (дни)"
                    size="xs"
                    value={Number(settings.referral_bonus_days) || 3}
                    onChange={v => updateSetting('referral_bonus_days', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="Приглашенному (дни)"
                    size="xs"
                    value={Number(settings.referral_referee_bonus_days) || 0}
                    onChange={v => updateSetting('referral_referee_bonus_days', v || 0)}
                    min={0}
                  />
                </SimpleGrid>
              </Paper>

              {/* Tiers */}
              <Paper p="xs" withBorder>
                <Group justify="space-between" mb="xs">
                  <Text fw={600} size="sm">Уровневая система</Text>
                  <Switch
                    size="xs"
                    checked={settings.referral_tiers_enabled === 'true'}
                    onChange={e => updateSetting('referral_tiers_enabled', e.currentTarget.checked)}
                  />
                </Group>
                <Text size="xs" c="dimmed" mb="xs">
                  Увеличивайте бонус в зависимости от количества приглашенных
                </Text>
                <SimpleGrid cols={2} spacing="xs" verticalSpacing="xs">
                  <NumberInput
                    label="Ур. 1: от рефералов"
                    size="xs"
                    value={Number(settings.referral_tier1_threshold) || 5}
                    onChange={v => updateSetting('referral_tier1_threshold', v || 5)}
                    min={1}
                    disabled={settings.referral_tiers_enabled !== 'true'}
                  />
                  <NumberInput
                    label="Бонус (дни)"
                    size="xs"
                    value={Number(settings.referral_tier1_bonus) || 3}
                    onChange={v => updateSetting('referral_tier1_bonus', v || 3)}
                    min={0}
                    disabled={settings.referral_tiers_enabled !== 'true'}
                  />
                  <NumberInput
                    label="Ур. 2: от рефералов"
                    size="xs"
                    value={Number(settings.referral_tier2_threshold) || 15}
                    onChange={v => updateSetting('referral_tier2_threshold', v || 15)}
                    min={1}
                    disabled={settings.referral_tiers_enabled !== 'true'}
                  />
                  <NumberInput
                    label="Бонус (дни)"
                    size="xs"
                    value={Number(settings.referral_tier2_bonus) || 5}
                    onChange={v => updateSetting('referral_tier2_bonus', v || 5)}
                    min={0}
                    disabled={settings.referral_tiers_enabled !== 'true'}
                  />
                  <NumberInput
                    label="Ур. 3: от рефералов"
                    size="xs"
                    value={Number(settings.referral_tier3_threshold) || 30}
                    onChange={v => updateSetting('referral_tier3_threshold', v || 30)}
                    min={1}
                    disabled={settings.referral_tiers_enabled !== 'true'}
                  />
                  <NumberInput
                    label="Бонус (дни)"
                    size="xs"
                    value={Number(settings.referral_tier3_bonus) || 7}
                    onChange={v => updateSetting('referral_tier3_bonus', v || 7)}
                    min={0}
                    disabled={settings.referral_tiers_enabled !== 'true'}
                  />
                </SimpleGrid>
              </Paper>

              {/* Recurring bonuses */}
              <Paper p="xs" withBorder>
                <Group justify="space-between" mb="xs">
                  <Text fw={600} size="sm">Повторяющиеся бонусы</Text>
                  <Switch
                    size="xs"
                    checked={settings.referral_recurring_enabled === 'true'}
                    onChange={e => updateSetting('referral_recurring_enabled', e.currentTarget.checked)}
                  />
                </Group>
                <Text size="xs" c="dimmed" mb="xs">
                  Получайте бонус при каждой покупке приглашенного пользователя
                </Text>
                <NumberInput
                  label="Процент от покупки (%)"
                  description="% от дней купленной подписки"
                  size="xs"
                  value={Number(settings.referral_recurring_percent) || 10}
                  onChange={v => updateSetting('referral_recurring_percent', v || 10)}
                  min={1}
                  max={100}
                  disabled={settings.referral_recurring_enabled !== 'true'}
                />
              </Paper>
            </Stack>
          </Accordion.Panel>
        </Accordion.Item>

        {/* Other Section */}
        <Accordion.Item value="other">
          <Accordion.Control>Прочее</Accordion.Control>
          <Accordion.Panel>
            <Stack gap="xs">
              <NumberInput
                label="Дней в месяце"
                size="xs"
                value={Number(settings.days_in_month) || 31}
                onChange={v => updateSetting('days_in_month', v || 31)}
                min={28}
                max={31}
              />
              
              <Switch
                label="WebApp Link"
                size="xs"
                checked={settings.is_web_app_link === 'true'}
                onChange={e => updateSetting('is_web_app_link', e.currentTarget.checked)}
              />
              
              <TextInput
                label="Заблокированные Telegram ID"
                size="xs"
                value={settings.blocked_telegram_ids || ''}
                onChange={e => updateSetting('blocked_telegram_ids', e.target.value)}
                placeholder="123456789,987654321"
              />
            </Stack>
          </Accordion.Panel>
        </Accordion.Item>
      </Accordion>

      <Button onClick={handleSave} loading={saving} fullWidth>
        Сохранить настройки
      </Button>

      {/* Plan Modal */}
      <Modal
        opened={planModalOpen}
        onClose={() => setPlanModalOpen(false)}
        title={editingPlan ? 'Редактировать тариф' : 'Новый тариф'}
        size="md"
      >
        <Stack gap="sm">
          <TextInput
            label="Название"
            value={planForm.name}
            onChange={e => setPlanForm(prev => ({ ...prev, name: e.target.value }))}
            required
          />
          
          <Text size="sm" fw={500}>Цены (₽)</Text>
          <SimpleGrid cols={4} spacing="xs">
            <NumberInput
              label="1 мес"
              size="xs"
              value={planForm.price_1}
              onChange={v => setPlanForm(prev => ({ ...prev, price_1: Number(v) || 0 }))}
              min={0}
            />
            <NumberInput
              label="3 мес"
              size="xs"
              value={planForm.price_3}
              onChange={v => setPlanForm(prev => ({ ...prev, price_3: Number(v) || 0 }))}
              min={0}
            />
            <NumberInput
              label="6 мес"
              size="xs"
              value={planForm.price_6}
              onChange={v => setPlanForm(prev => ({ ...prev, price_6: Number(v) || 0 }))}
              min={0}
            />
            <NumberInput
              label="12 мес"
              size="xs"
              value={planForm.price_12}
              onChange={v => setPlanForm(prev => ({ ...prev, price_12: Number(v) || 0 }))}
              min={0}
            />
          </SimpleGrid>

          <SimpleGrid cols={2} spacing="xs">
            <NumberInput
              label="Лимит трафика (ГБ)"
              description="0 = безлимит"
              size="xs"
              value={planForm.traffic_limit}
              onChange={v => setPlanForm(prev => ({ ...prev, traffic_limit: Number(v) || 0 }))}
              min={0}
            />
            <NumberInput
              label="Лимит устройств"
              description="Пусто = без лимита"
              size="xs"
              value={planForm.device_limit ?? ''}
              onChange={v => setPlanForm(prev => ({ ...prev, device_limit: v ? Number(v) : null }))}
              min={1}
              allowDecimal={false}
            />
          </SimpleGrid>

          <Divider label="Remnawave" labelPosition="center" />

          <TextInput
            label="Тег в Remnawave"
            size="xs"
            value={planForm.remnawave_tag}
            onChange={e => setPlanForm(prev => ({ ...prev, remnawave_tag: e.target.value.toUpperCase() }))}
          />

          <Box>
            <FieldLabel>Internal Squads</FieldLabel>
            <Stack gap={4}>
              {internalSquadOptions.map(opt => (
                <Switch
                  key={opt.value}
                  label={opt.label}
                  size="xs"
                  checked={getPlanSquads(planForm.internal_squads).includes(opt.value)}
                  onChange={e => {
                    const current = getPlanSquads(planForm.internal_squads)
                    if (e.currentTarget.checked) {
                      setPlanSquads([...current, opt.value])
                    } else {
                      setPlanSquads(current.filter(v => v !== opt.value))
                    }
                  }}
                />
              ))}
              {internalSquadOptions.length === 0 && (
                <Text size="xs" c="dimmed">Нет доступных squads</Text>
              )}
            </Stack>
          </Box>

          <Select
            label="External Squad"
            size="xs"
            data={externalSquadOptions}
            value={planForm.external_squad_uuid}
            onChange={v => setPlanForm(prev => ({ ...prev, external_squad_uuid: v || '' }))}
            clearable
          />

          <Divider label="Tribute" labelPosition="center" />

          <TextInput
            label="Tribute URL"
            description="Ссылка на подписку в Tribute для этого тарифа"
            size="xs"
            value={planForm.tribute_url}
            onChange={e => setPlanForm(prev => ({ ...prev, tribute_url: e.target.value }))}
            placeholder="https://tribute.tg/..."
          />

          <Divider />

          <Switch
            label="Активен"
            checked={planForm.is_active}
            onChange={e => setPlanForm(prev => ({ ...prev, is_active: e.currentTarget.checked }))}
          />

          <Group justify="flex-end" mt="md">
            <Button variant="default" onClick={() => setPlanModalOpen(false)}>Отмена</Button>
            <Button onClick={handleSavePlan} loading={savingPlan}>
              {editingPlan ? 'Сохранить' : 'Создать'}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  )
}

export default SettingsPage
