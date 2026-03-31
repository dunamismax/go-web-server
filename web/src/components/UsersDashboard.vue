<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';

import {
  BackendError,
  createUser,
  deactivateUser,
  deleteUser,
  fetchAuthState,
  fetchUser,
  fetchUserCount,
  fetchUsers,
  type ManagedUser,
  type ManagedUserPayload,
  type SessionUser,
  updateUser,
} from '../lib/backend';
import { appendQueryParams, redirectTo } from '../lib/navigation';

type UserFormState = {
  email: string;
  name: string;
  password: string;
  confirm_password: string;
  bio: string;
  avatar_url: string;
};

function emptyForm(): UserFormState {
  return {
    email: '',
    name: '',
    password: '',
    confirm_password: '',
    bio: '',
    avatar_url: '',
  };
}

const dateFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short',
});

const loading = ref(true);
const reloading = ref(false);
const redirecting = ref(false);
const saving = ref(false);
const loadingUserId = ref<number | null>(null);
const deactivatingId = ref<number | null>(null);
const deletingId = ref<number | null>(null);
const selectedUserId = ref<number | null>(null);
const authUser = ref<SessionUser | null>(null);
const users = ref<ManagedUser[]>([]);
const userCount = ref(0);
const notice = ref('');
const pageError = ref('');
const fieldErrors = ref<Record<string, string>>({});
const form = reactive<UserFormState>(emptyForm());

const editing = computed(() => selectedUserId.value !== null);
const submitLabel = computed(() => {
  if (saving.value) {
    return editing.value ? 'Saving changes...' : 'Creating user...';
  }

  return editing.value ? 'Save changes' : 'Create user';
});
const activeUserLabel = computed(() => {
  return userCount.value === 1
    ? '1 active user'
    : `${userCount.value} active users`;
});
const operatorLabel = computed(() => {
  if (!authUser.value) {
    return 'No active session';
  }

  return `${authUser.value.name} (${authUser.value.email})`;
});
const formEyebrow = computed(() => {
  return editing.value ? 'Edit managed user' : 'Create managed user';
});
const formTitle = computed(() => {
  return editing.value ? 'Update user' : 'Create user';
});

function resetForm() {
  Object.assign(form, emptyForm());
  selectedUserId.value = null;
  fieldErrors.value = {};
}

function applyUserToForm(user: ManagedUser) {
  Object.assign(form, {
    email: user.email,
    name: user.name,
    password: '',
    confirm_password: '',
    bio: user.bio ?? '',
    avatar_url: user.avatar_url ?? '',
  });
}

function resetErrors() {
  pageError.value = '';
  fieldErrors.value = {};
}

function messageFromUnknown(error: unknown, fallback: string): string {
  return error instanceof Error ? error.message : fallback;
}

function formatTimestamp(value: string): string {
  const timestamp = new Date(value);
  if (Number.isNaN(timestamp.getTime())) {
    return value;
  }

  return dateFormatter.format(timestamp);
}

function redirectToLogin() {
  redirecting.value = true;
  window.setTimeout(() => {
    redirectTo(
      appendQueryParams('/auth/login', {
        return_to: '/users',
        reason: 'auth-required',
      }),
      true,
    );
  }, 400);
}

async function refreshUsersData(showSpinner = false) {
  if (showSpinner) {
    reloading.value = true;
  }

  try {
    const [userList, count] = await Promise.all([
      fetchUsers(),
      fetchUserCount(),
    ]);
    users.value = userList.users;
    userCount.value = count.count;
  } catch (error) {
    if (error instanceof BackendError && error.status === 401) {
      redirectToLogin();
      return;
    }

    pageError.value = messageFromUnknown(error, 'Unable to load users.');
  } finally {
    if (showSpinner) {
      reloading.value = false;
    }
  }
}

async function bootstrap() {
  loading.value = true;
  resetErrors();

  try {
    const state = await fetchAuthState();
    if (!state.authenticated || !state.user) {
      redirectToLogin();
      return;
    }

    authUser.value = state.user;
    await refreshUsersData();
  } catch (error) {
    pageError.value = messageFromUnknown(
      error,
      'Unable to load the users dashboard.',
    );
  } finally {
    loading.value = false;
  }
}

function beginCreate() {
  notice.value = '';
  resetErrors();
  resetForm();
}

async function beginEdit(id: number) {
  notice.value = '';
  resetErrors();
  loadingUserId.value = id;

  try {
    const response = await fetchUser(id);
    selectedUserId.value = response.user.id;
    applyUserToForm(response.user);
  } catch (error) {
    if (error instanceof BackendError && error.status === 401) {
      redirectToLogin();
      return;
    }

    pageError.value = messageFromUnknown(error, 'Unable to load that user.');
  } finally {
    loadingUserId.value = null;
  }
}

function buildPayload(): ManagedUserPayload {
  return {
    email: form.email,
    name: form.name,
    password: form.password,
    confirm_password: form.confirm_password,
    bio: form.bio,
    avatar_url: form.avatar_url,
  };
}

async function submitForm() {
  resetErrors();
  saving.value = true;

  try {
    if (editing.value && selectedUserId.value !== null) {
      const response = await updateUser(selectedUserId.value, buildPayload());
      notice.value = response.message;
      applyUserToForm(response.user);
    } else {
      const response = await createUser(buildPayload());
      notice.value = response.message;
      resetForm();
    }

    await refreshUsersData();
  } catch (error) {
    if (error instanceof BackendError) {
      if (error.status === 401) {
        redirectToLogin();
        return;
      }

      fieldErrors.value = error.fieldErrors;
      pageError.value = error.message;
      return;
    }

    pageError.value = messageFromUnknown(error, 'Unable to save user changes.');
  } finally {
    saving.value = false;
  }
}

async function handleDeactivate(user: ManagedUser) {
  if (!window.confirm(`Deactivate ${user.name}?`)) {
    return;
  }

  resetErrors();
  deactivatingId.value = user.id;

  try {
    const response = await deactivateUser(user.id);
    notice.value = response.message;

    if (selectedUserId.value === user.id) {
      resetForm();
    }

    await refreshUsersData();
  } catch (error) {
    if (error instanceof BackendError && error.status === 401) {
      redirectToLogin();
      return;
    }

    pageError.value = messageFromUnknown(error, 'Unable to deactivate user.');
  } finally {
    deactivatingId.value = null;
  }
}

async function handleDelete(user: ManagedUser) {
  if (!window.confirm(`Permanently delete ${user.name}?`)) {
    return;
  }

  resetErrors();
  deletingId.value = user.id;

  try {
    const response = await deleteUser(user.id);
    notice.value = response.message;

    if (selectedUserId.value === user.id) {
      resetForm();
    }

    await refreshUsersData();
  } catch (error) {
    if (error instanceof BackendError && error.status === 401) {
      redirectToLogin();
      return;
    }

    pageError.value = messageFromUnknown(error, 'Unable to delete user.');
  } finally {
    deletingId.value = null;
  }
}

onMounted(() => {
  void bootstrap();
});
</script>

<template>
  <section class="dashboard-grid">
    <article class="card stack-lg">
      <div class="card-header">
        <div>
          <p class="eyebrow">Protected Astro route</p>
          <h2>User management</h2>
          <p class="muted">
            The full CRUD surface now runs through Astro + Vue and the explicit
            <code>/api/users/*</code> contracts instead of HTMX fragments.
          </p>
        </div>

        <div class="actions">
          <button type="button" class="button secondary" @click="refreshUsersData(true)">
            Reload data
          </button>
          <button type="button" class="button" @click="beginCreate">New user</button>
        </div>
      </div>

      <div class="stat-grid">
        <div class="panel">
          <span class="label">Signed in as</span>
          <strong>{{ operatorLabel }}</strong>
          <p class="muted compact">Protected CRUD stays on the same session cookie path.</p>
        </div>

        <div class="panel">
          <span class="label">Active users</span>
          <strong>{{ activeUserLabel }}</strong>
          <p class="muted compact">Loaded from <code>/api/users/count</code>.</p>
        </div>
      </div>

      <p v-if="loading" class="muted">Loading the protected users dashboard...</p>
      <p v-else-if="redirecting" class="notice warning" role="status">
        Your session is missing or expired. Redirecting to the Astro login flow...
      </p>
      <template v-else>
        <p v-if="notice" class="notice success" role="status">{{ notice }}</p>
        <p v-if="pageError" class="notice error" role="alert">{{ pageError }}</p>
        <p v-if="reloading" class="muted">Refreshing users and count...</p>

        <div v-if="users.length > 0" class="user-list">
          <article v-for="user in users" :key="user.id" class="panel user-card stack-md">
            <div class="card-header">
              <div>
                <h3>{{ user.name }}</h3>
                <p class="muted">{{ user.email }}</p>
              </div>
              <span class="status-pill">active</span>
            </div>

            <p v-if="user.bio" class="muted">{{ user.bio }}</p>
            <p v-else class="muted">No bio set for this user yet.</p>

            <dl class="detail-list compact-detail-list">
              <div class="detail-row">
                <dt>User ID</dt>
                <dd>{{ user.id }}</dd>
              </div>
              <div class="detail-row">
                <dt>Created</dt>
                <dd>{{ formatTimestamp(user.created_at) }}</dd>
              </div>
              <div class="detail-row">
                <dt>Updated</dt>
                <dd>{{ formatTimestamp(user.updated_at) }}</dd>
              </div>
              <div class="detail-row">
                <dt>Avatar URL</dt>
                <dd>
                  <a
                    v-if="user.avatar_url"
                    class="inline-link"
                    :href="user.avatar_url"
                    target="_blank"
                    rel="noreferrer"
                  >
                    Open avatar
                  </a>
                  <span v-else>Not set</span>
                </dd>
              </div>
            </dl>

            <div class="actions">
              <button
                type="button"
                class="button secondary"
                :disabled="saving || deactivatingId === user.id || deletingId === user.id"
                @click="beginEdit(user.id)"
              >
                {{ loadingUserId === user.id ? 'Loading...' : 'Edit' }}
              </button>
              <button
                type="button"
                class="button ghost"
                :disabled="saving || deletingId === user.id"
                @click="handleDeactivate(user)"
              >
                {{ deactivatingId === user.id ? 'Deactivating...' : 'Deactivate' }}
              </button>
              <button
                type="button"
                class="button destructive"
                :disabled="saving || deactivatingId === user.id"
                @click="handleDelete(user)"
              >
                {{ deletingId === user.id ? 'Deleting...' : 'Delete' }}
              </button>
            </div>
          </article>
        </div>
        <div v-else class="panel empty-state stack-md">
          <h3>No active users yet</h3>
          <p class="muted">
            Create the first managed user with the form on this page. The list and count will update
            through the explicit JSON contracts.
          </p>
        </div>
      </template>
    </article>

    <aside class="card stack-lg">
      <div class="card-header">
        <div>
          <p class="eyebrow">{{ formEyebrow }}</p>
          <h2>{{ formTitle }}</h2>
          <p class="muted">
            {{ editing ? 'Updates go to /api/users/:id.' : 'Creates go to /api/users.' }}
            Password fields are optional during edits.
          </p>
        </div>

        <button v-if="editing" type="button" class="button ghost" @click="beginCreate">
          Cancel edit
        </button>
      </div>

      <p v-if="loadingUserId !== null" class="muted">Loading the selected user record...</p>

      <form class="stack-md" @submit.prevent="submitForm">
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
            <span>Password <small class="muted">{{ editing ? 'optional' : 'required' }}</small></span>
            <input
              v-model="form.password"
              :autocomplete="editing ? 'new-password' : 'new-password'"
              name="password"
              type="password"
            />
            <small v-if="fieldErrors.password" class="field-error">{{ fieldErrors.password }}</small>
          </label>

          <label class="field">
            <span>Confirm password</span>
            <input
              v-model="form.confirm_password"
              :autocomplete="editing ? 'new-password' : 'new-password'"
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
          <textarea v-model="form.bio" name="bio" rows="4"></textarea>
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
          <button type="submit" class="button" :disabled="saving || loadingUserId !== null">
            {{ submitLabel }}
          </button>
          <button type="button" class="button secondary" @click="beginCreate">
            Reset form
          </button>
        </div>
      </form>
    </aside>
  </section>
</template>
