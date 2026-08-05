import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // Proxy das chamadas de API para o backend em dev (evita CORS local).
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
