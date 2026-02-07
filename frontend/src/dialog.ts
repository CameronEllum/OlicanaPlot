import './theme.css';
import { mount } from 'svelte'
import SchemaForm from './lib/SchemaForm.svelte'
import { Events } from "@wailsio/runtime";
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

const params = new URLSearchParams(window.location.search);
const requestID = params.get('requestID');
const title = params.get('title') || 'Plugin Configuration';

function handleFormSubmit(data: any) {
    Events.Emit(`ipc-form-result-${requestID}`, data);
    setTimeout(() => {
        (window as any).wails?.Window?.Close();
    }, 100);
}

function handleFormCancel() {
    Events.Emit(`ipc-form-result-${requestID}`, "error:cancelled");
    setTimeout(() => {
        (window as any).wails?.Window?.Close();
    }, 100);
}

let app: any;

// Listen for the initial data
Events.On(`ipc-form-init-${requestID}`, (e: any) => {
    const data = e.data || e;
    const schema = data.schema || {};
    const uiSchema = data.uiSchema || {};
    const initialData = data.data || {};
    const handleFormChange = data.handleFormChange || false;

    if (app) return; // Only mount once

    app = mount(SchemaForm, {
        target: target!,
        props: {
            schema,
            uiSchema,
            initialData,
            title,
            requestID: requestID || "",
            handleFormChange,
            onsubmit: handleFormSubmit,
            oncancel: handleFormCancel
        }
    })
});

// Signal that we are ready to receive data
Events.Emit(`ipc-form-ready-${requestID}`);

export default app
