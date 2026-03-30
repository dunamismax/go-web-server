<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';

import { BackendError, fetchAuthState, register } from '../lib/backend';
import {
  appendQueryParams,
  normalizeReturnTo,
  redirectTo,
} from '../lib/navigation';

const form = reactive({
  email: '',
  name: '',
  password: '',
  confirm_password: '',
  bio: '',
  avatar_url: '',
});

const booting = ref(true);
const submitting = ref(false);
const pageError = ref('');
const fieldErrors = ref<Record<string, string>>({});
const returnTo = ref('/profile');

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

  const params = new URLSearchParams(window.location.search);
  returnTo.value = normalizeReturnTo(params.get('return_to'), '/profile');

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

async function submitRegistration() {
  resetErrors();
  submitting.value = true;

  try {
    await register({ ...form });
    redirectTo(
      appendQueryParams(returnTo.value, {
        auth_notice: 'register',
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
      <p class="eyebrow">Session bootstrap</p>
      <h2>Create account</h2>
      <p class="muted">
        Registration now runs through the Astro frontend and the explicit
        <code>/api/auth/register</code> contract.
      </p>
    </div>

    <p v-if="booting" class="muted">Checking your current session...</p>

    <form v-else class="stack-md" @submit.prevent="submitRegistration">
      <p v-if="pageError" class="notice error" role="alert">{{ pageError }}</p>

      <div class="two-column-grid">
        <label class="field">
          <span>Name</span>
          <input v-model="form.name" autocomplete="name" name="name" type="text" />
          <small v-if="fieldErrors.name" class="field-error">{{ fieldErrors.name }}</small>
        </label>

        <label class="field">
          <span>Email</span>
          <input v-model="form.email" autocomplete="email" name="email" type="email" />
          <small v-if="fieldErrors.email" class="field-error">{{ fieldErrors.email }}</small>
        </label>
      </div>

      <div class="two-column-grid">
        <label class="field">
          <span>Password</span>
          <input
            v-model="form.password"
            autocomplete="new-password"
            name="password"
            type="password"
          />
          <small v-if="fieldErrors.password" class="field-error">{{ fieldErrors.password }}</small>
        </label>

        <label class="field">
          <span>Confirm password</span>
          <input
            v-model="form.confirm_password"
            autocomplete="new-password"
            name="confirm_password"
            type="password"
          />
          <small v-if="fieldErrors.confirm_password" class="field-error">
            {{ fieldErrors.confirm_password }}
          </small>
        </label>
      </div>

      <label class="field">
        <span>Bio <small class="muted">optional</small></span>
        <textarea v-model="form.bio" name="bio" rows="4" />
        <small v-if="fieldErrors.bio" class="field-error">{{ fieldErrors.bio }}</small>
      </label>

      <label class="field">
        <span>Avatar URL <small class="muted">optional</small></span>
        <input
          v-model="form.avatar_url"
          autocomplete="url"
          name="avatar_url"
          type="url"
        />
        <small v-if="fieldErrors.avatar_url" class="field-error">
          {{ fieldErrors.avatar_url }}
        </small>
      </label>

      <div class="actions">
        <button type="submit" class="button" :disabled="submitting">
          {{ submitting ? 'Creating account...' : 'Create account' }}
        </button>
        <a class="button secondary" href="/auth/login">Already have an account?</a>
      </div>
    </form>
  </section>
</template>
