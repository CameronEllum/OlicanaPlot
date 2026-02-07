import './theme.css';
import { mount } from 'svelte'
import OptionsWindow from './OptionsWindow.svelte'
import * as ConfigService from "../bindings/olicanaplot/internal/appconfig/configservice";

// Apply initial theme
ConfigService.GetTheme().then((theme: string) => {
    if (theme === "dark") {
        document.documentElement.classList.add('dark-mode');
    } else {
        document.documentElement.classList.remove('dark-mode');
    }
});

const target = document.getElementById('app');
if (!target) throw new Error("No target element found");

const app = mount(OptionsWindow, {
    target,
})

export default app
