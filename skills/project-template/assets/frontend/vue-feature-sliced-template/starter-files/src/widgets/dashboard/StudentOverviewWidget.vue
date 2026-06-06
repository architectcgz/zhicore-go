<template>
  <section class="overview-widget">
    <article
      v-for="card in cards"
      :key="card.title"
      class="overview-widget__card"
    >
      <p>{{ card.title }}</p>
      <strong>{{ card.value }}</strong>
      <span>{{ card.description }}</span>
    </article>
  </section>
</template>

<script setup lang="ts">
const cards = [
  { title: 'Entities', value: '1', description: '最小实体层样例已接入页面。' },
  { title: 'Widgets', value: '1', description: '页面级组合块应在这里收敛。' },
  { title: 'Features', value: '2', description: 'layout 与 auth 流程边界已分开。' },
]
</script>

<style scoped>
.overview-widget {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
}

.overview-widget__card {
  display: grid;
  gap: 10px;
  padding: 18px;
  border-radius: 20px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.56);
}

.overview-widget__card p,
.overview-widget__card span {
  margin: 0;
  color: #94a3b8;
}

.overview-widget__card strong {
  font-size: 32px;
  color: #f8fafc;
}
</style>
