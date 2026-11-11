(function () {
    var canvas = document.getElementById('chart-setoran');
    if (!canvas) return;

    var fallback = document.getElementById('chart-fallback');
    if (typeof Chart === 'undefined') {
        if (fallback) fallback.hidden = false;
        return;
    }

    var labels = [], values = [];
    try {
        labels = JSON.parse(canvas.dataset.labels);
        values = JSON.parse(canvas.dataset.values);
    } catch (e) {
        if (fallback) fallback.hidden = false;
        return;
    }

    var rupiah = function (v) { return 'Rp ' + Math.round(v).toLocaleString('id-ID'); };

    new Chart(canvas, {
        type: 'bar',
        data: {
            labels: labels,
            datasets: [{
                label: 'Nilai setoran',
                data: values,
                backgroundColor: '#43a047',
                borderRadius: 4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: { display: false },
                tooltip: { callbacks: { label: function (ctx) { return rupiah(ctx.parsed.y); } } }
            },
            scales: {
                y: { beginAtZero: true, ticks: { callback: function (v) { return rupiah(v); } } }
            }
        }
    });
})();