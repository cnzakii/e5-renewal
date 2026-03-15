import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import AppLogo from '../components/AppLogo.vue'

describe('AppLogo', () => {
  it('renders favicon SVG instead of text badge', () => {
    const wrapper = mount(AppLogo)
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-logo"]').exists()).toBe(true)
    // Should NOT contain the old text-based E5 badge
    expect(wrapper.find('.logo-text').exists()).toBe(false)
    expect(wrapper.find('.logo-box').exists()).toBe(false)
  })

  it('uses default size of 40px', () => {
    const wrapper = mount(AppLogo)
    const svg = wrapper.find('svg')
    expect(svg.attributes('width')).toBe('40')
    expect(svg.attributes('height')).toBe('40')
  })

  it('applies custom size prop', () => {
    const wrapper = mount(AppLogo, {
      props: { size: 64 },
    })
    const svg = wrapper.find('svg')
    expect(svg.attributes('width')).toBe('64')
    expect(svg.attributes('height')).toBe('64')
  })

  it('uses prefixed SVG IDs to avoid collisions', () => {
    const wrapper = mount(AppLogo)
    const html = wrapper.html()
    expect(html).toContain('app-logo-bg')
    expect(html).toContain('app-logo-clip')
  })

  it('adds no-hover class when animated is false (default)', () => {
    const wrapper = mount(AppLogo)
    const container = wrapper.find('.logo-container')
    expect(container.classes()).toContain('no-hover')
  })

  it('removes no-hover class when animated is true', () => {
    const wrapper = mount(AppLogo, {
      props: { animated: true },
    })
    const container = wrapper.find('.logo-container')
    expect(container.classes()).not.toContain('no-hover')
  })
})
