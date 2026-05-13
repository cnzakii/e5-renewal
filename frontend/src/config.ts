// Read the runtime config injected by Go
declare global {
  interface Window {
    __E5_CONFIG__?: {
      pathPrefix?: string
    }
  }
}

const runtimeConfig = window.__E5_CONFIG__ ?? {}

export const pathPrefix: string = runtimeConfig.pathPrefix ?? import.meta.env.VITE_PATH_PREFIX ?? ''
