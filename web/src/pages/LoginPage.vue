<script setup lang="ts">
import { ref } from 'vue'
import { apiFetch } from '../services/apiClient'
import { setAuthenticatedUser } from '../services/sessionContext'

const email = ref('')
const password = ref('')
const error = ref('')

async function login() {
  error.value = ''
  try {
    await apiFetch('/api/login', {
      method: 'POST',
      body: JSON.stringify({ email: email.value, password: password.value }),
    })
    const me = await apiFetch<{ id: number; email: string }>('/api/me')
    setAuthenticatedUser(me)
    window.location.hash = '#/'
  } catch {
    error.value = 'Login failed'
  }
}
</script>

<template>
  <div style="padding: 12px; display: flex; gap: 8px">
    <input v-model="email" placeholder="email" />
    <input v-model="password" type="password" placeholder="password" />
    <button @click="login">Login</button>
    <span v-if="error" style="color: crimson">{{ error }}</span>
  </div>
</template>
