import { defineConfig } from 'vite';

const localApi = {
  target: 'http://127.0.0.1:8080',
  changeOrigin: true,
  configure(proxy) {
    // The local API validates Origin against Host. The dev proxy changes the
    // upstream Host, so forward a matching local Origin for proxied requests.
    proxy.on('proxyReq', (proxyReq) => {
      proxyReq.setHeader('origin', 'http://127.0.0.1:8080');
    });
  },
};

export default defineConfig({
  server: {
    proxy: {
      '/api': localApi,
      '/ws': { target: 'ws://127.0.0.1:8080', ws: true },
    },
  },
  build: { sourcemap: false },
});
