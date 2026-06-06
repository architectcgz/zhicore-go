<template>
  <section class="dashboard-page">
    <header class="dashboard-page__header">
      <div>
        <p class="dashboard-page__eyebrow">Widget + Entity Sample</p>
        <h2>Student Dashboard</h2>
      </div>
    </header>
    <StudentOverviewWidget />
  </section>
</template>

<script setup lang="ts">
import StudentOverviewWidget from '@/widgets/dashboard/StudentOverviewWidget.vue'
</script>

<style scoped>
.dashboard-page {
  display: grid;
  gap: 20px;
}

.dashboard-page__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.dashboard-page__eyebrow {
  margin: 0 0 6px;
  font-size: 12px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #7dd3fc;
}
</style>
