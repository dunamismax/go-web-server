<script setup lang="ts">
import { onMounted, ref } from 'vue';
import type { HealthResponse } from '../lib/backend';
import { configuredBackendBase, fetchHealth } from '../lib/backend';

const backendProxyBase = configuredBackendBase();
const health = ref<HealthResponse | null>(null);
const error = ref('');
const loading = ref(true);

async function loadHealth() {
  loading.value = true;
  error.value = '';

  try {
    health.value = await fetchHealth();
  } catch (err) {
    health.value = null;
    error.value =
      err instanceof Error ? err.message : 'Unable to reach the Go backend.';
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void loadHealth();
});
</script>

<template>
  <main class="shell">
    <section class="hero card">
      <p class="eyebrow">Phase 1 migration workspace</p>
      <h1>go-web-server frontend migration shell</h1>
      <p class="lead">
        Astro owns the page shell, Vue owns the interactive status card, and Bun owns the
        frontend toolchain. The legacy Templ + HTMX app still remains live while this workspace
        is staged in parallel.
      </p>
      <div class="meta-grid">
        <div>
          <span class="label">Frontend lane</span>
          <strong>TypeScript + Bun + Astro + Vue</strong>
        </div>
        <div>
          <span class="label">Backend lane</span>
          <strong>Go + Echo + PostgreSQL</strong>
        </div>
        <div>
          <span class="label">Local backend proxy</span>
          <code>{{ backendProxyBase }}/*</code>
        </div>
      </div>
    </section>

    <section class="grid">
      <article class="card status-card">
        <div class="card-header">
          <div>
            <p class="eyebrow">Development handshake</p>
            <h2>Backend health</h2>
          </div>
          <button type="button" class="button" @click="loadHealth">Reload</button>
        </div>

        <p v-if="loading" class="muted">Checking the Go app through the same-origin proxy...</p>
        <p v-else-if="error" class="error">{{ error }}</p>
        <div v-else-if="health" class="status-details">
          <p>
            <strong>Status:</strong>
            <span class="status-pill">{{ health.status }}</span>
          </p>
          <p><strong>Service:</strong> {{ health.service }} {{ health.version }}</p>
          <p><strong>Uptime:</strong> {{ health.uptime }}</p>
          <ul>
            <li v-for="(value, key) in health.checks" :key="key">
              <strong>{{ key }}:</strong> {{ value }}
            </li>
          </ul>
        </div>
      </article>

      <article class="card">
        <p class="eyebrow">What Phase 1 covers</p>
        <h2>Safe migration scaffolding</h2>
        <ul>
          <li>Astro + Vue workspace under <code>web/</code></li>
          <li>Bun, Biome, unit test, and Playwright scaffolding</li>
          <li>Same-origin-friendly proxy path for local backend requests</li>
          <li>Mage tasks to install, check, build, preview, and run the frontend</li>
        </ul>
      </article>

      <article class="card">
        <p class="eyebrow">What stays for later</p>
        <h2>Not ported yet</h2>
        <ul>
          <li>Login and registration flows still live in Templ</li>
          <li>User CRUD still depends on HTMX fragment routes</li>
          <li>Backend contracts still need Phase 2 cleanup before real page parity work</li>
        </ul>
      </article>
    </section>
  </main>
</template>
