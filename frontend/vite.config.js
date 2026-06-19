import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  define: {
    'process.env.NODE_ENV': JSON.stringify('development'),
  },
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: true,
      },
    },
  },
})
