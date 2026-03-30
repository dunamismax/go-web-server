import vue from '@astrojs/vue';
import { defineConfig } from 'astro/config';

const defaultBackendOrigin = 'http://127.0.0.1:8080';
const defaultProxyBase = '/_backend';

function normalizeBasePath(value) {
  const trimmed = (value ?? '').trim();
  if (trimmed === '') {
    return defaultProxyBase;
  }

  const withLeadingSlash = trimmed.startsWith('/') ? trimmed : `/${trimmed}`;
  return withLeadingSlash.endsWith('/') && withLeadingSlash !== '/'
    ? withLeadingSlash.slice(0, -1)
    : withLeadingSlash;
}

const backendOrigin =
  process.env.FRONTEND_BACKEND_ORIGIN ?? defaultBackendOrigin;
const backendProxyBase = normalizeBasePath(
  process.env.PUBLIC_BACKEND_PROXY_BASE ?? defaultProxyBase,
);

const backendProxy = {
  target: backendOrigin,
  changeOrigin: true,
  secure: false,
  rewrite: (path) => {
    if (!path.startsWith(backendProxyBase)) {
      return path;
    }

    const rewritten = path.slice(backendProxyBase.length);
    return rewritten === '' ? '/' : rewritten;
  },
};

export default defineConfig({
  output: 'static',
  integrations: [vue()],
  server: {
    host: true,
    port: 4321,
  },
  vite: {
    server: {
      proxy: {
        [backendProxyBase]: backendProxy,
      },
    },
    preview: {
      proxy: {
        [backendProxyBase]: backendProxy,
      },
    },
  },
});
