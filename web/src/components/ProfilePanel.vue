<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';

import type { AuthStateResponse } from '../lib/backend';
import { fetchAuthState } from '../lib/backend';
import { appendQueryParams, redirectTo } from '../lib/navigation';

const loading = ref(true);
const redirecting = ref(false);
const error = ref('');
const notice = ref('');
const authState = ref<AuthStateResponse | null>(null);

const profileRows = computed(() => {
  const user = authState.value?.user;
  if (!user) {
    return [];
  }

  return [
    { label: 'User ID', value: String(user.id) },
    { label: 'Email', value: user.email },
    { label: 'Display name', value: user.name },
    { label: 'Active', value: user.is_active ? 'Yes' : 'No' },
  ];
});

function readNotice() {
  const params = new URLSearchParams(window.location.search);
  switch (params.get('auth_notice')) {
    case 'login':
      notice.value = 'Login successful.';
      break;
    case 'register':
      notice.value = 'Registration successful.';
      break;
    default:
      notice.value = '';
      break;
  }
}

async function initialize() {
  loading.value = true;
  error.value = '';
  readNotice();

  try {
    const state = await fetchAuthState();
    authState.value = state;

    if (!state.authenticated || !state.user) {
      redirecting.value = true;
      window.setTimeout(() => {
        redirectTo(
          appendQueryParams('/auth/login', {
            return_to: '/profile',
            reason: 'auth-required',
          }),
          true,
        );
      }, 400);
      return;
    }
  } catch (requestError) {
    error.value =
      requestError instanceof Error
        ? requestError.message
        : 'Unable to load the current session.';
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void initialize();
});
</script>

<template>
  <section class="card stack-lg">
    <div>
      <p class="eyebrow">Protected route</p>
      <h2>Your profile</h2>
      <p class="muted">
        This page bootstraps auth state from <code>/api/auth/state</code> and keeps the session on
        the same origin-friendly frontend path.
      </p>
    </div>

    <p v-if="loading" class="muted">Loading your session...</p>
    <p v-else-if="error" class="notice error" role="alert">{{ error }}</p>
    <p v-else-if="redirecting" class="notice warning" role="status">
      You are not signed in. Redirecting to the Astro login page...
    </p>
    <div v-else-if="authState?.user" class="stack-md">
      <p v-if="notice" class="notice success" role="status">{{ notice }}</p>

      <dl class="detail-list">
        <div v-for="row in profileRows" :key="row.label" class="detail-row">
          <dt>{{ row.label }}</dt>
          <dd>{{ row.value }}</dd>
        </div>
      </dl>

      <div class="actions">
        <a class="button" href="/users">Manage users</a>
        <a class="button secondary" href="/auth/logout">Log out</a>
      </div>
    </div>
  </section>
</template>
