export function formatDate(value: string | null, language: string) {
  return value
    ? new Intl.DateTimeFormat(language === 'zh' ? 'zh-CN' : 'en', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }).format(new Date(value))
    : '—'
}

// The API returns calendar dates already grouped in the browser's time zone.
export function formatDay(day: string, language: string) {
  const value = new Date(day + 'T00:00:00Z')
  if (Number.isNaN(value.getTime())) {
    return day
  }
  return new Intl.DateTimeFormat(language, {
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC',
  }).format(value)
}

export function timeZoneLabel(language: string) {
  const format = new Intl.DateTimeFormat(language, {
    timeZoneName: 'shortOffset',
  })
  return (
    format
      .formatToParts(new Date())
      .find((part) => part.type === 'timeZoneName')?.value ||
    format.resolvedOptions().timeZone
  )
}
