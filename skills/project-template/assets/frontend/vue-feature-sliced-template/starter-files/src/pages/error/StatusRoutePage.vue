<template>
  <section class="status-page">
    <div class="status-page__panel">
      <p class="status-page__code">{{ normalizedStatus }}</p>
      <h2>{{ title }}</h2>
      <p>{{ description }}</p>
      <RouterLink to="__DEFAULT_LOGIN_PATH__">返回登录页</RouterLink>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'

const props = defineProps<{
  status: number
}>()

const normalizedStatus = computed(() => Number.isFinite(props.status) ? props.status : 500)

const title = computed(() => {
  if (normalizedStatus.value === 401) return '登录状态已失效'
  if (normalizedStatus.value === 404) return '页面不存在'
  return '页面暂时不可用'
})

const description = computed(() => {
  if (normalizedStatus.value === 401) return '请重新登录后继续访问。'
  if (normalizedStatus.value === 404) return '请检查路由配置或页面落点。'
  return '这里保留一个统一错误落点，便于项目后续扩展错误体验。'
})
</script>

<style scoped>
.status-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 24px;
  background: linear-gradient(180deg, #0f172a 0%, #111827 100%);
}

.status-page__panel {
  width: min(420px, 100%);
  padding: 28px;
  border-radius: 24px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.78);
  color: #e2e8f0;
}

.status-page__code {
  margin: 0;
  font-size: 48px;
  font-weight: 700;
  color: #7dd3fc;
}

.status-page__panel a {
  color: #93c5fd;
}
</style>
