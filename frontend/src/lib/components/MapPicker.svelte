<script lang="ts">
    import { onMount, onDestroy } from "svelte";

    interface Marker {
        lat: number;
        lng: number;
        popup?: string;
    }

    interface Props {
        lat?: number;
        lng?: number;
        markers?: Marker[];
        onSelect?: (lat: number, lng: number) => void;
    }

    let { lat, lng, markers = [], onSelect }: Props = $props();
    let mapElement: HTMLElement;
    let map: any;
    let primaryMarker: any;
    let markerLayerGroup: any;
    let L: any;

    $effect(() => {
        if (!map || !L || !markerLayerGroup) return;

        // Clear all layers
        markerLayerGroup.clearLayers();
        primaryMarker = null;

        const bounds = L.latLngBounds();

        // Single primary marker
        if (lat && lng) {
            const latLng = [lat, lng];
            primaryMarker = L.marker(latLng).addTo(markerLayerGroup);
            bounds.extend(latLng);
        }

        // Multiple markers
        if (Array.isArray(markers) && markers.length > 0) {
            for (const m of markers) {
                if (m.lat && m.lng) {
                    const latLng = [m.lat, m.lng];
                    const mrk = L.marker(latLng);
                    if (m.popup) {
                        mrk.bindPopup(m.popup);
                    }
                    markerLayerGroup.addLayer(mrk);
                    bounds.extend(latLng);
                }
            }
        }

        if (bounds.isValid()) {
            map.fitBounds(bounds, { padding: [30, 30], maxZoom: 12 });
        }
    });

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

        // Fix for missing default icon in some bundlers
        L.Icon.Default.imagePath =
            "https://unpkg.com/leaflet@1.9.4/dist/images/";

        // Center of Canada roughly
        const initialLat = lat || 56.1304;
        const initialLng = lng || -106.3468;
        const initialZoom = lat ? 8 : 3;

        map = L.map(mapElement).setView([initialLat, initialLng], initialZoom);

        L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
            attribution: "© OpenStreetMap contributors",
        }).addTo(map);

        markerLayerGroup = L.layerGroup().addTo(map);

        map.on("click", (e: any) => {
            if (onSelect) {
                const { lat, lng } = e.latlng;
                onSelect(lat, lng);
            }
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
