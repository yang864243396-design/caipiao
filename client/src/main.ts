import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
// ElMessage 为命令式 API（各处显式 import），按需插件不会注入其样式，须手动引入
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/message-box/style/css'
import './styles/element-plus-theme.css'
import './styles/el-table-layout.css'
import './styles/global.css'
import './styles/cms-rich-html.css'
import './styles/member-subpage-shell.css'
import './styles/detail-tab-rg.css'
import './styles/layout-mobile.css'
import './styles/layout-web.css'
import { initLayoutMode } from '@/composables/useLayoutMode'

initLayoutMode()

const app = createApp(App)
app.use(router)
app.mount('#app')
