<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';

import type { AuthStateResponse, HealthResponse } from '../lib/backend';
import {
  configuredBackendBase,
  fetchAuthState,
  fetchHealth,
} from '../lib/backend';

const backendProxyBase = configuredBackendBase();
const authState = ref<AuthStateResponse | null>(null);
const authError = ref('');
const health = ref<HealthResponse | null>(null);
const healthError = ref('');
const loading = ref(true);

const sessionLabel = computed(() => {
  if (!authState.value?.authenticated || !authState.value.user) {
    return 'Signed out';
  }

  return `${authState.value.user.name} (${authState.value.user.email})`;
});

async function loadHomeData() {
  loading.value = true;
  authError.value = '';
  healthError.value = '';

  const [authResult, healthResult] = await Promise.allSettled([
    fetchAuthState(),
    fetchHealth(),
  ]);

  if (authResult.status === 'fulfilled') {
    authState.value = authResult.value;
  } else {
    authState.value = null;
    authError.value =
      authResult.reason instanceof Error
        ? authResult.reason.message
        : 'Unable to load the current session.';
  }

  if (healthResult.status === 'fulfilled') {
    health.value = healthResult.value;
  } else {
    health.value = null;
    healthError.value =
      healthResult.reason instanceof Error
        ? healthResult.reason.message
        : 'Unable to reach the Go backend.';
  }

  loading.value = false;
}

onMounted(() => {
  void loadHomeData();
});
</script>

<template>
  <section class="grid page-grid">
    <article class="card status-card stack-lg">
      <div class="card-header">
        <div>
          <p class="eyebrow">Phase 3 auth shell</p>
          <h2>Session-aware home page</h2>
        </div>
        <button type="button" class="button" @click="loadHomeData">Reload</button>
      </div>

      <p class="muted">
        Astro owns this page. Vue handles the live auth bootstrap and health check through
        <code>{{ backendProxyBase }}/*</code>.
      </p>

      <p v-if="loading" class="muted">Loading session state and backend health...</p>

      <div v-else class="stack-md">
        <div class="panel">
          <span class="label">Current session</span>
          <strong>{{ sessionLabel }}</strong>
          <p v-if="authState?.authenticated && authState.user" class="muted compact">
            Same-origin cookies and CSRF are already wired for the Astro frontend path.
          </p>
          <p v-else class="muted compact">
            Sign in or register through Astro to start a protected session without touching Templ.
          </p>
          <p v-if="authError" class="error compact">{{ authError }}</p>
        </div>

        <div class="panel">
          <span class="label">Backend health</span>
          <template v-if="health">
            <p class="compact">
              <strong>Status:</strong>
              <span class="status-pill">{{ health.status }}</span>
            </p>
            <p class="compact"><strong>Service:</strong> {{ health.service }} {{ health.version }}</p>
            <p class="compact"><strong>Uptime:</strong> {{ health.uptime }}</p>
          </template>
          <p v-else class="error compact">{{ healthError }}</p>
        </div>

        <div class="actions">
          <template v-if="authState?.authenticated">
            <a class="button" href="/profile">Open profile</a>
            <a class="button secondary" href="/users">Manage users</a>
            <a class="button ghost" href="/auth/logout">Log out</a>
          </template>
          <template v-else>
            <a class="button" href="/auth/login">Sign in</a>
            <a class="button secondary" href="/auth/register">Create account</a>
          </template>
        </div>
      </div>
    </article>

    <article class="card stack-md">
      <p class="eyebrow">What moved already</p>
      <h2>Astro + Vue auth coverage</h2>
      <ul>
        <li>Home, login, registration, logout, and profile pages now exist in <code>web/</code></li>
        <li>Auth flows talk to <code>/api/auth/*</code> instead of HTMX redirects</li>
        <li>Protected pages can redirect unauthenticated users back to the Astro login route</li>
      </ul>
    </article>

    <article class="card stack-md">
      <p class="eyebrow">Still ahead</p>
      <h2>Phase 4 target</h2>
      <ul>
        <li>Port the <code>/users</code> CRUD screen to Astro + Vue</li>
        <li>Replace HTMX fragments with the documented JSON contracts</li>
        <li>Retire the legacy browser path only after the new CRUD surface reaches parity</li>
      </ul>
    </article>
  </section>
</template>
