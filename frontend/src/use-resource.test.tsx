import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { useResource } from './use-resource'

afterEach(() => vi.useRealTimers())

it('waits for a slow response before scheduling the next poll', async () => {
  vi.useFakeTimers()
  let finish!: (response: Response) => void
  vi.stubGlobal(
    'fetch',
    vi.fn(
      () =>
        new Promise<Response>((resolve) => {
          finish = resolve
        }),
    ),
  )
  const { result, unmount } = renderHook(() =>
    useResource<{ count: number }>('/accounts', 5000),
  )
  await act(() => vi.advanceTimersByTimeAsync(6000))
  expect(fetch).toHaveBeenCalledOnce()
  await act(async () => {
    finish(new Response(JSON.stringify({ count: 1 })))
  })
  expect(result.current.data).toEqual({ count: 1 })
  await act(() => vi.advanceTimersByTimeAsync(5000))
  expect(fetch).toHaveBeenCalledTimes(2)
  unmount()
})

it('cancels requests when the page leaves and preserves data after a refresh failure', async () => {
  const mockFetch = vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(new Response(JSON.stringify({ count: 1 })))
    .mockRejectedValueOnce(new TypeError('offline'))
    .mockImplementation(() => new Promise<Response>(() => {}))
  vi.stubGlobal('fetch', mockFetch)
  const { result, unmount } = renderHook(() =>
    useResource<{ count: number }>('/accounts'),
  )
  await waitFor(() => expect(result.current.data).toEqual({ count: 1 }))
  await act(() => result.current.refresh())
  expect(result.current.data).toEqual({ count: 1 })
  expect(result.current.error).toBe('error.network_unavailable')
  act(() => {
    void result.current.refresh()
  })
  const signal = mockFetch.mock.calls[2]?.[1]?.signal
  unmount()
  expect(signal?.aborted).toBe(true)
})

it('never exposes a previous account after the resource address changes', async () => {
  let rejectNext!: (reason: Error) => void
  vi.stubGlobal(
    'fetch',
    vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(new Response(JSON.stringify({ id: 1 })))
      .mockImplementationOnce(
        () =>
          new Promise<Response>((_, reject) => {
            rejectNext = reject
          }),
      ),
  )
  const { result, rerender } = renderHook(
    ({ path }) => useResource<{ id: number }>(path),
    { initialProps: { path: '/accounts/1' } },
  )
  await waitFor(() => expect(result.current.data).toEqual({ id: 1 }))
  rerender({ path: '/accounts/2' })
  expect(result.current.data).toBeUndefined()
  await act(async () => {
    rejectNext(new Error('offline'))
    await Promise.resolve()
  })
  expect(result.current.data).toBeUndefined()
  expect(result.current.error).toBe('error.network_unavailable')
})
