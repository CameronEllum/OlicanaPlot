<script lang="ts">
    import { onMount, onDestroy } from "svelte";

    interface Props {
        lat: number;
        lng: number;
        onSelect: (lat: number, lng: number) => void;
    }

    let { lat, lng, onSelect }: Props = $props();
    let mapElement: HTMLElement;
    let map: any;
    let marker: any;
    let L: any;

    onMount(async () => {
        // Load Leaflet dynamically if not already present
        if (!(window as any).L) {
            const link = document.createElement("link");
            link.rel = "stylesheet";
            link.href = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.css";
            document.head.appendChild(link);

            await new Promise((resolve) => {
                const script = document.createElement("script");
                script.src = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.js";
                script.onload = resolve;
                document.head.appendChild(script);
            });
        }

        L = (window as any).L;

        // Center of Canada roughly
        const initialLat = lat || 56.1304;
        const initialLng = lng || -106.3468;
        const initialZoom = lat ? 8 : 3;

        map = L.map(mapElement).setView([initialLat, initialLng], initialZoom);

        L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
            attribution: "© OpenStreetMap contributors",
        }).addTo(map);

        if (lat && lng) {
            marker = L.marker([lat, lng]).addTo(map);
        }

        map.on("click", (e: any) => {
            const { lat, lng } = e.latlng;
            if (marker) {
                marker.setLatLng(e.latlng);
            } else {
                marker = L.marker(e.latlng).addTo(map);
            }
            onSelect(lat, lng);
        });
    });

    onDestroy(() => {
        if (map) {
            map.remove();
        }
    });
</script>

<div class="map-container">
    <div bind:this={mapElement} class="map-frame"></div>
</div>

<style>
    .map-container {
        width: 100%;
        height: 300px;
        margin-bottom: 1rem;
        border-radius: 8px;
        overflow: hidden;
        border: 1px solid var(--border-color);
        background: #f0f0f0;
    }
    .map-frame {
        width: 100%;
        height: 100%;
    }
</style>
