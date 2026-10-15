(function () {
  const root = document.querySelector(".lesson");
  const api = root.dataset.api;
  const kind = root.dataset.kind;
  const output = document.getElementById("output");
  const verdict = document.getElementById("verdict");

  const editor = CodeMirror.fromTextArea(document.getElementById("code"), {
    mode: kind === "sql" ? "text/x-sqlite" : "text/x-go",
    theme: "dracula",
    lineNumbers: true,
    indentUnit: kind === "go" ? 4 : 2,
    tabSize: 4,
    indentWithTabs: kind === "go",
    matchBrackets: true,
    autoCloseBrackets: true,
    extraKeys: {
      "Ctrl-Enter": () => run(),
      "Cmd-Enter": () => run(),
      "Shift-Ctrl-Enter": () => check(),
      "Shift-Cmd-Enter": () => check(),
    },
  });
  editor.setSize("100%", "100%");

  async function post(action, body) {
    const res = await fetch(`${api}/${action}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body || {}),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || res.statusText);
    return data;
  }

  function esc(s) {
    return String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c]);
  }

  function renderTable(t) {
    if (!t.columns || t.columns.length === 0) return '<span class="muted">Query returned no columns.</span>';
    const head = t.columns.map((c) => `<th>${esc(c)}</th>`).join("");
    const rows = t.rows.map((r) => `<tr>${r.map((v) => `<td>${esc(v)}</td>`).join("")}</tr>`).join("");
    return `<div class="muted small">${t.rows.length} row(s)</div><div class="scroll"><table><thead><tr>${head}</tr></thead><tbody>${rows}</tbody></table></div>`;
  }

  function showResult(data) {
    if (data.error) {
      output.innerHTML = `<pre class="err">${esc(data.error)}</pre>`;
    } else if (data.table) {
      output.innerHTML = renderTable(data.table);
    } else if (data.output !== undefined) {
      output.innerHTML = data.output
        ? `<pre>${esc(data.output)}</pre>`
        : '<span class="muted">(no output)</span>';
    }
  }

  function busy(on) {
    document.querySelectorAll(".actions button").forEach((b) => (b.disabled = on));
    if (on) output.innerHTML = '<span class="muted">Running…</span>';
  }

  async function run() {
    verdict.className = "verdict hidden";
    busy(true);
    try {
      showResult(await post("run", { code: editor.getValue() }));
    } catch (e) {
      output.innerHTML = `<pre class="err">${esc(e.message)}</pre>`;
    } finally {
      busy(false);
    }
  }

  async function check() {
    busy(true);
    try {
      const data = await post("check", { code: editor.getValue() });
      showResult(data);
      if (data.ok) {
        const gained = data.firstTime ? ` +${data.xp} XP` : "";
        verdict.innerHTML = `<span>✅ ${esc(data.message)}${gained}</span><a class="btn primary" href="${data.nextUrl}">${data.nextUrl === "/" ? "Back to tracks" : "Next lesson →"}</a>`;
        verdict.className = "verdict ok";
        document.getElementById("total-xp").textContent = data.totalXp;
      } else {
        verdict.innerHTML = `<span>❌ ${esc(data.message).replace(/\n/g, "<br>")}</span>`;
        verdict.className = "verdict fail";
      }
    } catch (e) {
      output.innerHTML = `<pre class="err">${esc(e.message)}</pre>`;
    } finally {
      busy(false);
    }
  }

  async function reset() {
    if (!confirm("Reset your code to the starter code?")) return;
    const data = await post("reset");
    editor.setValue(data.code);
    verdict.className = "verdict hidden";
    output.innerHTML = '<span class="muted">Output will appear here.</span>';
  }

  document.getElementById("run").addEventListener("click", run);
  document.getElementById("check").addEventListener("click", check);
  document.getElementById("reset").addEventListener("click", reset);
})();
