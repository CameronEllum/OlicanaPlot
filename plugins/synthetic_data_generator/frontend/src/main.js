import { SyntheticService } from "../bindings/synthetic-ipc";

// Parameter visibility based on simulation type
const paramConfig = {
    'Random Walk': ['param-noise'],
    'Gauss-Markov': ['param-noise', 'param-correlation'],
    'Sinusoidal': ['param-amplitude', 'param-frequency']
};

function updateVisibleParams(simType) {
    document.querySelectorAll('.param-item').forEach(el => {
        el.classList.remove('visible');
    });

    const visibleParams = paramConfig[simType] || [];
    visibleParams.forEach(paramId => {
        const el = document.getElementById(paramId);
        if (el) el.classList.add('visible');
    });
}

function formatNumber(num) {
    return num.toLocaleString();
}

function updateTotalPoints() {
    const order = parseInt(document.getElementById('order').value);
    const multiplier = parseInt(document.getElementById('multiplier').value);
    const total = multiplier * Math.pow(10, order);
    document.getElementById('total-points').textContent = formatNumber(total);
    document.getElementById('order-value').innerHTML = '10<sup>' + order + '</sup>';
    document.getElementById('multiplier-value').textContent = multiplier;
}

document.getElementById('order').addEventListener('input', updateTotalPoints);
document.getElementById('multiplier').addEventListener('input', updateTotalPoints);
updateTotalPoints();

// Sliders and display values
const sliders = [
    { id: 'num-series', valueId: 'series-value', format: v => parseInt(v).toString() },
    { id: 'noise', valueId: 'noise-value', format: v => v.toFixed(1) },
    { id: 'correlation', valueId: 'correlation-value', format: v => parseFloat(v).toFixed(1) },
    { id: 'amplitude', valueId: 'amplitude-value', format: v => parseFloat(v).toFixed(1) },
    { id: 'frequency', valueId: 'frequency-value', format: v => parseFloat(v).toFixed(2) + ' Hz' }
];

sliders.forEach(({ id, valueId, format }) => {
    const slider = document.getElementById(id);
    const valueDisplay = document.getElementById(valueId);
    if (slider && valueDisplay) {
        slider.addEventListener('input', () => {
            valueDisplay.textContent = format(parseFloat(slider.value));
        });
    }
});

document.getElementById('simulation-type').addEventListener('change', (e) => {
    updateVisibleParams(e.target.value);
});

updateVisibleParams(document.getElementById('simulation-type').value);

document.getElementById('synthetic-form').addEventListener('submit', async (e) => {
    e.preventDefault();

    const simulationType = document.getElementById('simulation-type').value;
    const order = parseInt(document.getElementById('order').value);
    const multiplier = parseInt(document.getElementById('multiplier').value);
    const numPoints = multiplier * Math.pow(10, order);
    const numSeries = parseInt(document.getElementById('num-series').value);
    const noise = parseFloat(document.getElementById('noise').value);
    const correlationTime = parseFloat(document.getElementById('correlation').value);
    const amplitude = parseFloat(document.getElementById('amplitude').value);
    const frequency = parseFloat(document.getElementById('frequency').value);

    try {
        await SyntheticService.Submit(simulationType, numPoints, numSeries, noise, correlationTime, amplitude, frequency);
    } catch (err) {
        console.error('Submit error:', err);
    }
});

document.getElementById('cancel-btn').addEventListener('click', async () => {
    try {
        await SyntheticService.Cancel();
    } catch (err) {
        console.error('Cancel error:', err);
    }
});

document.getElementById('close-btn').addEventListener('click', async () => {
    try {
        await SyntheticService.Cancel();
    } catch (err) {
        console.error('Cancel error:', err);
    }
});
