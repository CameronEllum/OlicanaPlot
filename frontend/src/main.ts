// Main entry point for the frontend application. This script initializes the
// global styles and mounts the root Svelte component to the DOM.
import './theme.css';
import { mount } from 'svelte'
import App from './App.svelte'

// Identify the root DOM element where the application will be injected.
const target = document.getElementById('app');
if (!target) throw new Error("No target element found");

// Create and mount the Svelte application instance.
const app = mount(App, {
    target,
})

export default app
