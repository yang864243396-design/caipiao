import { createApp } from 'vue'
import App from './App.vue'
import { createRouter, createWebHistory } from 'vue-router'
import AdminHome from './views/AdminHome.vue'

const app = createApp(App)
app.use(
  createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes: [{ path: '/', name: 'home', component: AdminHome }],
  }),
)
app.mount('#app')
