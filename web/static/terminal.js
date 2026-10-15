(function () {
  const root = document.querySelector(".lesson");
  const api = root.dataset.api;
  const output = document.getElementById("output");
  const verdict = document.getElementById("verdict");

  const term = new Terminal({
    cursorBlink: true,
    fontFamily: '"JetBrains Mono", ui-monospace, Menlo, Consolas, monospace',
    fontSize: 14,
    theme: { background: "#0d0e14" },
  });
  const fit = new FitAddon.FitAddon();
  term.loadAddon(fit);
  term.open(document.getElementById("terminal"));
  fit.fit();
  term.focus();

  let ws = null;

  function send(msg) {
    if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(msg));
  }

  function connect() {
    const proto = location.protocol === "https:" ? "wss" : "ws";
    ws = new WebSocket(`${proto}://${location.host}${api}/terminal`);
    ws.binaryType = "arraybuffer";
    ws.onopen = () => send({ type: "resize", cols: term.cols, rows: term.rows });
    ws.onmessage = (e) => term.write(new Uint8Array(e.data));
    ws.onclose = () => term.write("\r\n\x1b[33m[сессия терминала завершена — нажми «Начать заново» или обнови страницу]\x1b[0m\r\n");
  }

  term.onData((data) => send({ type: "input", data }));
  term.onResize(({ cols, rows }) => send({ type: "resize", cols, rows }));
  window.addEventListener("resize", () => fit.fit());

  function esc(s) {
    return String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c]);
  }

  async function post(action) {
    const res = await fetch(`${api}/${action}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || res.statusText);
    return data;
  }

  async function check() {
    const btn = document.getElementById("check");
    btn.disabled = true;
    try {
      const data = await post("check");
      output.innerHTML = data.output ? `<pre>${esc(data.output)}</pre>` : '<span class="muted">(нет вывода)</span>';
      if (data.ok) {
        const gained = data.firstTime ? ` +${data.xp} XP` : "";
        verdict.innerHTML = `<span>✅ ${esc(data.message)}${gained}</span><a class="btn primary" href="${data.nextUrl}">${data.nextUrl === "/" ? "К трекам" : "Следующий урок →"}</a>`;
        verdict.className = "verdict ok";
        document.getElementById("total-xp").textContent = data.totalXp;
      } else {
        verdict.innerHTML = `<span>❌ ${esc(data.message)}</span>`;
        verdict.className = "verdict fail";
      }
    } catch (e) {
      output.innerHTML = `<pre class="err">${esc(e.message)}</pre>`;
    } finally {
      btn.disabled = false;
      term.focus();
    }
  }

  async function reset() {
    if (!confirm("Начать урок заново? Все файлы в рабочей папке урока будут пересозданы.")) return;
    if (ws) { ws.onclose = null; ws.close(); }
    try {
      await post("reset");
    } catch (e) {
      output.innerHTML = `<pre class="err">${esc(e.message)}</pre>`;
      return;
    }
    term.reset();
    verdict.className = "verdict hidden";
    output.innerHTML = '<span class="muted">Здесь появится результат.</span>';
    connect();
    term.focus();
  }

  document.getElementById("check").addEventListener("click", check);
  document.getElementById("reset").addEventListener("click", reset);
  connect();
})();
