import { createPinia } from 'pinia'
import { createApp } from 'vue'

import App from './App.vue'
import router from './router'
import { createDefaultErrorRuntimeOptions, setupGlobalErrorRuntime } from './runtime/globalErrorRuntime'
import './style.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
setupGlobalErrorRuntime(app, router, pinia, createDefaultErrorRuntimeOptions())
app.mount('#app')
