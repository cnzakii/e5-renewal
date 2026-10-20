import { useCallback, useEffect, useRef, useState } from 'react'
import { request, errorKey } from './api'

interface ResourceState<T> {
  path: string
  data?: T
  error: string
  loading: boolean
}

export function useResource<T>(path: string, interval = 0) {
  const [state, setState] = useState<ResourceState<T>>({
    path,
    error: '',
    loading: true,
  })
  const sequence = useRef(0)
  const mounted = useRef(true)
  const controller = useRef<AbortController | null>(null)

  const refresh = useCallback(async () => {
    // Only the newest request may publish a result after navigation or refresh.
    const current = ++sequence.current
    controller.current?.abort()

    const active = new AbortController()
    controller.current = active
    setState((previous) => ({
      path,
      data: previous.path === path ? previous.data : undefined,
      error: previous.path === path ? previous.error : '',
      loading: true,
    }))

    try {
      const value = await request<T>(path, 'GET', undefined, active.signal)
      if (mounted.current && current === sequence.current) {
        setState({ path, data: value, error: '', loading: false })
      }
    } catch (error) {
      if (mounted.current && current === sequence.current) {
        setState((previous) => ({
          path,
          data: previous.path === path ? previous.data : undefined,
          error: errorKey(error),
          loading: false,
        }))
      }
    }
  }, [path])

  useEffect(() => {
    mounted.current = true
    let timer: number | undefined
    let disposed = false

    async function poll(initial = false) {
      if (initial || document.visibilityState === 'visible') {
        await refresh()
      }

      if (!disposed && interval) {
        timer = window.setTimeout(() => void poll(), interval)
      }
    }

    void poll(true)

    return () => {
      disposed = true
      mounted.current = false
      controller.current?.abort()
      window.clearTimeout(timer)
    }
  }, [refresh, interval])

  // A response belongs to its address, even while the next route is loading.
  const current = state.path === path
  return {
    data: current ? state.data : undefined,
    error: current ? state.error : '',
    loading: current ? state.loading : true,
    refresh,
  }
}
