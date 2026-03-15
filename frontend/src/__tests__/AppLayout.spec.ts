import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import AppLayout from '../components/AppLayout.vue'

describe('AppLayout', () => {
  it('renders router-view content', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/',
          component: AppLayout,
          children: [
            {
              path: 'test',
              component: { template: '<div data-testid="routed-content">Test Page</div>' },
            },
          ],
        },
      ],
    })

    await router.push('/test')
    await router.isReady()

    const wrapper = mount(AppLayout, {
      global: {
        plugins: [router],
      },
    })

    expect(wrapper.find('[data-testid="routed-content"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Test Page')
  })

  it('renders the sidebar component', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/',
          component: AppLayout,
          children: [
            { path: '', component: { template: '<div>Home</div>' } },
          ],
        },
      ],
    })

    await router.push('/')
    await router.isReady()

    const wrapper = mount(AppLayout, {
      global: {
        plugins: [router],
      },
    })

    // AppLayout includes AppSidebar which renders an <aside> element
    expect(wrapper.find('aside').exists()).toBe(true)
  })

  it('applies shared app-shell class to root element', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/',
          component: AppLayout,
          children: [
            { path: '', component: { template: '<div>Home</div>' } },
          ],
        },
      ],
    })

    await router.push('/')
    await router.isReady()

    const wrapper = mount(AppLayout, {
      global: {
        plugins: [router],
      },
    })

    expect(wrapper.find('.min-h-screen').exists()).toBe(true)
  })

  it('has a main element for content area', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/',
          component: AppLayout,
          children: [
            { path: '', component: { template: '<div>Home</div>' } },
          ],
        },
      ],
    })

    await router.push('/')
    await router.isReady()

    const wrapper = mount(AppLayout, {
      global: {
        plugins: [router],
      },
    })

    expect(wrapper.find('main').exists()).toBe(true)
  })
})
