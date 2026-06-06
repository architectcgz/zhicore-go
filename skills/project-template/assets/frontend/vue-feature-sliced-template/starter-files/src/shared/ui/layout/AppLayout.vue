<template>
  <div class="app-layout">
    <header class="app-layout__header">
      <div>
        <p class="app-layout__eyebrow">Feature-Sliced Starter</p>
        <h1>__APP_NAME__</h1>
      </div>
      <nav class="app-layout__actions">
        <RouterLink to="__DEFAULT_AUTH_REDIRECT__">Dashboard</RouterLink>
        <button
          type="button"
          @click="logout"
        >
          Logout
        </button>
      </nav>
    </header>
    <main class="app-layout__content">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router'

defineProps<{
  logout: () => void | Promise<void>
}>()
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
  background:
    radial-gradient(circle at top, rgba(59, 130, 246, 0.16), transparent 40%),
    linear-gradient(180deg, var(--color-bg-elevated) 0%, var(--color-bg-elevated-2) 100%);
  color: var(--color-text);
}

.app-layout__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 24px;
  border-bottom: 1px solid var(--color-border);
  backdrop-filter: blur(16px);
}

.app-layout__eyebrow {
  margin: 0 0 4px;
  font-size: 12px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--color-primary-soft);
}

.app-layout__content {
  padding: 24px;
}

.app-layout__actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.app-layout__actions a,
.app-layout__actions button {
  border: 1px solid var(--color-border-strong);
  border-radius: 999px;
  padding: 10px 14px;
  color: inherit;
  background: var(--color-panel-glass);
  text-decoration: none;
  cursor: pointer;
}

@media (max-width: 720px) {
  .app-layout__header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
