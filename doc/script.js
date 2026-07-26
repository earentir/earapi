const GROUPS = [
  {
    id: "core",
    title: "Core",
    description: "Service discovery and version.",
    endpoints: [
      {
        id: "root",
        method: "GET",
        path: "/",
        summary: "List registered routes",
        params: [],
      },
      {
        id: "version",
        method: "GET",
        path: "/version",
        summary: "Current API / app version",
        params: [],
      },
    ],
  },
  {
    id: "steam",
    title: "Steam",
    description: "Steam user library, app lookup, and store details.",
    endpoints: [
      {
        id: "steam-top",
        method: "GET",
        path: "/steam/v1/top",
        summary: "Top apps for a user by playtime or last played",
        params: [
          { name: "userid", required: true, value: "76561198011985757", hint: "SteamID64" },
          { name: "count", value: "5", hint: "How many to return" },
          { name: "sortby", value: "playtime", hint: "playtime | lastplayed" },
        ],
      },
      {
        id: "steam-getuserid",
        method: "GET",
        path: "/steam/v1/getuserid",
        summary: "Resolve vanity name to SteamID64",
        params: [
          { name: "username", required: true, value: "earentir", hint: "Vanity URL name" },
        ],
      },
      {
        id: "steam-appsused",
        method: "GET",
        path: "/steam/v1/appsused",
        summary: "All owned/played apps for a user",
        params: [
          { name: "userid", required: true, value: "76561198011985757" },
        ],
      },
      {
        id: "steam-appdata",
        method: "GET",
        path: "/steam/v1/appdata",
        summary: "Store details for an app ID",
        params: [
          { name: "appid", required: true, value: "1086940", hint: "Numeric app ID" },
        ],
      },
      {
        id: "steam-search",
        method: "GET",
        path: "/steam/v1/search",
        summary: "Find an app ID by name",
        params: [
          { name: "app", required: true, value: "Baldur's Gate 3" },
        ],
      },
    ],
  },
  {
    id: "joke",
    title: "Joke",
    description: "Random jokes from local datasets.",
    endpoints: [
      {
        id: "joke",
        method: "GET",
        path: "/joke",
        summary: "Geek joke or BOFH excuse",
        params: [
          { name: "type", value: "excuse", hint: "geek | excuse" },
        ],
      },
    ],
  },
  {
    id: "netflix",
    title: "Netflix",
    description: "Weekly Top 10 charts.",
    endpoints: [
      {
        id: "netflix-top",
        method: "GET",
        path: "/netflix/v1/top",
        summary: "Top titles for a country / type",
        params: [
          { name: "type", value: "series", hint: "films | series | popular (or movies/tv aliases)" },
          { name: "country", value: "", hint: "e.g. us — spaces as -" },
        ],
      },
    ],
  },
  {
    id: "tmdb",
    title: "TMDB",
    description: "Movie / TV search via TMDB.",
    endpoints: [
      {
        id: "tmdb-search",
        method: "GET",
        path: "/tmdb/v1/search",
        summary: "Search media by title",
        params: [
          { name: "q", required: true, value: "Blade Runner", hint: "Also accepts query=" },
        ],
      },
    ],
  },
  {
    id: "tilecalc",
    title: "Tilecalc",
    description: "Tile layouts, coverage, and optional pricing.",
    endpoints: [
      {
        id: "tilecalc-arrange",
        method: "GET",
        path: "/tilecalc/v1/arrange",
        summary: "Grid arrangements for a fixed tile count",
        params: [
          { name: "size", required: true, value: "15x40", hint: "WxH in cm" },
          { name: "count", required: true, value: "32" },
          { name: "price", value: "16", hint: "Optional price amount" },
          { name: "per", value: "6", hint: "Tiles covered by price (default 1)" },
          { name: "graph", value: "", hint: "true to include ASCII grid" },
        ],
      },
      {
        id: "tilecalc-coverage",
        method: "GET",
        path: "/tilecalc/v1/coverage",
        summary: "Tiles and cuts to fill a space",
        params: [
          { name: "size", required: true, value: "15x40", hint: "Tile WxH cm" },
          { name: "space", required: true, value: "300x130", hint: "Space WxH cm" },
          { name: "price", value: "", hint: "Optional" },
          { name: "per", value: "", hint: "Optional pack size" },
          { name: "singledimensionpattern", value: "", hint: "true = one orientation only" },
        ],
      },
    ],
  },
  {
    id: "dmt",
    title: "Discord Magic Time",
    description: "Build Discord &lt;t:unix:style&gt; timestamp tags.",
    endpoints: [
      {
        id: "dmt-formats",
        method: "GET",
        path: "/dmt/v1/formats",
        summary: "List Discord timestamp styles",
        params: [],
      },
      {
        id: "dmt-timestamp",
        method: "GET",
        path: "/dmt/v1/timestamp",
        summary: "Convert date/time to Discord tags",
        params: [
          { name: "year", value: "2026" },
          { name: "month", value: "7" },
          { name: "day", value: "25" },
          { name: "hour", value: "21" },
          { name: "minute", value: "45" },
          { name: "second", value: "0" },
          { name: "offset", value: "+03:00", hint: "Zone for component/datetime form" },
          { name: "format", value: "f", hint: "f F d D t T R or 0–6" },
          { name: "complete", value: "", hint: "true = round up to 5 minutes" },
        ],
      },
    ],
  },
  {
    id: "youtube",
    title: "YouTube",
    description: "Playlist helpers (mostly POST). Inline try covers the GET reads.",
    endpoints: [
      {
        id: "yt-items",
        method: "GET",
        path: "/youtube/v1/playlist/items",
        summary: "List videos in a playlist by name",
        params: [
          { name: "name", required: true, value: "My Playlist" },
          { name: "fuzzy", value: "false" },
          { name: "metadata", value: "false" },
        ],
      },
    ],
  },
];

function $(sel, root = document) {
  return root.querySelector(sel);
}

function el(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === "className") node.className = v;
    else if (k === "text") node.textContent = v;
    else if (k === "html") node.innerHTML = v;
    else if (k.startsWith("on") && typeof v === "function") node.addEventListener(k.slice(2).toLowerCase(), v);
    else node.setAttribute(k, v);
  }
  for (const child of [].concat(children)) {
    if (child == null || child === false) continue;
    node.append(typeof child === "string" ? document.createTextNode(child) : child);
  }
  return node;
}

function getBase() {
  const raw = ($("#baseUrl").value || "").trim().replace(/\/+$/, "");
  return raw || "https://api.earentir.dev";
}

function buildUrl(endpoint, form) {
  const params = new URLSearchParams();
  for (const p of endpoint.params) {
    const input = form.querySelector(`[name="${p.name}"]`);
    const value = (input?.value ?? "").trim();
    if (value !== "") params.set(p.name, value);
  }
  const qs = params.toString();
  return getBase() + endpoint.path + (qs ? `?${qs}` : "");
}

function render() {
  const toc = $("#toc");
  const root = $("#endpoints");
  toc.replaceChildren();
  root.replaceChildren();

  for (const group of GROUPS) {
    toc.append(el("div", { className: "toc-group", text: group.title }));
    for (const ep of group.endpoints) {
      toc.append(el("a", { href: `#${ep.id}`, text: ep.path }));
    }

    const section = el("section", { className: "group", id: group.id }, [
      el("h2", { className: "group-title", text: group.title }),
      el("p", { className: "group-desc", html: group.description }),
    ]);

    for (const ep of group.endpoints) {
      section.append(renderEndpoint(ep));
    }
    root.append(section);
  }
}

function renderEndpoint(ep) {
  const card = el("article", { className: "endpoint", id: ep.id });
  const head = el("div", { className: "endpoint-head" }, [
    el("span", { className: `method ${ep.method.toLowerCase()}`, text: ep.method }),
    el("span", { className: "path", text: ep.path }),
    el("span", { className: "summary", text: ep.summary }),
    el("span", { className: "chev", text: "▸" }),
  ]);

  const body = el("div", { className: "endpoint-body" });
  const form = el("form", { className: "params", onSubmit: (e) => e.preventDefault() });

  for (const p of ep.params) {
    form.append(
      el("label", { className: "param" }, [
        el("div", { className: "label-row" }, [
          el("span", { className: "name", text: p.name }),
          p.required ? el("span", { className: "req", text: "required" }) : null,
          p.hint ? el("span", { className: "hint", text: p.hint }) : null,
        ]),
        el("input", {
          name: p.name,
          value: p.value ?? "",
          spellcheck: "false",
          autocomplete: "off",
        }),
      ])
    );
  }

  const urlPreview = el("p", { className: "url-preview", text: "…" });
  const status = el("span", { className: "status-pill", text: "idle" });
  const result = el("pre", { className: "result empty", text: "Run a request to see the JSON response here." });

  const runBtn = el("button", { className: "btn", type: "button", text: "Run request" });
  const copyBtn = el("button", { className: "btn btn-ghost", type: "button", text: "Copy URL" });

  const refreshUrl = () => {
    urlPreview.textContent = buildUrl(ep, form);
  };

  form.addEventListener("input", refreshUrl);
  refreshUrl();

  runBtn.addEventListener("click", async () => {
    const url = buildUrl(ep, form);
    urlPreview.textContent = url;
    runBtn.disabled = true;
    status.textContent = "loading…";
    status.className = "status-pill";
    result.className = "result";
    result.textContent = "Fetching…";

    const started = performance.now();
    try {
      const res = await fetch(url);
      const text = await res.text();
      const ms = Math.round(performance.now() - started);
      let pretty = text;
      try {
        pretty = JSON.stringify(JSON.parse(text), null, 2);
      } catch {
        /* keep raw */
      }
      status.textContent = `${res.status} · ${ms}ms`;
      status.className = `status-pill ${res.ok ? "ok" : "bad"}`;
      result.className = res.ok ? "result" : "result error";
      result.textContent = pretty || "(empty body)";
    } catch (err) {
      status.textContent = "error";
      status.className = "status-pill bad";
      result.className = "result error";
      result.textContent = String(err.message || err);
    } finally {
      runBtn.disabled = false;
    }
  });

  copyBtn.addEventListener("click", async () => {
    const url = buildUrl(ep, form);
    try {
      await navigator.clipboard.writeText(url);
      copyBtn.textContent = "Copied";
      setTimeout(() => { copyBtn.textContent = "Copy URL"; }, 1000);
    } catch {
      copyBtn.textContent = "Copy failed";
      setTimeout(() => { copyBtn.textContent = "Copy URL"; }, 1000);
    }
  });

  head.addEventListener("click", () => {
    card.classList.toggle("open");
  });

  body.append(
    form,
    urlPreview,
    el("div", { className: "actions" }, [runBtn, copyBtn, status]),
    result
  );
  card.append(head, body);

  // Open first endpoint of each group by default feels noisy; open tilecalc/dmt & core version
  if (["version", "tilecalc-arrange", "dmt-timestamp"].includes(ep.id)) {
    card.classList.add("open");
  }

  return card;
}

document.addEventListener("DOMContentLoaded", () => {
  render();
  $("#baseUrl").addEventListener("input", () => {
    document.querySelectorAll(".endpoint form").forEach((form) => {
      form.dispatchEvent(new Event("input"));
    });
  });
});
