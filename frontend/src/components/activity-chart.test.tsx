import { fireEvent, render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router'
import { expect, it } from 'vitest'
import '../i18n'
import { ActivityChart } from './activity-chart'
import { formatDay } from '@/lib/date'

it('opens the selected local calendar day from keyboard navigation', () => {
  window.history.replaceState({}, '', '/accounts')
  render(
    <BrowserRouter>
      <ActivityChart
        accountID={7}
        statistics={{
          requests: 7,
          successes: 7,
          daily: [10, 11, 12, 13, 14, 15, 16].map((day) => ({
            day: `2026-10-${day}`,
            requests: 1,
            successes: 1,
          })),
        }}
      />
    </BrowserRouter>,
  )
  const chart = screen.getByRole('button')
  fireEvent.focus(chart)
  fireEvent.keyDown(chart, { key: 'ArrowLeft' })
  fireEvent.keyDown(chart, { key: 'Enter' })
  expect(window.location.pathname + window.location.search).toBe(
    '/runs?account_id=7&date=2026-10-15',
  )
  expect(formatDay('invalid-date', 'en')).toBe('invalid-date')
})
