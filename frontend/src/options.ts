// Entry point for the options dialog. Manage theme initialization and mount the 
// OptionsWindow component.
import './theme.css';
import { mount } from 'svelte'
import OptionsWindow from './OptionsWindow.svelte'
import * as ConfigService from "../bindings/olicanaplot/internal/appconfig/configservice";

// Retrieve and apply the user's preferred theme (light/dark) from the
// configuration service on application startup.
ConfigService.GetTheme().then((theme: string) => {
    if (theme === "dark") {
        document.documentElement.classList.add('dark-mode');
    } else {
        document.documentElement.classList.remove('dark-mode');
    }
});

// Identify the root DOM element where the options window will be mounted.
const target = document.getElementById('app');
if (!target) throw new Error("No target element found");

// Create and mount the Svelte options window instance.
const app = mount(OptionsWindow, {
    target,
})

export default app
