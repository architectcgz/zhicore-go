<template>
  <section class="login-page">
    <div class="login-page__panel">
      <p class="login-page__eyebrow">Starter Auth Flow</p>
      <h2>登录 __APP_NAME__</h2>
      <p class="login-page__description">
        这里保留最小登录页面骨架，真实项目可在 feature 内继续补 MFA、组织切换、验证码等流程。
      </p>

      <form
        class="login-page__form"
        @submit.prevent="submit"
      >
        <label>
          <span>用户名</span>
          <input
            v-model="username"
            type="text"
            autocomplete="username"
            placeholder="student.demo"
          />
        </label>
        <label>
          <span>密码</span>
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            placeholder="请输入密码"
          />
        </label>

        <p
          v-if="errorMessage"
          class="login-page__error"
        >
          {{ errorMessage }}
        </p>

        <button
          type="submit"
          :disabled="submitting"
        >
          {{ submitting ? '登录中...' : '登录' }}
        </button>
      </form>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useLoginForm } from '@/features/auth/model/useLoginForm'

const { username, password, submitting, errorMessage, submit } = useLoginForm()
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 24px;
  background:
    radial-gradient(circle at top, rgba(56, 189, 248, 0.18), transparent 30%),
    linear-gradient(180deg, var(--color-bg-elevated) 0%, var(--color-bg-elevated-2) 100%);
}

.login-page__panel {
  width: min(420px, 100%);
  padding: 28px;
  border-radius: 24px;
  border: 1px solid var(--color-border);
  background: var(--color-panel);
  box-shadow: var(--shadow-panel);
}

.login-page__eyebrow {
  margin: 0 0 8px;
  font-size: 12px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--color-primary-soft-2);
}

.login-page__description {
  color: var(--color-text-soft);
}

.login-page__form {
  display: grid;
  gap: 16px;
  margin-top: 24px;
}

.login-page__form label {
  display: grid;
  gap: 8px;
}

.login-page__form input {
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid var(--color-border);
  background: var(--color-panel-soft);
  color: var(--color-text);
}

.login-page__form button {
  padding: 12px 16px;
  border: 0;
  border-radius: 14px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-strong));
  color: white;
  cursor: pointer;
}

.login-page__error {
  margin: 0;
  color: var(--color-danger);
}
</style>
