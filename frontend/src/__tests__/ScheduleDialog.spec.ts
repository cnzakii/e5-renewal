import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import ScheduleDialog from '../components/ScheduleDialog.vue'

const mountOptions = {
  global: {
    stubs: {
      Teleport: true,
    },
  },
}

const enabledSchedule = {
  enabled: true,
  paused: false,
  pause_reason: '',
  pause_threshold: 30,
  next_run_at: '2026-03-15T10:00:00Z',
  last_run_at: '2026-03-14T10:00:00Z',
}

const disabledSchedule = {
  enabled: false,
  paused: false,
  pause_reason: '',
  pause_threshold: 30,
  next_run_at: null,
  last_run_at: null,
}

const pausedSchedule = {
  enabled: true,
  paused: true,
  pause_reason: 'Health too low',
  pause_threshold: 20,
  next_run_at: null,
  last_run_at: '2026-03-13T08:00:00Z',
}

describe('ScheduleDialog', () => {
  it('displays enabled status when schedule is active', () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: true, accountId: 1, schedule: enabledSchedule },
      ...mountOptions,
    })
    // Uses i18n key 'accounts.schedule.enabled' -> Chinese default '已启用'
    expect(wrapper.text()).toContain('已启用')
  })

  it('displays disabled status when schedule is inactive', () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: true, accountId: 1, schedule: disabledSchedule },
      ...mountOptions,
    })
    expect(wrapper.text()).toContain('已关闭')
  })

  it('displays paused status and pause reason', () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: true, accountId: 1, schedule: pausedSchedule },
      ...mountOptions,
    })
    expect(wrapper.text()).toContain('已暂停')
    expect(wrapper.text()).toContain('Health too low')
  })

  it('shows resume button when schedule is paused', () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: true, accountId: 1, schedule: pausedSchedule },
      ...mountOptions,
    })
    // Resume text: '恢复'
    const resumeBtn = wrapper.findAll('button').find((b) => b.text().includes('恢复'))
    expect(resumeBtn).toBeTruthy()
  })

  it('emits resume event when resume button is clicked', async () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: true, accountId: 42, schedule: pausedSchedule },
      ...mountOptions,
    })
    const resumeBtn = wrapper.findAll('button').find((b) => b.text().includes('恢复'))
    await resumeBtn!.trigger('click')
    expect(wrapper.emitted('resume')).toBeTruthy()
    expect(wrapper.emitted('resume')![0]).toEqual([42])
  })

  it('does not show resume button when schedule is not paused', () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: true, accountId: 1, schedule: enabledSchedule },
      ...mountOptions,
    })
    const resumeBtn = wrapper.findAll('button').find((b) => b.text().includes('恢复'))
    expect(resumeBtn).toBeUndefined()
  })

  it('toggles enabled state when toggle button is clicked', async () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: true, accountId: 1, schedule: enabledSchedule },
      ...mountOptions,
    })
    // The toggle is a button with a rounded-full class
    const toggleBtn = wrapper.find('button[type="button"].rounded-full')
    expect(toggleBtn.exists()).toBe(true)

    // Initially enabled (from schedule), clicking should disable
    await toggleBtn.trigger('click')
    // After clicking, the toggle class should change (bg-gray means disabled)
    expect(toggleBtn.classes()).toContain('bg-gray-300')
  })

  it('emits save with correct data when save button is clicked', async () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: true, accountId: 5, schedule: enabledSchedule },
      ...mountOptions,
    })

    // Find save button (contains '保存' text)
    const saveBtn = wrapper.findAll('button').find((b) => b.text() === '保存')
    expect(saveBtn).toBeTruthy()

    await saveBtn!.trigger('click')
    expect(wrapper.emitted('save')).toBeTruthy()
    expect(wrapper.emitted('save')![0]).toEqual([
      5,
      { enabled: true, pause_threshold: 30 },
    ])
  })

  it('emits update:visible false when close button is clicked', async () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: true, accountId: 1, schedule: enabledSchedule },
      ...mountOptions,
    })

    // Close button is the X button in the header (first small button with svg)
    // or the cancel button in footer ('取消')
    const cancelBtn = wrapper.findAll('button').find((b) => b.text() === '取消')
    await cancelBtn!.trigger('click')
    expect(wrapper.emitted('update:visible')).toBeTruthy()
    expect(wrapper.emitted('update:visible')![0]).toEqual([false])
  })

  it('resets form to defaults when schedule is null after being opened', async () => {
    // Mount with visible=false first, then set visible=true so the watch fires
    const wrapper = mount(ScheduleDialog, {
      props: { visible: false, accountId: 1, schedule: null },
      ...mountOptions,
    })

    await wrapper.setProps({ visible: true })
    // When schedule is null, form keeps its default (enabled: true), so toggle shows bg-apple-blue
    const toggleBtn = wrapper.find('button[type="button"].rounded-full')
    expect(toggleBtn.classes()).toContain('bg-apple-blue')
  })

  it('does not render content when visible is false', () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: false, accountId: 1, schedule: enabledSchedule },
      ...mountOptions,
    })
    expect(wrapper.text()).not.toContain('运行中')
    expect(wrapper.text()).not.toContain('定时任务')
  })

  it('does not emit save when accountId is null', async () => {
    const wrapper = mount(ScheduleDialog, {
      props: { visible: true, accountId: null, schedule: enabledSchedule },
      ...mountOptions,
    })

    const saveBtn = wrapper.findAll('button').find((b) => b.text() === '保存')
    await saveBtn!.trigger('click')
    expect(wrapper.emitted('save')).toBeFalsy()
  })
})
