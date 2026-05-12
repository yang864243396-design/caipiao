import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import './styles/element-plus-theme.css'
import './styles/el-table-layout.css'
import './styles/global.css'
import './styles/detail-tab-rg.css'

const app = createApp(App)
app.use(router)
app.mount('#app')
