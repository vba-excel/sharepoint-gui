import './style.css'
// @ts-ignore
import App from './App.svelte'
import './styles/global.css';   // <— importa o CSS global
import './styles/global-dark.css'; // importa as overrides (aplicam quando data-theme='dark')

const app = new App({
  target: document.getElementById('app')
})

export default app
