import { describe, expect, it } from 'vitest'

import viteConfig from '../../vite.config'

describe('vite production config', () => {
  it('does not split ECharts/ZRender internals into custom chunk groups', () => {
    const config = viteConfig as {
      build?: {
        rolldownOptions?: {
          output?: {
            codeSplitting?: {
              groups?: unknown[]
            }
          }
        }
      }
    }

    expect(config.build?.rolldownOptions?.output?.codeSplitting?.groups).toBeUndefined()
  })
})
