import { createApp } from 'vue'
import App from './App.vue'
import { router } from './router'
import { initTheme } from './store'
import '@fontsource/poppins/500.css'
import '@fontsource/poppins/600.css'
import '@fontsource/poppins/700.css'
import '@fontsource/open-sans/400.css'
import '@fontsource/open-sans/600.css'
import '@fontsource/open-sans/700.css'
import './styles.css'

initTheme()
createApp(App).use(router).mount('#app')
