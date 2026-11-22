// Alpine component that renders a Chart.js chart from the JSON block that
// templ renders next to the canvas. Alpine calls destroy() when HTMX swaps
// the widget area out, and init() again for the new markup.
(function () {
  var rupiah = new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 });
  function rp(v) { return 'Rp' + rupiah.format(v); }
  function rpShort(v) {
    var abs = Math.abs(v);
    if (abs >= 1e9) return 'Rp' + rupiah.format(Math.round(v / 1e8) / 10) + ' M';
    if (abs >= 1e6) return 'Rp' + rupiah.format(Math.round(v / 1e5) / 10) + ' jt';
    if (abs >= 1e3) return 'Rp' + rupiah.format(Math.round(v / 1e3)) + ' rb';
    return rp(v);
  }

  var colors = ['#059669', '#0ea5e9', '#f59e0b', '#6366f1', '#ef4444', '#14b8a6', '#a855f7', '#84cc16', '#ec4899', '#64748b'];

  function config(kind, data) {
    var values = data.values || [];
    var labels = data.labels || [];
    var tooltip = { callbacks: { label: function (ctx) { return ' ' + rp(ctx.parsed[kind === 'hbar' ? 'x' : 'y']); } } };
    var base = {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      plugins: { legend: { display: false }, tooltip: tooltip },
    };

    if (kind === 'line') {
      base.scales = { y: { beginAtZero: true, ticks: { callback: rpShort } } };
      return {
        type: 'line',
        data: { labels: labels, datasets: [{ label: 'Pendapatan Bersih', data: values, borderColor: '#059669', backgroundColor: 'rgba(5,150,105,0.12)', fill: true, tension: 0.25, pointRadius: 3 }] },
        options: base,
      };
    }

    if (kind === 'hbar') {
      base.indexAxis = 'y';
      base.scales = {
        x: { beginAtZero: true, ticks: { callback: rpShort } },
        y: { ticks: { callback: function (v) { var l = this.getLabelForValue(v); return l.length > 32 ? l.slice(0, 31) + '…' : l; } } },
      };
      return {
        type: 'bar',
        data: { labels: labels, datasets: [{ label: 'Pendapatan Bersih', data: values, backgroundColor: '#059669' }] },
        options: base,
      };
    }

    base.scales = { y: { beginAtZero: true, ticks: { callback: rpShort } } };
    return {
      type: 'bar',
      data: { labels: labels, datasets: [{ label: 'Pendapatan Bersih', data: values, backgroundColor: labels.map(function (_, i) { return colors[i % colors.length]; }) }] },
      options: base,
    };
  }

  document.addEventListener('alpine:init', function () {
    window.Alpine.data('chart', function (id, kind) {
      var chart = null; // kept outside Alpine's reactive proxy on purpose
      return {
        init: function () {
          var el = document.getElementById(id + '-data');
          if (!el || !window.Chart) return;
          var data = JSON.parse(el.textContent || '{}');
          chart = new window.Chart(this.$refs.canvas, config(kind, data));
        },
        destroy: function () {
          if (chart) { chart.destroy(); chart = null; }
        },
      };
    });
  });
})();
