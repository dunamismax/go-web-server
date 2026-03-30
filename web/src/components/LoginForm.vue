<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';

import { BackendError, fetchAuthState, login } from '../lib/backend';
import {
  appendQueryParams,
  normalizeReturnTo,
  redirectTo,
} from '../lib/navigation';

const form = reactive({
  email: '',
  password: '',
});

const booting = ref(true);
const submitting = ref(false);
const notice = ref('');
const pageError = ref('');
const fieldErrors = ref<Record<string, string>>({});
const returnTo = ref('/profile');

function readQueryState() {
  const params = new URLSearchParams(window.location.search);
  returnTo.value = normalizeReturnTo(params.get('return_to'), '/profile');

  if (params.get('logged_out') === '1') {
    notice.value = 'You have been logged out.';
  }

  if (params.get('reason') === 'auth-required') {
    notice.value = 'Please sign in to continue.';
  }
}

function resetErrors() {
  pageError.value = '';
  fieldErrors.value = {};
}

function messageFromUnknown(error: unknown): string {
  return error instanceof Error ? error.message : 'Request failed.';
}

async function initialize() {
  booting.value = true;
  resetErrors();
  readQueryState();

  try {
    const state = await fetchAuthState();
    if (state.authenticated && state.user) {
      redirectTo(returnTo.value, true);
      return;
    }
  } catch (error) {
    pageError.value = messageFromUnknown(error);
  } finally {
    booting.value = false;
  }
}

async function submitLogin() {
  resetErrors();
  submitting.value = true;

  try {
    await login({
      email: form.email,
      password: form.password,
    });

    redirectTo(
      appendQueryParams(returnTo.value, {
        auth_notice: 'login',
      }),
      true,
    );
  } catch (error) {
    if (error instanceof BackendError) {
      fieldErrors.value = error.fieldErrors;
      pageError.value = error.message;
    } else {
      pageError.value = messageFromUnknown(error);
    }
  } finally {
    submitting.value = false;
  }
}

onMounted(() => {
  void initialize();
});
</script>

<template>
  <section class="card form-card stack-lg">
    <div>
      <p class="eyebrow">Session login</p>
      <h2>Sign in</h2>
      <p class="muted">
        This Astro route posts to <code>/api/auth/login</code> and relies on the same session
        cookies and CSRF middleware as the Go-rendered app.
      </p>
    </div>

    <p v-if="booting" class="muted">Checking your current session...</p>

    <form v-else class="stack-md" @submit.prevent="submitLogin">
      <p v-if="notice" class="notice success" role="status">{{ notice }}</p>
      <p v-if="pageError" class="notice error" role="alert">{{ pageError }}</p>

      <label class="field">
        <span>Email</span>
        <input v-model="form.email" autocomplete="email" name="email" type="email" />
        <small v-if="fieldErrors.email" class="field-error">{{ fieldErrors.email }}</small>
      </label>

      <label class="field">
        <span>Password</span>
        <input
          v-model="form.password"
          autocomplete="current-password"
          name="password"
          type="password"
        />
        <small v-if="fieldErrors.password" class="field-error">{{ fieldErrors.password }}</small>
      </label>

      <div class="actions">
        <button type="submit" class="button" :disabled="submitting">
          {{ submitting ? 'Signing in...' : 'Sign in' }}
        </button>
        <a class="button secondary" href="/auth/register">Need an account?</a>
      </div>
    </form>
  </section>
</template>
