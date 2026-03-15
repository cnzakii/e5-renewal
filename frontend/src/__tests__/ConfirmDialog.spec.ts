import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import ConfirmDialog from '../components/ConfirmDialog.vue'

const mountOptions = {
  global: {
    stubs: {
      Teleport: true,
    },
  },
}

describe('ConfirmDialog', () => {
  it('renders title and message when visible', () => {
    const wrapper = mount(ConfirmDialog, {
      props: {
        visible: true,
        title: 'Delete Item',
        message: 'Are you sure you want to delete this?',
      },
      ...mountOptions,
    })
    expect(wrapper.text()).toContain('Delete Item')
    expect(wrapper.text()).toContain('Are you sure you want to delete this?')
  })

  it('does not render content when visible is false', () => {
    const wrapper = mount(ConfirmDialog, {
      props: {
        visible: false,
        title: 'Delete Item',
        message: 'Are you sure?',
      },
      ...mountOptions,
    })
    expect(wrapper.text()).not.toContain('Delete Item')
  })

  it('emits confirm and update:visible when confirm button is clicked', async () => {
    const wrapper = mount(ConfirmDialog, {
      props: {
        visible: true,
        title: 'Confirm',
        message: 'Proceed?',
        confirmText: 'Yes',
        cancelText: 'No',
      },
      ...mountOptions,
    })

    // The confirm button is the last button (second in the actions row)
    const buttons = wrapper.findAll('button')
    const confirmBtn = buttons.find((b) => b.text() === 'Yes')
    expect(confirmBtn).toBeTruthy()

    await confirmBtn!.trigger('click')
    expect(wrapper.emitted('confirm')).toBeTruthy()
    expect(wrapper.emitted('update:visible')).toBeTruthy()
    expect(wrapper.emitted('update:visible')![0]).toEqual([false])
  })

  it('emits cancel and update:visible when cancel button is clicked', async () => {
    const wrapper = mount(ConfirmDialog, {
      props: {
        visible: true,
        title: 'Confirm',
        message: 'Proceed?',
        confirmText: 'Yes',
        cancelText: 'No',
      },
      ...mountOptions,
    })

    const buttons = wrapper.findAll('button')
    const cancelBtn = buttons.find((b) => b.text() === 'No')
    expect(cancelBtn).toBeTruthy()

    await cancelBtn!.trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
    expect(wrapper.emitted('update:visible')).toBeTruthy()
    expect(wrapper.emitted('update:visible')![0]).toEqual([false])
  })

  it('emits cancel when backdrop is clicked', async () => {
    const wrapper = mount(ConfirmDialog, {
      props: {
        visible: true,
        title: 'Confirm',
        message: 'Proceed?',
      },
      ...mountOptions,
    })

    // Backdrop is the first inner div with absolute positioning
    const backdrop = wrapper.find('.absolute')
    await backdrop.trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
    expect(wrapper.emitted('update:visible')![0]).toEqual([false])
  })

  it('uses default confirm and cancel text', () => {
    const wrapper = mount(ConfirmDialog, {
      props: {
        visible: true,
        title: 'Confirm',
        message: 'Sure?',
      },
      ...mountOptions,
    })
    // Default props are Chinese: 确定 / 取消
    expect(wrapper.text()).toContain('确定')
    expect(wrapper.text()).toContain('取消')
  })

  it('applies danger styling to confirm button when danger is true', () => {
    const wrapper = mount(ConfirmDialog, {
      props: {
        visible: true,
        title: 'Delete',
        message: 'Danger!',
        danger: true,
      },
      ...mountOptions,
    })

    const buttons = wrapper.findAll('button')
    // The confirm button (last) should have red/danger classes
    const confirmBtn = buttons[buttons.length - 1]
    expect(confirmBtn.classes().some((c) => c.includes('bg-red-500'))).toBe(true)
  })

  it('does not apply danger styling when danger is false', () => {
    const wrapper = mount(ConfirmDialog, {
      props: {
        visible: true,
        title: 'Save',
        message: 'Save changes?',
        danger: false,
      },
      ...mountOptions,
    })

    const buttons = wrapper.findAll('button')
    const confirmBtn = buttons[buttons.length - 1]
    expect(confirmBtn.classes().some((c) => c.includes('bg-red-500'))).toBe(false)
    expect(confirmBtn.classes().some((c) => c.includes('bg-apple-blue'))).toBe(true)
  })
})
