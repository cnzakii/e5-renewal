import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import TriggerResultDialog from '../components/TriggerResultDialog.vue'

const mockResult = {
  task_log: {
    id: 1,
    trigger_type: 'manual',
    total_endpoints: 3,
    success_count: 2,
    fail_count: 1,
    started_at: '2026-03-09T10:00:00Z',
    finished_at: '2026-03-09T10:00:05Z',
  },
  endpoints: [
    {
      id: 1,
      endpoint_name: 'me/drive/root',
      scope: 'Files.Read',
      http_status: 200,
      success: true,
      error_message: '',
      response_body: '',
      executed_at: '2026-03-09T10:00:01Z',
    },
    {
      id: 2,
      endpoint_name: 'me/messages',
      scope: 'Mail.Read',
      http_status: 200,
      success: true,
      error_message: '',
      response_body: '',
      executed_at: '2026-03-09T10:00:02Z',
    },
    {
      id: 3,
      endpoint_name: 'me/presence',
      scope: 'Presence.Read',
      http_status: 401,
      success: false,
      error_message: 'HTTP 401',
      response_body: '{"error":{"code":"InvalidAuthenticationToken"}}',
      executed_at: '2026-03-09T10:00:03Z',
    },
  ],
}

const mountOptions = {
  global: {
    stubs: {
      // router-link is used in the footer; stub it to avoid needing a router
      RouterLink: { template: '<a><slot /></a>' },
      // Teleport renders outside the component; attach to document.body
      Teleport: true,
    },
  },
}

describe('TriggerResultDialog', () => {
  it('shows scope as primary and endpoint name as secondary', () => {
    const wrapper = mount(TriggerResultDialog, {
      props: { visible: true, result: mockResult },
      ...mountOptions,
    })
    // Scope names should be visible as primary text
    expect(wrapper.text()).toContain('Files.Read')
    expect(wrapper.text()).toContain('Presence.Read')
    // Endpoint names should still be present as secondary text
    expect(wrapper.text()).toContain('me/drive/root')
    expect(wrapper.text()).toContain('me/presence')
  })

  it('shows success and fail counts', () => {
    const wrapper = mount(TriggerResultDialog, {
      props: { visible: true, result: mockResult },
      ...mountOptions,
    })
    // success_count=2 and fail_count=1 appear in the summary "{success}/{total} succeeded"
    expect(wrapper.text()).toContain('2')
    expect(wrapper.text()).toContain('1')
  })

  it('shows status code for failed endpoints without error details', () => {
    const wrapper = mount(TriggerResultDialog, {
      props: { visible: true, result: mockResult },
      ...mountOptions,
    })
    // Status code should be visible
    expect(wrapper.text()).toContain('401')
    // Error details should not be shown in the trigger dialog
    expect(wrapper.text()).not.toContain('InvalidAuthenticationToken')
    expect(wrapper.text()).not.toContain('HTTP 401')
  })

  it('does not render content when visible is false', () => {
    const wrapper = mount(TriggerResultDialog, {
      props: { visible: false, result: mockResult },
      ...mountOptions,
    })
    expect(wrapper.text()).not.toContain('me/drive/root')
  })

  it('emits update:visible when close button is clicked', async () => {
    const wrapper = mount(TriggerResultDialog, {
      props: { visible: true, result: mockResult },
      ...mountOptions,
    })
    // Find the close button (last button rendered)
    const buttons = wrapper.findAll('button')
    expect(buttons.length).toBeGreaterThan(0)
    await buttons[buttons.length - 1].trigger('click')
    expect(wrapper.emitted('update:visible')).toBeTruthy()
    expect(wrapper.emitted('update:visible')![0]).toEqual([false])
  })
})
