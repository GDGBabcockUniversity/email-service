import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'
import { defineConfig } from 'vite'

export default defineConfig({
  base: '/admin/',
  plugins: [react(), tailwindcss()],
  resolve: { alias: { '@': path.resolve(__dirname, 'src') } },
  // Dev: run the Go service on :8080 and the console talks to it through this proxy.
  server: { proxy: { '/api': 'http://localhost:8080' } },
})
