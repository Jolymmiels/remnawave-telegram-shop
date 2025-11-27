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
  Select,
  Accordion,
  SimpleGrid,
  Box,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { httpGet, httpPut } from '@/lib/http'

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

const SettingsPage: React.FC = () => {
  const [settings, setSettings] = useState<Settings>({})
  const [internalSquads, setInternalSquads] = useState<Squad[]>([])
  const [externalSquads, setExternalSquads] = useState<Squad[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [settingsRes, squadsRes] = await Promise.all([
        httpGet<{ settings: Settings }>('/api/settings'),
        httpGet<SquadsResponse>('/api/squads'),
      ])
      setSettings(settingsRes.settings || {})
      setInternalSquads(squadsRes.internal_squads || [])
      setExternalSquads(squadsRes.external_squads || [])
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

  const FieldLabel: React.FC<{ children: React.ReactNode }> = ({ children }) => (
    <Text size="sm" fw={500} mb={4}>{children}</Text>
  )

  return (
    <Stack gap="md" pos="relative">
      <LoadingOverlay visible={loading} />
      
      <Group justify="space-between" align="center">
        <Title order={3}>Настройки</Title>
        <Button onClick={handleSave} loading={saving} size="sm">
          Сохранить
        </Button>
      </Group>

      <Accordion defaultValue={['prices']} multiple variant="separated">
        {/* Prices Section */}
        <Accordion.Item value="prices">
          <Accordion.Control>Цены</Accordion.Control>
          <Accordion.Panel>
            <Stack gap="md">
              <Box>
                <Text fw={600} size="sm" mb="xs">Обычные цены</Text>
                <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="xs">
                  <NumberInput
                    label="1 мес"
                    size="xs"
                    value={Number(settings.price_1) || 0}
                    onChange={v => updateSetting('price_1', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="3 мес"
                    size="xs"
                    value={Number(settings.price_3) || 0}
                    onChange={v => updateSetting('price_3', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="6 мес"
                    size="xs"
                    value={Number(settings.price_6) || 0}
                    onChange={v => updateSetting('price_6', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="12 мес"
                    size="xs"
                    value={Number(settings.price_12) || 0}
                    onChange={v => updateSetting('price_12', v || 0)}
                    min={0}
                  />
                </SimpleGrid>
              </Box>

              <Divider />

              <Box>
                <Text fw={600} size="sm" mb="xs">Telegram Stars</Text>
                <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="xs">
                  <NumberInput
                    label="1 мес"
                    size="xs"
                    value={Number(settings.stars_price_1) || 0}
                    onChange={v => updateSetting('stars_price_1', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="3 мес"
                    size="xs"
                    value={Number(settings.stars_price_3) || 0}
                    onChange={v => updateSetting('stars_price_3', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="6 мес"
                    size="xs"
                    value={Number(settings.stars_price_6) || 0}
                    onChange={v => updateSetting('stars_price_6', v || 0)}
                    min={0}
                  />
                  <NumberInput
                    label="12 мес"
                    size="xs"
                    value={Number(settings.stars_price_12) || 0}
                    onChange={v => updateSetting('stars_price_12', v || 0)}
                    min={0}
                  />
                </SimpleGrid>
              </Box>
            </Stack>
          </Accordion.Panel>
        </Accordion.Item>

        {/* Trial Section */}
        <Accordion.Item value="trial">
          <Accordion.Control>Триал</Accordion.Control>
          <Accordion.Panel>
            <Stack gap="sm">
              <SimpleGrid cols={{ base: 2 }} spacing="xs">
                <NumberInput
                  label="Дни триала"
                  size="xs"
                  value={Number(settings.trial_days) || 0}
                  onChange={v => updateSetting('trial_days', v || 0)}
                  min={0}
                />
                <NumberInput
                  label="Лимит трафика (ГБ)"
                  size="xs"
                  value={Number(settings.trial_traffic_limit) || 0}
                  onChange={v => updateSetting('trial_traffic_limit', v || 0)}
                  min={0}
                />
              </SimpleGrid>
              
              <TextInput
                label="Тег в Remnawave"
                size="xs"
                value={settings.trial_remnawave_tag || ''}
                onChange={e => updateSetting('trial_remnawave_tag', e.target.value)}
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

        {/* Squads Section */}
        <Accordion.Item value="squads">
          <Accordion.Control>Squads (покупатели)</Accordion.Control>
          <Accordion.Panel>
            <Stack gap="sm">
              <TextInput
                label="Тег в Remnawave"
                size="xs"
                value={settings.remnawave_tag || ''}
                onChange={e => updateSetting('remnawave_tag', e.target.value)}
              />

              <Box>
                <FieldLabel>Internal Squads</FieldLabel>
                <Stack gap={4}>
                  {internalSquadOptions.map(opt => (
                    <Switch
                      key={opt.value}
                      label={opt.label}
                      size="xs"
                      checked={getSelectedSquads('squad_uuids').includes(opt.value)}
                      onChange={e => {
                        const current = getSelectedSquads('squad_uuids')
                        if (e.currentTarget.checked) {
                          setSelectedSquads('squad_uuids', [...current, opt.value])
                        } else {
                          setSelectedSquads('squad_uuids', current.filter(v => v !== opt.value))
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
                value={settings.external_squad_uuid || ''}
                onChange={v => updateSetting('external_squad_uuid', v || '')}
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
                <Text fw={600} size="sm" mb="xs">Tribute</Text>
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
                  <TextInput
                    label="Payment URL"
                    size="xs"
                    value={settings.tribute_payment_url || ''}
                    onChange={e => updateSetting('tribute_payment_url', e.target.value)}
                  />
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
            </Stack>
          </Accordion.Panel>
        </Accordion.Item>

        {/* Other Section */}
        <Accordion.Item value="other">
          <Accordion.Control>Прочее</Accordion.Control>
          <Accordion.Panel>
            <Stack gap="xs">
              <SimpleGrid cols={{ base: 2, sm: 3 }} spacing="xs">
                <NumberInput
                  label="Реферальные дни"
                  size="xs"
                  value={Number(settings.referral_days) || 0}
                  onChange={v => updateSetting('referral_days', v || 0)}
                  min={0}
                />
                <NumberInput
                  label="Лимит трафика (ГБ)"
                  size="xs"
                  value={Number(settings.traffic_limit) || 0}
                  onChange={v => updateSetting('traffic_limit', v || 0)}
                  min={0}
                />
                <NumberInput
                  label="Дней в месяце"
                  size="xs"
                  value={Number(settings.days_in_month) || 31}
                  onChange={v => updateSetting('days_in_month', v || 31)}
                  min={28}
                  max={31}
                />
              </SimpleGrid>
              
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
    </Stack>
  )
}

export default SettingsPage
