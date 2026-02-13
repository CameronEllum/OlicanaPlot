// Imports for application components, Svelte mounting, and backend bindings.
import './theme.css';
import { mount } from 'svelte'
import SchemaForm from './lib/components/SchemaForm.svelte'
import { Events } from "@wailsio/runtime";
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

// Setup target element and parse configuration from URL query parameters.
const target = document.getElementById('app');
if (!target) throw new Error("No target element found");

const params = new URLSearchParams(window.location.search);
const requestID = params.get('requestID');
const title = params.get('title') || 'Plugin Configuration';

// Emit the form data result back to the main process via an event and close
// the window.
function handleFormSubmit(data: any) {
    Events.Emit(`ipc-form-result-${requestID}`, data);
    setTimeout(() => {
        (window as any).wails?.Window?.Close();
    }, 100);
}

// Emit a cancellation error message back to the main process and close the
// window.
function handleFormCancel() {
    Events.Emit(`ipc-form-result-${requestID}`, "error:cancelled");
    setTimeout(() => {
        (window as any).wails?.Window?.Close();
    }, 100);
}

let app: any;

// Listen for the initial data transmission to initialize the dynamic form.
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

// Signal that the dialog is ready to receive the initialization payload
// from the backend.
Events.Emit(`ipc-form-ready-${requestID}`);

export default app
