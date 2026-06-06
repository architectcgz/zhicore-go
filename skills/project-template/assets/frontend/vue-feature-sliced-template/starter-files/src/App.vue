<template>
  <div class="app-root">
    <RouterView v-slot="{ Component, route: resolvedRoute }">
      <Transition
        name="app-route"
        mode="out-in"
        appear
      >
        <component
          :is="Component"
          :key="resolvedRoute.matched[0]?.path || resolvedRoute.path"
        />
      </Transition>
    </RouterView>
  </div>
</template>

<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<style scoped>
.app-root {
  min-height: 100vh;
}
</style>
