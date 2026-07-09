import { createApp } from 'vue'

// 编程式组件（ElMessage/ElMessageBox/ElNotification/ElLoading）样式不会被按需自动引入，需显式加载
import 'element-plus/theme-chalk/el-message.css'
import 'element-plus/theme-chalk/el-message-box.css'
import 'element-plus/theme-chalk/el-notification.css'
import 'element-plus/theme-chalk/el-loading.css'

import '@/styles/admin-theme.css'
import '@/styles/el-table-layout.css'
import '@/styles/admin-dialog.css'

import App from './App.vue'
import { router } from '@/router/index'
import { createPinia } from 'pinia'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.mount('#app')
