<script setup lang="ts">
import { onMounted, ref } from 'vue';

import { fetchAuthState, logout } from '../lib/backend';
import {
  appendQueryParams,
  normalizeReturnTo,
  redirectTo,
} from '../lib/navigation';

const working = ref(true);
const error = ref('');
const target = ref('/auth/login');

async function runLogout() {
  working.value = true;
  error.value = '';

  const params = new URLSearchParams(window.location.search);
  target.value = normalizeReturnTo(params.get('return_to'), '/auth/login');

  try {
    await fetchAuthState();
    await logout();
    redirectTo(
      appendQueryParams(target.value, {
        logged_out: '1',
      }),
      true,
    );
  } catch (requestError) {
    error.value =
      requestError instanceof Error
        ? requestError.message
        : 'Unable to complete logout.';
  } finally {
    working.value = false;
  }
}

onMounted(() => {
  void runLogout();
});
</script>

<template>
  <section class="card stack-lg">
    <div>
      <p class="eyebrow">Session teardown</p>
      <h2>Logging out</h2>
      <p class="muted">
        The Astro frontend fetches a fresh CSRF token, calls <code>/api/auth/logout</code>, and
        then returns you to the sign-in page.
      </p>
    </div>

    <p v-if="working" class="muted">Ending your session...</p>
    <div v-else-if="error" class="stack-md">
      <p class="notice error" role="alert">{{ error }}</p>
      <div class="actions">
        <button type="button" class="button" @click="runLogout">Try again</button>
        <a class="button secondary" href="/">Back home</a>
      </div>
    </div>
  </section>
</template>
