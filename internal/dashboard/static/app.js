(() => {
  const SVG_NS = "http://www.w3.org/2000/svg";
  const $ = (id) => document.getElementById(id);

  // =========================================================
  // pixel-art house sprite (archetype-aware, per-home palette)
  // =========================================================
  // Shade a hex color by a percentage. Negative = darker, positive = lighter.
  function shade(hex, pct) {
    const h = (hex || "#888").replace("#", "");
    const r = parseInt(h.substring(0, 2), 16);
    const g = parseInt(h.substring(2, 4), 16);
    const b = parseInt(h.substring(4, 6), 16);
    const m = (v) => Math.max(0, Math.min(255, Math.round(v * (1 + pct))));
    return "#" + [m(r), m(g), m(b)].map((x) => x.toString(16).padStart(2, "0")).join("");
  }

  // spec: { shape, wall, roof, door, solar_cells, has_ev }
  function makeHouseSprite(spec, socFrac, solarActive) {
    const S = 3;
    const svg = document.createElementNS(SVG_NS, "svg");
    svg.setAttribute("width", String(16 * S));
    svg.setAttribute("height", String(14 * S));
    svg.setAttribute("viewBox", `0 0 ${16 * S} ${14 * S}`);
    svg.setAttribute("shape-rendering", "crispEdges");
    svg.classList.add("tile-sprite");
    const r = (x, y, w, h, fill) => {
      const e = document.createElementNS(SVG_NS, "rect");
      e.setAttribute("x", x * S);
      e.setAttribute("y", y * S);
      e.setAttribute("width", w * S);
      e.setAttribute("height", h * S);
      e.setAttribute("fill", fill);
      svg.appendChild(e);
    };

    const shape = (spec && spec.shape) || "standard";
    const wall = (spec && spec.wall) || "#d9c99e";
    const roof = (spec && spec.roof) || "#8b2e2b";
    const door = (spec && spec.door) || "#6f4a2a";
    const cells = (spec && spec.solar_cells) || 0;
    const hasEV = !!(spec && spec.has_ev);

    switch (shape) {
      case "apartment": drawApartment(r, wall, roof, door, cells); break;
      case "big":       drawBig(r, wall, roof, door, cells); break;
      case "cabin":     drawCabin(r, wall, roof, door, cells); break;
      case "ranch":     drawRanch(r, wall, roof, door, cells); break;
      case "standard":
      default:          drawStandard(r, wall, roof, door, cells);
    }

    // solar glow indicator (small yellow dots when producing)
    if (solarActive) {
      r(0, 0, 1, 1, "#f2c043");
      r(1, 1, 1, 1, "#f2c043");
    }

    // EV accessory (tiny car to the left of the house)
    if (hasEV) drawEV(r, 0, 10);

    // battery bar at bottom of the sprite
    drawBatteryBar(r, 4, 13, 8, socFrac);
    return svg;
  }

  function drawSolarStrip(r, x, y, w, cells) {
    if (cells <= 0) return;
    const panelW = Math.max(1, Math.min(w, Math.round(w * cells / 4)));
    r(x, y, panelW, 1, "#1c3a5c");
    for (let i = 0; i < panelW; i += 2) r(x + i, y, 1, 1, "#2c5a85");
  }

  function drawBatteryBar(r, x, y, w, frac) {
    const cells = Math.min(w, 8);
    r(x - 1, y, cells + 2, 1, "#06090f");
    const lit = Math.max(0, Math.min(cells, Math.round(frac * cells)));
    for (let i = 0; i < cells; i++) {
      const c = i < lit ? (frac > 0.3 ? "#3fe06a" : "#f2c043") : "#19232e";
      r(x + i, y, 1, 1, c);
    }
  }

  function drawEV(r, x, y) {
    // small hatchback
    r(x + 1, y + 1, 4, 1, shade("#d0d0d8", -0.1));
    r(x, y + 2, 6, 1, "#d0d0d8");
    r(x + 1, y + 1, 1, 1, "#6fb6ff"); // windshield hint
    r(x + 1, y + 3, 1, 1, "#222");    // wheels
    r(x + 4, y + 3, 1, 1, "#222");
  }

  function drawStandard(r, wall, roof, door, cells) {
    r(11, 1, 2, 2, "#5a2f1a");                              // chimney
    r(5, 3, 6, 1, shade(roof, -0.25));
    r(4, 4, 8, 1, roof);
    r(3, 5, 10, 1, roof);
    r(2, 6, 12, 1, roof);
    r(1, 7, 14, 1, roof);
    drawSolarStrip(r, 2, 7, 12, cells);
    r(2, 8, 12, 5, wall);
    r(2, 8, 12, 1, shade(wall, -0.2));
    r(2, 8, 1, 5, shade(wall, -0.1));
    r(13, 8, 1, 5, shade(wall, -0.2));
    r(4, 9, 3, 3, "#6fb6ff");                                // window
    r(5, 9, 1, 3, "#a8d4ff");
    r(4, 10, 3, 1, "#2c5a85");
    r(9, 9, 3, 4, door);                                     // door
    r(9, 9, 3, 1, shade(door, -0.35));
    r(11, 11, 1, 1, "#f2c043");                              // knob
  }

  function drawApartment(r, wall, roof, door, cells) {
    r(1, 2, 14, 2, shade(roof, -0.15));
    drawSolarStrip(r, 2, 2, 12, cells);
    r(1, 4, 14, 9, wall);
    r(1, 4, 14, 1, shade(wall, -0.2));
    r(1, 4, 1, 9, shade(wall, -0.1));
    r(14, 4, 1, 9, shade(wall, -0.2));
    // 3 rows × 4 windows
    for (let row = 0; row < 3; row++) {
      for (let col = 0; col < 4; col++) {
        r(3 + col * 3, 5 + row * 2, 2, 1, "#6fb6ff");
      }
    }
    r(7, 10, 2, 3, door);
    r(7, 10, 2, 1, shade(door, -0.35));
    r(8, 11, 1, 1, "#f2c043");
  }

  function drawBig(r, wall, roof, door, cells) {
    r(4, 0, 2, 2, "#5a2f1a");                                // chimneys
    r(10, 0, 2, 2, "#5a2f1a");
    r(3, 2, 4, 1, shade(roof, -0.25));
    r(9, 2, 4, 1, shade(roof, -0.25));
    r(2, 3, 6, 1, roof);
    r(8, 3, 6, 1, roof);
    r(1, 4, 14, 1, roof);
    r(0, 5, 16, 1, roof);
    drawSolarStrip(r, 1, 6, 14, cells);
    r(0, 7, 16, 6, wall);
    r(0, 7, 16, 1, shade(wall, -0.2));
    r(0, 7, 1, 6, shade(wall, -0.1));
    r(15, 7, 1, 6, shade(wall, -0.2));
    r(2, 8, 2, 2, "#6fb6ff");
    r(6, 8, 2, 2, "#6fb6ff");
    r(12, 8, 2, 2, "#6fb6ff");
    r(9, 9, 2, 4, door);
    r(9, 9, 2, 1, shade(door, -0.35));
    r(10, 11, 1, 1, "#f2c043");
  }

  function drawCabin(r, wall, roof, door, cells) {
    r(6, 2, 4, 1, shade(roof, -0.25));
    r(5, 3, 6, 1, roof);
    r(4, 4, 8, 1, roof);
    r(3, 5, 10, 1, roof);
    r(2, 6, 12, 1, roof);
    drawSolarStrip(r, 3, 6, 10, Math.min(cells, 2));
    r(2, 7, 12, 5, wall);
    r(2, 7, 12, 1, shade(wall, -0.2));
    r(4, 8, 3, 3, "#6fb6ff");
    r(5, 8, 1, 3, "#a8d4ff");
    r(9, 8, 3, 4, door);
    r(9, 8, 3, 1, shade(door, -0.35));
    r(11, 10, 1, 1, "#f2c043");
  }

  function drawRanch(r, wall, roof, door, cells) {
    // low wide roof with slight peak
    r(6, 2, 4, 1, shade(roof, -0.25));
    r(2, 3, 12, 1, roof);
    r(0, 4, 16, 1, roof);
    // double solar strip (ranch style)
    drawSolarStrip(r, 0, 5, 16, cells);
    if (cells >= 3) drawSolarStrip(r, 0, 6, 16, cells - 1);
    const wallYStart = cells >= 3 ? 7 : 6;
    r(0, wallYStart, 16, 13 - wallYStart, wall);
    r(0, wallYStart, 16, 1, shade(wall, -0.2));
    r(0, wallYStart, 1, 13 - wallYStart, shade(wall, -0.1));
    r(15, wallYStart, 1, 13 - wallYStart, shade(wall, -0.2));
    // windows + door arranged along the long wall
    r(2, wallYStart + 2, 2, 2, "#6fb6ff");
    r(6, wallYStart + 2, 2, 2, "#6fb6ff");
    r(13, wallYStart + 2, 2, 2, "#6fb6ff");
    r(9, wallYStart + 1, 2, 12 - wallYStart - 1, door);
    r(9, wallYStart + 1, 2, 1, shade(door, -0.35));
    r(10, wallYStart + 3, 1, 1, "#f2c043");
  }

  // =========================================================
  // fleet render (50 tiles in a grid, reconciled each update)
  // =========================================================
  let tileEls = {}; // home_id -> el

  function renderFleet(homes) {
    const grid = $("fleet-grid");
    const ids = new Set(homes.map((h) => h.id));

    // remove missing
    for (const id of Object.keys(tileEls)) {
      if (!ids.has(id)) {
        tileEls[id].remove();
        delete tileEls[id];
      }
    }

    // add / update
    for (const h of homes) {
      let el = tileEls[h.id];
      if (!el) {
        el = document.createElement("div");
        el.className = "home-tile";
        el.dataset.id = h.id;
        el.addEventListener("click", () => openInspector(h.id));
        el.innerHTML = `
          <div class="hid">${h.id}</div>
          <div class="tile-badge">—</div>
          <div class="tile-row">
            <span class="tile-sprite-slot"></span>
            <div class="tile-stats">
              <div class="tile-batt"><div class="tile-batt-fill"></div></div>
              <div class="tile-num">
                <span class="soc">0%</span>
                <span class="gid">+0.0</span>
              </div>
            </div>
          </div>
        `;
        grid.appendChild(el);
        tileEls[h.id] = el;
      }
      // status class
      let status = "idle";
      if (!h.online) status = "offline";
      else if (h.grid_kw > 0.1) status = "importing";
      else if (h.grid_kw < -0.1) status = "exporting";
      el.classList.remove("importing", "exporting", "idle", "offline");
      el.classList.add(status);

      // update badge
      const badge = el.querySelector(".tile-badge");
      badge.textContent = status.toUpperCase();

      // battery (normalize by this home's own capacity; no-battery homes = 0)
      const cap = h.battery_capacity_kwh || 0;
      const socFrac = cap > 0 ? Math.max(0, Math.min(1, h.battery_soc_kwh / cap)) : 0;
      const fill = el.querySelector(".tile-batt-fill");
      fill.style.width = (socFrac * 100).toFixed(1) + "%";
      fill.style.background = socFrac > 0.3 ? "#3fe06a" : (socFrac > 0.1 ? "#f2c043" : "#ff5f5f");

      el.querySelector(".soc").textContent = cap > 0 ? (Math.round(socFrac * 100) + "%") : "—";
      const gv = el.querySelector(".gid");
      const g = h.grid_kw;
      gv.textContent = (g >= 0 ? "+" : "") + g.toFixed(1);
      gv.className = "gid " + (g > 0.1 ? "imp" : (g < -0.1 ? "exp" : ""));

      // sprite: archetype-aware
      const slot = el.querySelector(".tile-sprite-slot");
      slot.innerHTML = "";
      slot.appendChild(makeHouseSprite(h.sprite || {}, socFrac, h.solar_kw > 0.2));
    }
  }

  // =========================================================
  // inspector (drill-down on click)
  // =========================================================
  let inspectedID = null;
  function openInspector(id) {
    inspectedID = id;
    $("inspector").classList.remove("hidden");
    refreshInspector(currentState);
  }
  function closeInspector() {
    inspectedID = null;
    $("inspector").classList.add("hidden");
  }
  $("insp-close").addEventListener("click", closeInspector);
  $("inspector").addEventListener("click", (e) => {
    if (e.target.id === "inspector") closeInspector();
  });

  function refreshInspector(state) {
    if (!inspectedID || !state) return;
    const h = (state.fleet.homes || []).find((x) => x.id === inspectedID);
    const c = $("insp-content");
    if (!h) {
      c.innerHTML = "<div class='insp-title'>home not found</div>";
      return;
    }
    const cap = h.battery_capacity_kwh || 0;
    const socPct = cap > 0 ? Math.round(h.battery_soc_kwh / cap * 100) : 0;
    const dir = h.grid_kw > 0.1 ? "IMPORT" : h.grid_kw < -0.1 ? "EXPORT" : "IDLE";
    c.innerHTML = `
      <div class="insp-title">${h.id} <span class="insp-strategy ${h.strategy||''}">${(h.strategy||'—').toUpperCase()}</span></div>
      <div class="insp-archetype">${h.archetype || ''}</div>
      <div class="insp-grid">
        <div class="insp-cell"><div class="k">TICK</div><div class="v">${h.tick}</div></div>
        <div class="insp-cell"><div class="k">STATUS</div><div class="v">${h.online ? dir : 'OFFLINE'}</div></div>
        <div class="insp-cell"><div class="k">SOLAR NOW</div><div class="v">${h.solar_kw.toFixed(2)} kW</div></div>
        <div class="insp-cell"><div class="k">SOLAR PEAK</div><div class="v">${(h.solar_peak_kw||0).toFixed(1)} kW</div></div>
        <div class="insp-cell"><div class="k">LOAD</div><div class="v">${h.load_kw.toFixed(2)} kW</div></div>
        <div class="insp-cell"><div class="k">GRID</div><div class="v">${(h.grid_kw>=0?'+':'')}${h.grid_kw.toFixed(2)} kW</div></div>
        <div class="insp-cell"><div class="k">BATTERY</div><div class="v">${cap>0 ? (socPct+'% · '+h.battery_soc_kwh.toFixed(2)+' / '+cap.toFixed(1)+' kWh') : 'none'}</div></div>
        <div class="insp-cell"><div class="k">BATT FLOW</div><div class="v">${(h.battery_kw>=0?'+':'')}${h.battery_kw.toFixed(2)} kW</div></div>
        <div class="insp-cell"><div class="k">RPC LATENCY</div><div class="v">${h.rpc_ms.toFixed(1)} ms</div></div>
        <div class="insp-cell"><div class="k">RPCS OK</div><div class="v">${h.rpc_count}</div></div>
        <div class="insp-cell"><div class="k">TOTAL IMPORT</div><div class="v">${h.total_import_kwh.toFixed(2)} kWh</div></div>
        <div class="insp-cell"><div class="k">TOTAL EXPORT</div><div class="v">${h.total_export_kwh.toFixed(2)} kWh</div></div>
        <div class="insp-cell" style="grid-column: span 2;"><div class="k">RUNNING COST</div>
          <div class="v" style="color:${h.total_cost>=0?'var(--amber)':'var(--green)'}">
            ${h.total_cost>=0?'$':'-$'}${Math.abs(h.total_cost).toFixed(2)}
          </div>
        </div>
      </div>
    `;
  }

  // =========================================================
  // chart
  // =========================================================
  Chart.defaults.font.family = "VT323, monospace";
  Chart.defaults.font.size = 13;
  const chart = new Chart($("chart").getContext("2d"), {
    type: "line",
    data: {
      labels: [],
      datasets: [
        { label: "Price $/kWh", data: [], borderColor: "#f2c043",
          backgroundColor: "rgba(242,192,67,0.10)",
          yAxisID: "yP", pointRadius: 0, borderWidth: 2, tension: 0.25 },
        { label: "Agg Load kW", data: [], borderColor: "#ff5f5f",
          yAxisID: "yK", pointRadius: 0, borderWidth: 1.6, tension: 0.25 },
        { label: "Agg Solar kW", data: [], borderColor: "#3fe06a",
          yAxisID: "yK", pointRadius: 0, borderWidth: 1.6, tension: 0.25 },
        { label: "Net Grid kW", data: [], borderColor: "#58a6ff",
          backgroundColor: "rgba(88,166,255,0.08)", yAxisID: "yK", fill: true,
          pointRadius: 0, borderWidth: 1.8, tension: 0.25 },
      ],
    },
    options: {
      animation: false, responsive: true, maintainAspectRatio: false,
      scales: {
        x: { ticks: { color: "#7d8590", maxTicksLimit: 8 }, grid: { color: "#1a222c" } },
        yP: { position: "right", ticks: { color: "#f2c043", callback: (v) => "$" + v.toFixed(2) }, grid: { drawOnChartArea: false } },
        yK: { position: "left", ticks: { color: "#7d8590", callback: (v) => v + " kW" }, grid: { color: "#1a222c" } },
      },
      plugins: { legend: { labels: { color: "#e6edf3", boxWidth: 14, font: { size: 13 } } } },
    },
  });
  function renderChart(history) {
    chart.data.labels = history.map((h) => h.tick);
    chart.data.datasets[0].data = history.map((h) => h.price);
    chart.data.datasets[1].data = history.map((h) => h.agg_load_kw);
    chart.data.datasets[2].data = history.map((h) => h.agg_solar_kw);
    chart.data.datasets[3].data = history.map((h) => h.agg_grid_kw);
    chart.update("none");
  }

  // =========================================================
  // RPC log
  // =========================================================
  function renderRPCLog(evts) {
    const box = $("rpc-log");
    box.innerHTML = "";
    // newest first, cap at 30
    const show = (evts || []).slice(-30).reverse();
    for (const e of show) {
      const line = document.createElement("div");
      line.className = "rpc-line" + (e.ok ? "" : " err");
      const d = new Date(e.at);
      const ts = d.toLocaleTimeString([], { hour12: false });
      const msg = e.ok
        ? `bat=${(e.battery_kw>=0?'+':'')}${e.battery_kw.toFixed(2)} grid=${(e.grid_kw>=0?'+':'')}${e.grid_kw.toFixed(2)}`
        : (e.err || "RPC failed");
      line.innerHTML = `
        <span class="ts">${ts.slice(3)}</span>
        <span class="hid">${e.home_id}</span>
        <span class="lat">${e.latency_ms.toFixed(1)} ms</span>
        <span class="msg">${msg}</span>
      `;
      box.appendChild(line);
    }
  }

  // =========================================================
  // server + fleet header
  // =========================================================
  function fmtHour(h) {
    const hh = Math.floor(h), mm = Math.floor((h - hh) * 60);
    return String(hh).padStart(2, "0") + ":" + String(mm).padStart(2, "0");
  }
  function fmtUptime(s) {
    s = Math.floor(s);
    if (s < 60) return s + "s";
    const m = Math.floor(s / 60), r = s % 60;
    if (m < 60) return m + "m " + r + "s";
    const h = Math.floor(m / 60), rm = m % 60;
    return h + "h " + rm + "m";
  }

  let currentState = null;
  function applyState(st) {
    currentState = st;
    const f = st.fleet || {};
    const srv = st.server || {};
    $("day").textContent = f.day ?? 0;
    $("tick").textContent = f.tick ?? 0;
    $("hour").textContent = fmtHour(f.hour ?? 0);
    $("price").textContent = "$" + (f.price ?? 0).toFixed(3);
    $("homes-count").textContent = (f.homes || []).length;

    $("srv-strategy").textContent = (srv.strategy || "—").toUpperCase();
    $("srv-rate").textContent = (srv.rate_per_sec ?? 0).toFixed(1);
    $("srv-latency").textContent = (srv.avg_latency_ms ?? 0).toFixed(2) + " ms";
    $("srv-active").textContent = srv.active_calls ?? 0;
    $("srv-total").textContent = (srv.total_rpcs ?? 0).toLocaleString();
    $("srv-uptime").textContent = fmtUptime(srv.uptime_sec ?? 0);

    const agg = f.fleet_now || {};
    $("agg-grid").textContent = (agg.agg_grid_kw >= 0 ? "+" : "") + (agg.agg_grid_kw || 0).toFixed(1) + " kW";
    $("agg-grid").style.color = (agg.agg_grid_kw || 0) >= 0 ? "var(--amber)" : "var(--green)";
    $("agg-solar").textContent = (agg.agg_solar_kw || 0).toFixed(1) + " kW";
    $("agg-load").textContent = (agg.agg_load_kw || 0).toFixed(1) + " kW";
    $("agg-soc").textContent = Math.round(agg.avg_soc_pct || 0) + "%";

    renderFleet(f.homes || []);
    renderChart(f.history || []);
    renderRPCLog(f.recent_rpcs || []);
    if (inspectedID) refreshInspector(st);
  }

  // =========================================================
  // websocket
  // =========================================================
  function connect() {
    const proto = location.protocol === "https:" ? "wss" : "ws";
    const ws = new WebSocket(`${proto}://${location.host}/api/ws`);
    ws.onmessage = (ev) => {
      try { applyState(JSON.parse(ev.data)); }
      catch (e) { console.error(e); }
    };
    ws.onclose = () => setTimeout(connect, 500);
  }
  connect();

  // =========================================================
  // controls
  // =========================================================
  document.querySelectorAll(".speed button").forEach((b) => {
    b.addEventListener("click", () => {
      document.querySelectorAll(".speed button").forEach((x) => x.classList.remove("active"));
      b.classList.add("active");
      fetch(`/api/speed?x=${b.dataset.speed}`, { method: "POST" });
    });
  });
  let paused = false;
  $("pause-btn").addEventListener("click", () => {
    paused = !paused;
    $("pause-btn").textContent = paused ? "> RESUME" : "|| PAUSE";
    fetch(`/api/${paused ? "pause" : "resume"}`, { method: "POST" });
  });

})();
