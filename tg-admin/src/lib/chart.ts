type Point = { month: string; amount: number }

export function renderGrowthChart(canvas: HTMLCanvasElement, data: Point[]) {
  const ctx = canvas.getContext('2d')!
  const dpr = window.devicePixelRatio || 1
  const rect = canvas.getBoundingClientRect()
  canvas.width = rect.width * dpr
  canvas.height = rect.height * dpr
  ctx.scale(dpr, dpr)

  const width = rect.width, height = rect.height
  const padding = 50
  const chartWidth = width - 2 * padding
  const chartHeight = height - 2 * padding

  ctx.clearRect(0,0,width,height)

  if (!data?.length) {
    ctx.fillStyle = getCss('--tg-hint-color')
    ctx.font = '16px Arial'
    ctx.textAlign = 'center'
    ctx.fillText('Нет данных для отображения', width/2, height/2)
    return
  }

  const labels = data.map(d => d.month)
  const values = data.map(d => d.amount)
  const maxValue = Math.max(...values, 1)
  const minValue = Math.min(...values, 0)
  const range = maxValue - minValue || 1
  const stepX = chartWidth / (labels.length - 1)
  const stepY = chartHeight / range

  const textColor = getCss('--tg-text-color')
  const hintColor = getCss('--tg-hint-color')
  const buttonColor = getCss('--tg-button-color')

  // Оси
  ctx.strokeStyle = hintColor
  ctx.lineWidth = 1
  ctx.beginPath()
  ctx.moveTo(padding, padding)
  ctx.lineTo(padding, height - padding)
  ctx.lineTo(width - padding, height - padding)
  ctx.stroke()

  // Сетка + подписи Y
  ctx.textAlign = 'right'
  ctx.font = '10px Arial'
  for (let i = 0; i <= 5; i++) {
    const yValue = minValue + (range * i / 5)
    const yPos = height - padding - ((yValue - minValue) * stepY)
    ctx.beginPath()
    ctx.strokeStyle = hintColor + '40'
    ctx.moveTo(padding, yPos)
    ctx.lineTo(width - padding, yPos)
    ctx.stroke()
    ctx.fillStyle = hintColor
    ctx.fillText(formatNumber(yValue), padding - 10, yPos + 3)
  }

  // Линия
  ctx.beginPath()
  ctx.strokeStyle = buttonColor
  ctx.lineWidth = 2
  ctx.moveTo(padding, height - padding - ((values[0] - minValue) * stepY))
  for (let i = 1; i < values.length; i++) {
    ctx.lineTo(padding + i * stepX, height - padding - ((values[i] - minValue) * stepY))
  }
  ctx.stroke()

  // Точки
  ctx.fillStyle = buttonColor
  for (let i = 0; i < values.length; i++) {
    const x = padding + i * stepX
    const y = height - padding - ((values[i] - minValue) * stepY)
    ctx.beginPath(); ctx.arc(x, y, 4, 0, Math.PI*2); ctx.fill()
  }

  // Подписи X
  ctx.textAlign = 'center'
  ctx.fillStyle = hintColor
  const maxLabelWidth = chartWidth / labels.length
  const every = Math.ceil(60 / maxLabelWidth)
  for (let i = 0; i < labels.length; i++) {
    if (i % every === 0 || i === labels.length - 1) {
      const x = padding + i * stepX
      const y = height - padding + 20
      ctx.fillText(labels[i], x, y)
    }
  }
}

function getCss(name: string) { return getComputedStyle(document.documentElement).getPropertyValue(name).trim() }
function formatNumber(n: number) { return new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 0 }).format(n) }
