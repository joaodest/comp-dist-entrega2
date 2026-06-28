import { defineConfig } from 'vite';

// O showcase roda 100% com dados mock (nao precisa do backend).
// O proxy abaixo so e usado quando ligarmos o "modo ao vivo" futuro,
// encaminhando as chamadas do Gateway (porta 8080) e evitando CORS no dev.
export default defineConfig({
  server: {
    port: 5173,
    open: true,
    proxy: {
      // ws: true encaminha o WebSocket de tempo real (/v1/match/ws) ao Gateway.
      '/v1': { target: 'http://localhost:8080', changeOrigin: true, ws: true },
      '/healthz': { target: 'http://localhost:8080', changeOrigin: true },
      '/readyz': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
});
