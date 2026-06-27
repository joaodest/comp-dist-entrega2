import { defineConfig } from 'vite';

// O showcase roda 100% com dados mock (nao precisa do backend).
// O proxy abaixo so e usado quando ligarmos o "modo ao vivo" futuro,
// encaminhando as chamadas do Gateway (porta 8080) e evitando CORS no dev.
export default defineConfig({
  server: {
    port: 5173,
    open: true,
    proxy: {
      '/v1': { target: 'http://localhost:8080', changeOrigin: true },
      '/healthz': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
});
