import './theme.css';
import { mount } from 'svelte'
import SchemaForm from './lib/SchemaForm.svelte'
import { Events } from "@wailsio/runtime";
import * as ConfigService from "../bindings/olicanaplot/internal/appconfig/configservice";

// Apply initial theme
ConfigService.GetTheme().then((theme) => {
    if (theme === "dark") {
        document.documentElement.classList.add('dark-mode');
    } else {
        document.documentElement.classList.remove('dark-mode');
    }
});

const target = document.getElementById('app');

const params = new URLSearchParams(window.location.search);
const requestID = params.get('requestID');
const title = params.get('title') || 'Plugin Configuration';

function handleFormSubmit(data) {
    Events.Emit(`ipc-form-result-${requestID}`, data);
    setTimeout(() => {
        window.wails?.Window?.Close();
    }, 100);
}

function handleFormCancel() {
    Events.Emit(`ipc-form-result-${requestID}`, "error:cancelled");
    setTimeout(() => {
        window.wails?.Window?.Close();
    }, 100);
}

let app;

// Listen for the initial data
Events.On(`ipc-form-init-${requestID}`, (e) => {
    const data = e.data || e;
    const schema = data.schema || {};
    const uiSchema = data.uiSchema || {};
    const initialData = data.data || {};
    const handleFormChange = data.handleFormChange || false;

    if (app) return; // Only mount once

    app = mount(SchemaForm, {
        target: target,
        props: {
            schema,
            uiSchema,
            initialData,
            title,
            requestID,
            handleFormChange,
            onsubmit: handleFormSubmit,
            oncancel: handleFormCancel
        }
    })
});

// Signal that we are ready to receive data
Events.Emit(`ipc-form-ready-${requestID}`);

export default app
