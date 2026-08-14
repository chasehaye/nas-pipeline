import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  optimizeDeps: {
    exclude: ['maplibre-gl'],
  },
  // Dev server mirrors the nginx reverse proxy: relative /api calls are
  // forwarded to the local api, so the frontend uses the same paths in dev and
  // in the built image. Point the target at wherever your api runs locally.
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8090',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
