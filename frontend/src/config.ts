// Read the runtime config injected by Go
const runtimeConfig = (window as any).__E5_CONFIG__ ?? {}

export const pathPrefix: string = runtimeConfig.pathPrefix ?? import.meta.env.VITE_PATH_PREFIX ?? ''
