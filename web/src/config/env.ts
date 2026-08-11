export const env = {
  apiBase:
    (import.meta.env.VITE_API_BASE as string | undefined) ??
    'http://localhost:8090',
}
