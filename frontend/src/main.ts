// Application entry point. Mounts the root Svelte component into the DOM.

import './style.scss'
import App from './App.svelte'
import { mount } from 'svelte'

mount(App, {
  target: document.getElementById('app')!
})
