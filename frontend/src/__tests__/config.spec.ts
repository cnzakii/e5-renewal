import { describe, it, expect } from 'vitest'

describe('runtime config', () => {
  it('exports pathPrefix as a string', async () => {
    // config.ts exports pathPrefix - just verify it's a string
    const mod = await import('../config')
    expect(typeof mod.pathPrefix).toBe('string')
  })

  it('pathPrefix falls back to empty string when no config set', async () => {
    // In test environment, window.__E5_CONFIG__ is undefined
    // and import.meta.env.VITE_PATH_PREFIX is undefined
    // so pathPrefix should be ''
    const mod = await import('../config')
    expect(mod.pathPrefix).toBe('')
  })
})
