import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: parseInt(process.env.VITE_PORT || '4280'),
    proxy: {
      // SSE/streaming endpoint — needs a long timeout so Vite's http-proxy
      // doesn't cut the connection while the LLM is thinking before the
      // first token arrives (Gemini extended thinking can take 20-30 s).
      '/api/chat/stream': {
        target: process.env.VITE_API_URL || 'http://localhost:4290',
        changeOrigin: true,
        proxyTimeout: 300000, // 5 min
        timeout: 300000,      // 5 min
      },
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:4290',
        changeOrigin: true,
      },
    },
  },
})

