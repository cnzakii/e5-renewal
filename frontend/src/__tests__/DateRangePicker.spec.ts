import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'

// Mock v-calendar — we test our wrapper logic, not VCalendar internals
vi.mock('v-calendar', () => ({
  DatePicker: {
    name: 'DatePicker',
    template: '<div class="vc-stub" />',
    props: ['modelValue', 'mode', 'locale', 'isDark', 'color', 'columns', 'rows', 'borderless', 'transparent', 'firstDayOfWeek'],
    emits: ['update:modelValue', 'dayclick'],
  },
}))
vi.mock('v-calendar/style.css', () => ({}))

import DateRangePicker from '../components/DateRangePicker.vue'

const mountOptions = {
  global: {
    stubs: {
      Teleport: true,
      Transition: { template: '<div><slot /></div>' },
    },
  },
}

describe('DateRangePicker', () => {
  it('renders trigger with placeholders when no value', () => {
    const wrapper = mount(DateRangePicker, {
      props: { modelValue: '', startPlaceholder: 'Start', endPlaceholder: 'End' },
      ...mountOptions,
    })
    expect(wrapper.text()).toContain('Start')
    expect(wrapper.text()).toContain('End')
    expect(wrapper.text()).toContain('~')
  })

  it('renders trigger with date values when provided', () => {
    const wrapper = mount(DateRangePicker, {
      props: { modelValue: ['2026-03-08', '2026-03-14'] as [string, string] },
      ...mountOptions,
    })
    expect(wrapper.text()).toContain('2026-03-08')
    expect(wrapper.text()).toContain('2026-03-14')
  })

  it('opens calendar panel on trigger click', async () => {
    const wrapper = mount(DateRangePicker, {
      props: { modelValue: '' },
      ...mountOptions,
    })
    await wrapper.find('.date-range-trigger').trigger('click')
    expect(wrapper.find('.calendar-panel').exists()).toBe(true)
  })

  it('shows clear button only when has value', () => {
    const empty = mount(DateRangePicker, {
      props: { modelValue: '' },
      ...mountOptions,
    })
    expect(empty.find('.clear-btn').exists()).toBe(false)

    const filled = mount(DateRangePicker, {
      props: { modelValue: ['2026-03-08', '2026-03-14'] as [string, string] },
      ...mountOptions,
    })
    expect(filled.find('.clear-btn').exists()).toBe(true)
  })

  it('emits empty string on clear', async () => {
    const wrapper = mount(DateRangePicker, {
      props: { modelValue: ['2026-03-08', '2026-03-14'] as [string, string] },
      ...mountOptions,
    })
    await wrapper.find('.clear-btn').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([''])
    expect(wrapper.emitted('change')![0]).toEqual([null])
  })

  it('closes panel on ESC', async () => {
    const wrapper = mount(DateRangePicker, {
      props: { modelValue: '' },
      ...mountOptions,
    })
    await wrapper.find('.date-range-trigger').trigger('click')
    expect(wrapper.find('.calendar-panel').exists()).toBe(true)

    // ESC on the overlay
    const overlay = wrapper.find('.fixed')
    await overlay.trigger('keydown', { key: 'Escape' })
    expect(wrapper.find('.calendar-panel').exists()).toBe(false)
  })

  it('hides clear button when clearable is false', () => {
    const wrapper = mount(DateRangePicker, {
      props: {
        modelValue: ['2026-03-08', '2026-03-14'] as [string, string],
        clearable: false,
      },
      ...mountOptions,
    })
    expect(wrapper.find('.clear-btn').exists()).toBe(false)
  })
})
