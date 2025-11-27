import React, { useState, useEffect } from 'react'
import {
  Stack,
  Title,
  Paper,
  TextInput,
  NumberInput,
  Switch,
  Button,
  Group,
  Divider,
  Text,
  LoadingOverlay,
  MultiSelect,
  Accordion,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { httpGet, httpPut } from '@/lib/http'

interface Squad {
  uuid: string
  name: string
}

interface Settings {
  [key: string]: string
}

const SettingsPage: React.FC = () => {
  const [settings, setSettings] = useState<Settings>({})
  const [squads, setSquads] = useState<Squad[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [settingsRes, squadsRes] = await Promise.all([
        httpGet<{ settings: Settings }>('/api/settings'),
        httpGet<{ squads: Squad[] }>('/api/squads'),
      ])
      setSettings(settingsRes.settings || {})
      setSquads(squadsRes.squads || [])
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

  const squadOptions = squads.map(s => ({
    value: s.uuid,
    label: s.name,
  }))

  const getSelectedSquads = (key: string): string[] => {
    const value = settings[key]
    if (!value) return []
    return value.split(',').filter(Boolean)
  }

  const setSelectedSquads = (key: string, values: string[]) => {
    updateSetting(key, values.join(','))
  }

  return (
    <Stack gap="md" pos="relative">
      <LoadingOverlay visible={loading} />
      
      <Group justify="space-between" align="center">
        <Title order={2}>Настройки</Title>
        <Button onClick={handleSave} loading={saving}>
          Сохранить
        </Button>
      </Group>

      <Accordion defaultValue={['prices', 'trial', 'payments']} multiple>
        <Accordion.Item value="prices">
          <Accordion.Control>Цены</Accordion.Control>
          <Accordion.Panel>
            <Paper p="md" withBorder>
              <Stack gap="sm">
                <Title order={5}>Обычные цены (в валюте)</Title>
                <Group grow>
                  <NumberInput
                    label="1 месяц"
                    value={Number(settings.price_1) || 0}
                    onChange={v => updateSetting('price_1', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="3 месяца"
                    value={Number(settings.price_3) || 0}
                    onChange={v => updateSetting('price_3', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="6 месяцев"
                    value={Number(settings.price_6) || 0}
                    onChange={v => updateSetting('price_6', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="12 месяцев"
                    value={Number(settings.price_12) || 0}
                    onChange={v => updateSetting('price_12', v || 0)}
                    min={0}
                  />
                </Group>

                <Divider my="sm" />

                <Title order={5}>Цены в Telegram Stars</Title>
                <Group grow>
                  <NumberInput
                    label="1 месяц"
                    value={Number(settings.stars_price_1) || 0}
                    onChange={v => updateSetting('stars_price_1', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="3 месяца"
                    value={Number(settings.stars_price_3) || 0}
                    onChange={v => updateSetting('stars_price_3', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="6 месяцев"
                    value={Number(settings.stars_price_6) || 0}
                    onChange={v => updateSetting('stars_price_6', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="12 месяцев"
                    value={Number(settings.stars_price_12) || 0}
                    onChange={v => updateSetting('stars_price_12', v || 0)}
                    min={0}
                  />
                </Group>
              </Stack>
            </Paper>
          </Accordion.Panel>
        </Accordion.Item>

        <Accordion.Item value="trial">
          <Accordion.Control>Триал</Accordion.Control>
          <Accordion.Panel>
            <Paper p="md" withBorder>
              <Stack gap="sm">
                <Group grow>
                  <NumberInput
                    label="Дни триала"
                    value={Number(settings.trial_days) || 0}
                    onChange={v => updateSetting('trial_days', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="Лимит трафика (ГБ)"
                    value={Number(settings.trial_traffic_limit) || 0}
                    onChange={v => updateSetting('trial_traffic_limit', v || 0)}
                    min={0}
                  />
                </Group>
                <TextInput
                  label="Тег триала в Remnawave"
                  value={settings.trial_remnawave_tag || ''}
                  onChange={e => updateSetting('trial_remnawave_tag', e.target.value)}
                />
                <MultiSelect
                  label="Internal Squads для триала"
                  data={squadOptions}
                  value={getSelectedSquads('trial_internal_squads')}
                  onChange={v => setSelectedSquads('trial_internal_squads', v)}
                  searchable
                  clearable
                />
                <TextInput
                  label="External Squad UUID для триала"
                  value={settings.trial_external_squad_uuid || ''}
                  onChange={e => updateSetting('trial_external_squad_uuid', e.target.value)}
                  placeholder="UUID или оставьте пустым"
                />
              </Stack>
            </Paper>
          </Accordion.Panel>
        </Accordion.Item>

        <Accordion.Item value="payments">
          <Accordion.Control>Платежи</Accordion.Control>
          <Accordion.Panel>
            <Paper p="md" withBorder>
              <Stack gap="md">
                <Title order={5}>Telegram Stars</Title>
                <Switch
                  label="Включить Telegram Stars"
                  checked={settings.telegram_stars_enabled === 'true'}
                  onChange={e => updateSetting('telegram_stars_enabled', e.currentTarget.checked)}
                />
                <Switch
                  label="Требовать оплаченную покупку для Stars"
                  checked={settings.require_paid_purchase_for_stars === 'true'}
                  onChange={e => updateSetting('require_paid_purchase_for_stars', e.currentTarget.checked)}
                />

                <Divider />

                <Title order={5}>CryptoPay</Title>
                <Switch
                  label="Включить CryptoPay"
                  checked={settings.crypto_pay_enabled === 'true'}
                  onChange={e => updateSetting('crypto_pay_enabled', e.currentTarget.checked)}
                />
                <TextInput
                  label="CryptoPay Token"
                  value={settings.crypto_pay_token || ''}
                  onChange={e => updateSetting('crypto_pay_token', e.target.value)}
                  type="password"
                />
                <TextInput
                  label="CryptoPay URL"
                  value={settings.crypto_pay_url || ''}
                  onChange={e => updateSetting('crypto_pay_url', e.target.value)}
                />

                <Divider />

                <Title order={5}>YooKassa</Title>
                <Switch
                  label="Включить YooKassa"
                  checked={settings.yookasa_enabled === 'true'}
                  onChange={e => updateSetting('yookasa_enabled', e.currentTarget.checked)}
                />
                <TextInput
                  label="Shop ID"
                  value={settings.yookasa_shop_id || ''}
                  onChange={e => updateSetting('yookasa_shop_id', e.target.value)}
                />
                <TextInput
                  label="Secret Key"
                  value={settings.yookasa_secret_key || ''}
                  onChange={e => updateSetting('yookasa_secret_key', e.target.value)}
                  type="password"
                />
                <TextInput
                  label="URL"
                  value={settings.yookasa_url || ''}
                  onChange={e => updateSetting('yookasa_url', e.target.value)}
                />
                <TextInput
                  label="Email"
                  value={settings.yookasa_email || ''}
                  onChange={e => updateSetting('yookasa_email', e.target.value)}
                />

                <Divider />

                <Title order={5}>Tribute</Title>
                <TextInput
                  label="Webhook URL"
                  value={settings.tribute_webhook_url || ''}
                  onChange={e => updateSetting('tribute_webhook_url', e.target.value)}
                />
                <TextInput
                  label="API Key"
                  value={settings.tribute_api_key || ''}
                  onChange={e => updateSetting('tribute_api_key', e.target.value)}
                  type="password"
                />
                <TextInput
                  label="Payment URL"
                  value={settings.tribute_payment_url || ''}
                  onChange={e => updateSetting('tribute_payment_url', e.target.value)}
                />
              </Stack>
            </Paper>
          </Accordion.Panel>
        </Accordion.Item>

        <Accordion.Item value="squads">
          <Accordion.Control>Squads</Accordion.Control>
          <Accordion.Panel>
            <Paper p="md" withBorder>
              <Stack gap="sm">
                <Text size="sm" c="dimmed">
                  Доступные сквады из Remnawave: {squads.map(s => s.name).join(', ') || 'нет'}
                </Text>
                <MultiSelect
                  label="Internal Squads для покупателей"
                  data={squadOptions}
                  value={getSelectedSquads('squad_uuids')}
                  onChange={v => setSelectedSquads('squad_uuids', v)}
                  searchable
                  clearable
                />
                <TextInput
                  label="External Squad UUID"
                  value={settings.external_squad_uuid || ''}
                  onChange={e => updateSetting('external_squad_uuid', e.target.value)}
                  placeholder="UUID или оставьте пустым"
                />
                <TextInput
                  label="Тег в Remnawave"
                  value={settings.remnawave_tag || ''}
                  onChange={e => updateSetting('remnawave_tag', e.target.value)}
                />
              </Stack>
            </Paper>
          </Accordion.Panel>
        </Accordion.Item>

        <Accordion.Item value="links">
          <Accordion.Control>Ссылки</Accordion.Control>
          <Accordion.Panel>
            <Paper p="md" withBorder>
              <Stack gap="sm">
                <TextInput
                  label="Mini App URL"
                  value={settings.mini_app_url || ''}
                  onChange={e => updateSetting('mini_app_url', e.target.value)}
                />
                <TextInput
                  label="Server Status URL"
                  value={settings.server_status_url || ''}
                  onChange={e => updateSetting('server_status_url', e.target.value)}
                />
                <TextInput
                  label="Support URL"
                  value={settings.support_url || ''}
                  onChange={e => updateSetting('support_url', e.target.value)}
                />
                <TextInput
                  label="Feedback URL"
                  value={settings.feedback_url || ''}
                  onChange={e => updateSetting('feedback_url', e.target.value)}
                />
                <TextInput
                  label="Channel URL"
                  value={settings.channel_url || ''}
                  onChange={e => updateSetting('channel_url', e.target.value)}
                />
              </Stack>
            </Paper>
          </Accordion.Panel>
        </Accordion.Item>

        <Accordion.Item value="other">
          <Accordion.Control>Прочее</Accordion.Control>
          <Accordion.Panel>
            <Paper p="md" withBorder>
              <Stack gap="sm">
                <Group grow>
                  <NumberInput
                    label="Реферальные дни"
                    value={Number(settings.referral_days) || 0}
                    onChange={v => updateSetting('referral_days', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="Лимит трафика (ГБ, 0 = без лимита)"
                    value={Number(settings.traffic_limit) || 0}
                    onChange={v => updateSetting('traffic_limit', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="Дней в месяце"
                    value={Number(settings.days_in_month) || 31}
                    onChange={v => updateSetting('days_in_month', v || 31)}
                    min={28}
                    max={31}
                  />
                </Group>
                <Switch
                  label="WebApp Link"
                  checked={settings.is_web_app_link === 'true'}
                  onChange={e => updateSetting('is_web_app_link', e.currentTarget.checked)}
                />
                <TextInput
                  label="Заблокированные Telegram ID (через запятую)"
                  value={settings.blocked_telegram_ids || ''}
                  onChange={e => updateSetting('blocked_telegram_ids', e.target.value)}
                  placeholder="123456789,987654321"
                />
              </Stack>
            </Paper>
          </Accordion.Panel>
        </Accordion.Item>
      </Accordion>

      <Group justify="flex-end">
        <Button onClick={handleSave} loading={saving} size="lg">
          Сохранить настройки
        </Button>
      </Group>
    </Stack>
  )
}

export default SettingsPage
