import './theme.css';
import { mount } from 'svelte'
import OptionsWindow from './OptionsWindow.svelte'
import * as ConfigService from "../bindings/olicanaplot/internal/appconfig/configservice";

// Apply initial theme
ConfigService.GetTheme().then((theme) => {
    if (theme === "dark") {
        document.documentElement.classList.add('dark-mode');
    } else {
        document.documentElement.classList.remove('dark-mode');
    }
});

const app = mount(OptionsWindow, {
    target: document.getElementById('app'),
})

export default app
