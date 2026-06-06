<template>
  <AppLayout :logout="logout" />
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

import { useLayoutSessionActionsBridge } from '@/features/layout'
import AppLayout from '@/shared/ui/layout/AppLayout.vue'

const router = useRouter()
const { logout } = useLayoutSessionActionsBridge(
  () => { void router.push('__DEFAULT_LOGIN_PATH__') }
)
</script>
