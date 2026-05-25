package graph

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type ViewConfig struct {
	Base   string
	Data   string
	Output string
	Open   bool
}

type ViewResult struct {
	Output string
}

func WriteViewer(cfg ViewConfig) (ViewResult, error) {
	if cfg.Base == "" {
		return ViewResult{}, fmt.Errorf("zmx graph view: --base is required")
	}
	if cfg.Data == "" {
		cfg.Data = filepath.Join(cfg.Base, "graph", "actions.json")
	}
	if cfg.Output == "" {
		cfg.Output = filepath.Join(cfg.Base, "graph", "index.html")
	}
	data, err := os.ReadFile(cfg.Data)
	if err != nil {
		return ViewResult{}, fmt.Errorf("zmx graph view: read graph data: %w", err)
	}
	if err := writeViewerHTML(cfg.Output, data); err != nil {
		return ViewResult{}, fmt.Errorf("zmx graph view: write viewer: %w", err)
	}
	if cfg.Open {
		if err := exec.Command("xdg-open", cfg.Output).Start(); err != nil {
			return ViewResult{}, fmt.Errorf("zmx graph view: open viewer: %w", err)
		}
	}
	return ViewResult{Output: cfg.Output}, nil
}

func writeViewerHTML(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	html := viewerHTML(base64.StdEncoding.EncodeToString(data))
	tempFile, err := os.CreateTemp(dir, ".tmp-*.html")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	if _, err := tempFile.WriteString(html); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func viewerHTML(dataBase64 string) string {
	return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>zmx action graph</title>
<style>
:root {
  color-scheme: light;
  --bg: #f6f7f8;
  --panel: #ffffff;
  --line: #d7dce1;
  --ink: #171a1f;
  --muted: #687380;
  --teal: #0f766e;
  --rose: #be123c;
  --amber: #b7791f;
  --violet: #6d28d9;
}
* { box-sizing: border-box; }
html, body { height: 100%; margin: 0; }
body {
  font: 14px/1.4 ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  color: var(--ink);
  background: var(--bg);
}
button, input, select {
  font: inherit;
}
#app {
  display: grid;
  grid-template-columns: minmax(240px, 310px) minmax(0, 1fr) minmax(260px, 360px);
  height: 100vh;
  min-height: 560px;
}
#app.no-selection {
  grid-template-columns: minmax(240px, 310px) minmax(0, 1fr);
}
#app.no-selection .right {
  display: none;
}
.side {
  background: var(--panel);
  border-color: var(--line);
  min-width: 0;
  overflow: auto;
}
.left { border-right: 1px solid var(--line); }
.right { border-left: 1px solid var(--line); }
.section { padding: 14px; border-bottom: 1px solid var(--line); }
.brand {
  font-weight: 700;
  letter-spacing: 0;
  font-size: 15px;
}
.stats {
  margin-top: 8px;
  color: var(--muted);
  font-size: 12px;
}
.search {
  width: 100%;
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 9px 10px;
  outline: none;
  background: #fff;
}
.search:focus {
  border-color: var(--teal);
  box-shadow: 0 0 0 3px rgba(15, 118, 110, 0.14);
}
.controls {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.controls select, .controls button {
  border: 1px solid var(--line);
  border-radius: 6px;
  background: #fff;
  padding: 8px;
  min-width: 0;
}
.segmented {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  border: 1px solid var(--line);
  border-radius: 6px;
  overflow: hidden;
}
.segmented button {
  border: 0;
  border-right: 1px solid var(--line);
  border-radius: 0;
  padding: 8px 4px;
}
.segmented button:last-child { border-right: 0; }
.segmented button.active {
  background: #e6f4f1;
  color: var(--teal);
  font-weight: 650;
}
.matches {
  display: flex;
  flex-direction: column;
}
.match {
  appearance: none;
  border: 0;
  border-bottom: 1px solid var(--line);
  background: transparent;
  text-align: left;
  padding: 10px 14px;
  cursor: pointer;
}
.match:hover, .match.active {
  background: #eef7f5;
}
.match-name {
  font-weight: 650;
  overflow-wrap: anywhere;
}
.match-path {
  color: var(--muted);
  font-size: 12px;
  overflow-wrap: anywhere;
  margin-top: 2px;
}
.stage {
  position: relative;
  min-width: 0;
  overflow: hidden;
}
canvas {
  display: block;
  width: 100%;
  height: 100%;
  background:
    linear-gradient(#edf0f2 1px, transparent 1px),
    linear-gradient(90deg, #edf0f2 1px, transparent 1px),
    #fbfcfd;
  background-size: 28px 28px;
}
.badge {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 2px 8px;
  background: #eef0f2;
  color: var(--muted);
  font-size: 12px;
  margin-right: 6px;
  margin-top: 6px;
}
.title {
  font-weight: 750;
  font-size: 17px;
  overflow-wrap: anywhere;
}
.subtle {
  color: var(--muted);
  font-size: 12px;
  overflow-wrap: anywhere;
}
.kv {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 8px;
  margin-top: 8px;
}
.kv div:nth-child(odd) {
  color: var(--muted);
}
.list {
  display: flex;
  flex-direction: column;
}
.list button {
  appearance: none;
  border: 0;
  border-bottom: 1px solid var(--line);
  background: transparent;
  text-align: left;
  padding: 9px 14px;
  cursor: pointer;
  overflow-wrap: anywhere;
}
.list button:hover {
  background: #eef7f5;
}
.empty {
  color: var(--muted);
  padding: 10px 14px;
}
@media (max-width: 980px) {
  #app {
    grid-template-columns: 1fr;
    grid-template-rows: auto minmax(420px, 1fr) auto;
    min-height: 100vh;
  }
  .left, .right { border: 0; }
  .left { border-bottom: 1px solid var(--line); }
  .right { border-top: 1px solid var(--line); max-height: 42vh; }
  .matches { max-height: 180px; overflow: auto; }
}
</style>
</head>
<body>
<div id="app" class="no-selection">
  <aside class="side left">
    <div class="section">
      <div class="brand">zmx graph</div>
      <div class="stats" id="stats"></div>
    </div>
    <div class="section">
      <input class="search" id="search" type="search" autocomplete="off" placeholder="Search action">
    </div>
    <div class="section controls">
      <select id="radius">
        <option value="1">1 hop</option>
        <option value="2">2 hops</option>
        <option value="all" selected>all</option>
      </select>
      <button id="reheat" type="button">Reheat</button>
      <button id="reset" type="button">Reset</button>
      <div class="segmented">
        <button data-direction="both" class="active" type="button">Both</button>
        <button data-direction="out" type="button">Callees</button>
        <button data-direction="in" type="button">Callers</button>
      </div>
    </div>
    <div class="matches" id="matches"></div>
  </aside>
  <main class="stage">
    <canvas id="graph"></canvas>
  </main>
  <aside class="side right" id="details"></aside>
</div>
<script>
const graphBytes = Uint8Array.from(atob("` + dataBase64 + `"), (ch) => ch.charCodeAt(0));
const GRAPH = JSON.parse(new TextDecoder().decode(graphBytes));
const palette = ["#0f766e", "#be123c", "#b7791f", "#2563eb", "#7c3aed", "#15803d", "#c2410c", "#0891b2", "#a21caf", "#4d7c0f"];
const nodes = GRAPH.nodes.map((node) => ({...node, x: 0, y: 0, r: 5, degree: 0}));
const byId = new Map(nodes.map((node) => [node.id, node]));
const outEdges = new Map(nodes.map((node) => [node.id, []]));
const inEdges = new Map(nodes.map((node) => [node.id, []]));
const neighbors = new Map(nodes.map((node) => [node.id, []]));
for (const edge of GRAPH.edges) {
  if (!byId.has(edge.from) || !byId.has(edge.to)) continue;
  outEdges.get(edge.from).push(edge);
  inEdges.get(edge.to).push(edge);
  neighbors.get(edge.from).push(edge.to);
  neighbors.get(edge.to).push(edge.from);
  byId.get(edge.from).degree++;
  byId.get(edge.to).degree++;
}
const sourceColors = new Map();
for (const node of nodes) {
  if (!sourceColors.has(node.source_group)) {
    sourceColors.set(node.source_group, palette[sourceColors.size % palette.length]);
  }
}
nodes.sort((a, b) => b.degree - a.degree || a.name.localeCompare(b.name));
assignComponents();

const state = {
  selected: null,
  query: "",
  radius: "all",
  direction: "both",
  scale: 1,
  tx: 0,
  ty: 0,
  dragging: false,
  dragStart: null,
  nodeDrag: null,
  suppressClick: false,
};
let simulation = null;
let animationFrame = 0;

const canvas = document.getElementById("graph");
const ctx = canvas.getContext("2d");
const search = document.getElementById("search");
const matches = document.getElementById("matches");
const details = document.getElementById("details");
const stats = document.getElementById("stats");
const radius = document.getElementById("radius");
const reheat = document.getElementById("reheat");
const reset = document.getElementById("reset");
const app = document.getElementById("app");

stats.textContent = GRAPH.stats.nodes + " actions, " + GRAPH.stats.edges + " edges, " + GRAPH.stats.ambiguous_calls + " ambiguous, " + GRAPH.stats.dynamic_calls + " dynamic";

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, (ch) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;",
  }[ch]));
}

function hash(value) {
  let h = 2166136261;
  for (let i = 0; i < value.length; i++) {
    h ^= value.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

function assignComponents() {
  let componentID = 0;
  const components = [];
  for (const node of nodes) {
    if (node.component !== undefined) continue;
    const queue = [node.id];
    const ids = [];
    node.component = componentID;
    while (queue.length) {
      const id = queue.shift();
      ids.push(id);
      for (const nextID of neighbors.get(id)) {
        const next = byId.get(nextID);
        if (!next || next.component !== undefined) continue;
        next.component = componentID;
        queue.push(nextID);
      }
    }
    for (const id of ids) {
      byId.get(id).component_size = ids.length;
    }
    components.push({id: componentID, size: ids.length});
    componentID++;
  }
  components.sort((a, b) => b.size - a.size || a.id - b.id);
  components.forEach((component, index) => component.rank = index);
  const rankByID = new Map(components.map((component) => [component.id, component.rank]));
  for (const node of nodes) {
    node.component_rank = rankByID.get(node.component) || 0;
  }
}

function resize() {
  const rect = canvas.getBoundingClientRect();
  const ratio = window.devicePixelRatio || 1;
  canvas.width = Math.max(1, Math.floor(rect.width * ratio));
  canvas.height = Math.max(1, Math.floor(rect.height * ratio));
  ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
  draw();
}

function visibleGraph() {
  const visible = new Map();
  const edgeSet = new Map();
  const maxGlobalNodes = 180;
  if (!state.selected || state.radius === "all") {
    const selectedNode = state.selected ? byId.get(state.selected) : null;
    const baseNodes = state.radius === "all" ? nodes : nodes.slice(0, maxGlobalNodes);
    for (const node of baseNodes) visible.set(node.id, 0);
    if (selectedNode) visible.set(selectedNode.id, 0);
  } else {
    const maxDepth = Number(state.radius);
    const queue = [{id: state.selected, depth: 0}];
    visible.set(state.selected, 0);
    while (queue.length) {
      const item = queue.shift();
      if (item.depth >= maxDepth) continue;
      const nextEdges = [];
      if (state.direction !== "in") nextEdges.push(...outEdges.get(item.id));
      if (state.direction !== "out") nextEdges.push(...inEdges.get(item.id));
      for (const edge of nextEdges) {
        const nextID = edge.from === item.id ? edge.to : edge.from;
        if (!visible.has(nextID)) {
          visible.set(nextID, item.depth + 1);
          queue.push({id: nextID, depth: item.depth + 1});
        }
      }
    }
  }
  for (const edge of GRAPH.edges) {
    if (visible.has(edge.from) && visible.has(edge.to)) {
      edgeSet.set(edge.from + "->" + edge.to + ":" + edge.line + ":" + edge.kind, edge);
    }
  }
  return {
    nodes: [...visible.keys()].map((id) => byId.get(id)).filter(Boolean),
    depths: visible,
    edges: [...edgeSet.values()],
  };
}

function graphKey(subgraph) {
  return state.selected + "|" + state.radius + "|" + state.direction + "|" + subgraph.nodes.map((node) => node.id).sort().join(",");
}

function restartSimulation() {
  simulation = null;
  if (animationFrame) {
    cancelAnimationFrame(animationFrame);
    animationFrame = 0;
  }
}

function reheatSimulation(alpha) {
  if (!simulation) {
    draw();
    return;
  }
  simulation.alpha = Math.max(simulation.alpha, alpha);
  scheduleSimulation();
}

function seedLayout(subgraph) {
  const rect = canvas.getBoundingClientRect();
  if (state.radius === "all") {
    seedGlobalLayout(subgraph, rect);
    return;
  }
  const cx = rect.width / 2;
  const cy = rect.height / 2;
  const selected = state.selected ? byId.get(state.selected) : null;
  if (selected && subgraph.depths.has(selected.id) && state.radius !== "all") {
    selected.x = cx;
    selected.y = cy;
    selected.vx = 0;
    selected.vy = 0;
    selected.r = 9;
    const rings = new Map();
    for (const node of subgraph.nodes) {
      const depth = subgraph.depths.get(node.id) || 0;
      if (depth === 0) continue;
      if (!rings.has(depth)) rings.set(depth, []);
      rings.get(depth).push(node);
    }
    for (const [depth, ringNodes] of rings) {
      ringNodes.sort((a, b) => hash(a.id) - hash(b.id));
      const ringRadius = Math.min(rect.width, rect.height) * (0.22 + depth * 0.18);
      ringNodes.forEach((node, index) => {
        const angle = ((Math.PI * 2) / ringNodes.length) * index + (hash(node.id) % 360) * Math.PI / 180;
        node.x = cx + Math.cos(angle) * ringRadius;
        node.y = cy + Math.sin(angle) * ringRadius;
        node.vx = 0;
        node.vy = 0;
        node.r = Math.max(4, Math.min(10, 4 + Math.sqrt(node.degree)));
      });
    }
    return;
  }
  const grouped = new Map();
  for (const node of subgraph.nodes) {
    if (!grouped.has(node.source_group)) grouped.set(node.source_group, []);
    grouped.get(node.source_group).push(node);
  }
  const groups = [...grouped.keys()].sort();
  const groupRadius = Math.min(rect.width, rect.height) * 0.31;
  groups.forEach((group, groupIndex) => {
    const groupAngle = ((Math.PI * 2) / groups.length) * groupIndex;
    const gx = cx + Math.cos(groupAngle) * groupRadius;
    const gy = cy + Math.sin(groupAngle) * groupRadius;
    const groupNodes = grouped.get(group).sort((a, b) => b.degree - a.degree || a.name.localeCompare(b.name));
    const localRadius = Math.min(130, 18 + groupNodes.length * 2.4);
    groupNodes.forEach((node, index) => {
      const angle = ((Math.PI * 2) / Math.max(1, groupNodes.length)) * index;
      node.x = gx + Math.cos(angle) * localRadius;
      node.y = gy + Math.sin(angle) * localRadius;
      node.vx = 0;
      node.vy = 0;
      node.r = Math.max(4, Math.min(11, 4 + Math.sqrt(node.degree)));
    });
  });
}

function seedGlobalLayout(subgraph, rect) {
  const cx = rect.width / 2;
  const cy = rect.height / 2;
  const componentNodes = new Map();
  for (const node of subgraph.nodes) {
    if (!componentNodes.has(node.component)) componentNodes.set(node.component, []);
    componentNodes.get(node.component).push(node);
  }
  const components = [...componentNodes.entries()]
    .map(([id, items]) => ({id, items, size: items.length, rank: items[0]?.component_rank || 0}))
    .sort((a, b) => b.size - a.size || a.rank - b.rank);

  const isolateNodes = [];
  const componentCenters = new Map();
  let componentIndex = 0;
  const maxComponentRadius = Math.min(rect.width, rect.height) * 0.36;
  for (const component of components) {
    if (component.size === 1) {
      isolateNodes.push(component.items[0]);
      continue;
    }
    const angle = componentIndex === 0 ? 0 : componentIndex * 2.399963229728653;
    const radius = componentIndex === 0 ? 0 : Math.min(maxComponentRadius, 52 + Math.sqrt(componentIndex) * 46);
    const center = {
      x: cx + Math.cos(angle) * radius,
      y: cy + Math.sin(angle) * radius * 0.72,
    };
    componentCenters.set(component.id, center);
    const localRadius = Math.min(110, 20 + Math.sqrt(component.size) * 17);
    component.items
      .sort((a, b) => b.degree - a.degree || hash(a.id) - hash(b.id))
      .forEach((node, index) => {
        const localAngle = index * 2.399963229728653;
        const localDistance = Math.sqrt(index + 1) * (localRadius / Math.sqrt(component.size + 1));
        node.x = center.x + Math.cos(localAngle) * localDistance;
        node.y = center.y + Math.sin(localAngle) * localDistance;
        node.vx = 0;
        node.vy = 0;
        node.r = Math.max(3.2, Math.min(9, 3.2 + Math.sqrt(node.degree)));
      });
    componentIndex++;
  }

  seedIsolateGrid(isolateNodes, rect);
}

function seedIsolateGrid(isolateNodes, rect) {
  if (!isolateNodes.length) return;
  const margin = 54;
  const width = Math.max(1, rect.width - margin * 2);
  const height = Math.max(1, rect.height - margin * 2);
  const columns = Math.max(1, Math.ceil(Math.sqrt(isolateNodes.length * width / height)));
  const rows = Math.max(1, Math.ceil(isolateNodes.length / columns));
  const cellWidth = width / columns;
  const cellHeight = height / rows;
  const sorted = isolateNodes.slice().sort((a, b) => hash(a.id) - hash(b.id));
  sorted.forEach((node, index) => {
    const row = Math.floor(index / columns);
    const column = index % columns;
    const nodeHash = hash(node.id);
    const jitterX = (((nodeHash & 255) / 255) - 0.5) * cellWidth * 0.42;
    const jitterY = ((((nodeHash >> 8) & 255) / 255) - 0.5) * cellHeight * 0.42;
    node.x = margin + column * cellWidth + cellWidth / 2 + jitterX;
    node.y = margin + row * cellHeight + cellHeight / 2 + jitterY;
    node.vx = 0;
    node.vy = 0;
    node.r = node.parse_error ? 4.2 : 3.2;
  });
}

function sourceGroupCenters(groups, rect) {
  const cx = rect.width / 2;
  const cy = rect.height / 2;
  const radius = Math.min(rect.width, rect.height) * 0.28;
  const result = new Map();
  groups.sort().forEach((group, index) => {
    const angle = ((Math.PI * 2) / Math.max(1, groups.length)) * index;
    result.set(group, {
      x: cx + Math.cos(angle) * radius,
      y: cy + Math.sin(angle) * radius,
    });
  });
  return result;
}

function ensureSimulation(subgraph) {
  const key = graphKey(subgraph);
  const rect = canvas.getBoundingClientRect();
  if (simulation && simulation.key === key) {
    return simulation;
  }
  seedLayout(subgraph);
  const groups = [...new Set(subgraph.nodes.map((node) => node.source_group))];
  const componentCenters = componentCenterMap(subgraph, rect);
  simulation = {
    key,
    nodes: subgraph.nodes,
    edges: subgraph.edges,
    depths: subgraph.depths,
    groupCenters: sourceGroupCenters(groups, rect),
    componentCenters,
    alpha: 1,
  };
  scheduleSimulation();
  return simulation;
}

function componentCenterMap(subgraph, rect) {
  const centers = new Map();
  const sums = new Map();
  for (const node of subgraph.nodes) {
    if (!sums.has(node.component)) {
      sums.set(node.component, {x: 0, y: 0, count: 0});
    }
    const item = sums.get(node.component);
    item.x += node.x;
    item.y += node.y;
    item.count++;
  }
  for (const [component, item] of sums) {
    centers.set(component, {
      x: item.x / item.count,
      y: item.y / item.count,
    });
  }
  if (!centers.size) {
    centers.set(0, {x: rect.width / 2, y: rect.height / 2});
  }
  return centers;
}

function applyPairRepulsion(a, b, alpha, strength) {
  if (state.radius === "all") {
    const aIsolate = a.component_size === 1;
    const bIsolate = b.component_size === 1;
    if (aIsolate && bIsolate) return;
    if (aIsolate || bIsolate) strength *= 0.08;
  }
  let dx = b.x - a.x;
  let dy = b.y - a.y;
  let distance2 = dx * dx + dy * dy;
  if (distance2 < 0.01) {
    const angle = ((hash(a.id + b.id) % 360) * Math.PI) / 180;
    dx = Math.cos(angle) * 0.1;
    dy = Math.sin(angle) * 0.1;
    distance2 = dx * dx + dy * dy;
  }
  const minDistance = a.r + b.r + 18;
  const clamped = Math.max(distance2, minDistance * minDistance * 0.35);
  const force = Math.min(2.8, (strength * alpha) / clamped);
  const fx = dx * force;
  const fy = dy * force;
  a.vx -= fx;
  a.vy -= fy;
  b.vx += fx;
  b.vy += fy;
}

function applyRepulsion(sim) {
  const count = sim.nodes.length;
  const global = state.radius === "all";
  const strength = global ? 180 : count > 400 ? 760 : 1450;
  if (count <= 420) {
    for (let i = 0; i < count; i++) {
      for (let j = i + 1; j < count; j++) {
        applyPairRepulsion(sim.nodes[i], sim.nodes[j], sim.alpha, strength);
      }
    }
    return;
  }
  for (let i = 0; i < count; i++) {
    const a = sim.nodes[i];
    for (let step = 1; step <= 32; step++) {
      const j = (i + step * 37) % count;
      if (j === i) continue;
      applyPairRepulsion(a, sim.nodes[j], sim.alpha, strength);
    }
  }
}

function stepSimulation(sim) {
  const rect = canvas.getBoundingClientRect();
  const cx = rect.width / 2;
  const cy = rect.height / 2;
  const global = state.radius === "all";
  const selectedID = state.radius !== "all" ? state.selected : null;
  applyRepulsion(sim);

  for (const edge of sim.edges) {
    const from = byId.get(edge.from);
    const to = byId.get(edge.to);
    if (!from || !to) continue;
    let dx = to.x - from.x;
    let dy = to.y - from.y;
    let distance = Math.hypot(dx, dy);
    if (distance < 0.01) distance = 0.01;
    const target = global ? 58 : edge.kind === "zmx-call" ? 92 : 116;
    const force = (distance - target) * (global ? 0.026 : 0.018) * sim.alpha;
    const fx = (dx / distance) * force;
    const fy = (dy / distance) * force;
    from.vx += fx;
    from.vy += fy;
    to.vx -= fx;
    to.vy -= fy;
  }

  for (const node of sim.nodes) {
    const groupCenter = sim.groupCenters.get(node.source_group) || {x: cx, y: cy};
    const componentCenter = sim.componentCenters.get(node.component) || groupCenter;
    if (node.fixed) {
      node.x = node.fx;
      node.y = node.fy;
      node.vx = 0;
      node.vy = 0;
      continue;
    }
    const isolate = global && node.component_size === 1;
    const centerStrength = selectedID === node.id ? 0.07 : isolate ? 0 : global ? 0.0012 : 0.004;
    const groupStrength = selectedID === node.id ? 0 : isolate ? 0 : global ? 0.0008 : 0.0022;
    const componentStrength = global ? node.component_size > 1 ? 0.006 : 0.045 : 0;
    node.vx += (cx - node.x) * centerStrength * sim.alpha;
    node.vy += (cy - node.y) * centerStrength * sim.alpha;
    node.vx += (groupCenter.x - node.x) * groupStrength * sim.alpha;
    node.vy += (groupCenter.y - node.y) * groupStrength * sim.alpha;
    node.vx += (componentCenter.x - node.x) * componentStrength * sim.alpha;
    node.vy += (componentCenter.y - node.y) * componentStrength * sim.alpha;

    const speed = Math.hypot(node.vx, node.vy);
    if (speed > 24) {
      node.vx = (node.vx / speed) * 24;
      node.vy = (node.vy / speed) * 24;
    }
    node.x += node.vx;
    node.y += node.vy;
    node.vx *= 0.78;
    node.vy *= 0.78;

    const padding = global ? 58 : 28;
    const boundaryStrength = global ? 0.035 : 0.018;
    if (node.x < padding) node.vx += (padding - node.x) * boundaryStrength * sim.alpha;
    if (node.x > rect.width - padding) node.vx -= (node.x - rect.width + padding) * boundaryStrength * sim.alpha;
    if (node.y < padding) node.vy += (padding - node.y) * boundaryStrength * sim.alpha;
    if (node.y > rect.height - padding) node.vy -= (node.y - rect.height + padding) * boundaryStrength * sim.alpha;
  }
  sim.alpha *= 0.94;
}

function scheduleSimulation() {
  if (animationFrame) return;
  animationFrame = requestAnimationFrame(runSimulation);
}

function runSimulation() {
  animationFrame = 0;
  if (!simulation || simulation.alpha < 0.018) {
    return;
  }
  const ticks = simulation.nodes.length > 700 ? 1 : 2;
  for (let i = 0; i < ticks; i++) {
    stepSimulation(simulation);
  }
  drawFrame(simulation);
  scheduleSimulation();
}

function screenToWorld(x, y) {
  return {
    x: (x - state.tx) / state.scale,
    y: (y - state.ty) / state.scale,
  };
}

function drawArrow(edge, highlighted) {
  const from = byId.get(edge.from);
  const to = byId.get(edge.to);
  if (!from || !to) return;
  const dx = to.x - from.x;
  const dy = to.y - from.y;
  const length = Math.hypot(dx, dy) || 1;
  const ux = dx / length;
  const uy = dy / length;
  const startX = from.x + ux * (from.r + 2);
  const startY = from.y + uy * (from.r + 2);
  const endX = to.x - ux * (to.r + 5);
  const endY = to.y - uy * (to.r + 5);
  ctx.strokeStyle = highlighted ? "#0f766e" : "rgba(104, 115, 128, 0.42)";
  ctx.lineWidth = highlighted ? 1.8 : 1;
  ctx.beginPath();
  ctx.moveTo(startX, startY);
  ctx.lineTo(endX, endY);
  ctx.stroke();
  const size = highlighted ? 7 : 5;
  ctx.fillStyle = highlighted ? "#0f766e" : "rgba(104, 115, 128, 0.58)";
  ctx.beginPath();
  ctx.moveTo(endX, endY);
  ctx.lineTo(endX - ux * size - uy * size * 0.55, endY - uy * size + ux * size * 0.55);
  ctx.lineTo(endX - ux * size + uy * size * 0.55, endY - uy * size - ux * size * 0.55);
  ctx.closePath();
  ctx.fill();
}

function draw() {
  const subgraph = visibleGraph();
  const sim = ensureSimulation(subgraph);
  drawFrame(sim);
}

function drawFrame(sim) {
  const rect = canvas.getBoundingClientRect();
  ctx.clearRect(0, 0, rect.width, rect.height);
  ctx.save();
  ctx.translate(state.tx, state.ty);
  ctx.scale(state.scale, state.scale);
  for (const edge of sim.edges) {
    const highlighted = edge.from === state.selected || edge.to === state.selected;
    drawArrow(edge, highlighted);
  }
  for (const node of sim.nodes) {
    const selected = node.id === state.selected;
    const color = sourceColors.get(node.source_group) || "#687380";
    ctx.fillStyle = selected ? "#ffffff" : color;
    ctx.strokeStyle = selected ? "#0f766e" : node.duplicate_name ? "#be123c" : "#ffffff";
    ctx.lineWidth = selected ? 3 : node.duplicate_name ? 2 : 1.5;
    ctx.beginPath();
    ctx.arc(node.x, node.y, selected ? node.r + 2 : node.r, 0, Math.PI * 2);
    ctx.fill();
    ctx.stroke();
  }
  const globalView = state.radius === "all";
  ctx.font = globalView ? "10px ui-sans-serif, system-ui" : "12px ui-sans-serif, system-ui";
  ctx.textAlign = "center";
  ctx.textBaseline = "top";
  for (const node of sim.nodes) {
    ctx.fillStyle = node.id === state.selected ? "#0f766e" : "#171a1f";
    const maxLength = globalView ? 24 : 32;
    const label = node.name.length > maxLength ? node.name.slice(0, maxLength - 1) + "…" : node.name;
    ctx.fillText(label, node.x, node.y + node.r + (globalView ? 5 : 7));
  }
  ctx.restore();
}

function renderMatches() {
  const query = state.query.trim().toLowerCase();
  const found = (query
    ? nodes.filter((node) => (node.name + " " + node.real_source).toLowerCase().includes(query))
    : nodes.slice(0, 60)
  ).slice(0, 80);
  matches.innerHTML = found.map((node) =>
    '<button class="match ' + (node.id === state.selected ? "active" : "") + '" data-id="' + escapeHTML(node.id) + '" type="button">' +
      '<div class="match-name">' + escapeHTML(node.name) + '</div>' +
      '<div class="match-path">' + escapeHTML(node.source_group) + ':' + node.line + '</div>' +
    '</button>'
  ).join("");
}

function linkedButtons(title, edges, selector) {
  if (!edges.length) {
    return '<div class="section"><div class="brand">' + escapeHTML(title) + '</div><div class="empty">None</div></div>';
  }
  return '<div class="section"><div class="brand">' + escapeHTML(title) + '</div></div><div class="list">' +
    edges.slice(0, 80).map((edge) => {
      const node = byId.get(selector(edge));
      return '<button data-id="' + escapeHTML(node.id) + '" type="button">' + escapeHTML(node.name) + ' <span class="subtle">' + escapeHTML(node.source_group) + ':' + node.line + '</span></button>';
    }).join("") +
    '</div>';
}

function renderDetails() {
  const node = state.selected ? byId.get(state.selected) : null;
  const wasNoSelection = app.classList.contains("no-selection");
  app.classList.toggle("no-selection", !node);
  if (wasNoSelection !== !node) {
    requestAnimationFrame(resize);
  }
  if (!node) {
    details.innerHTML = '<div class="section"><div class="title">No action selected</div></div>';
    return;
  }
  const callers = inEdges.get(node.id).slice().sort((a, b) => byId.get(a.from).name.localeCompare(byId.get(b.from).name));
  const callees = outEdges.get(node.id).slice().sort((a, b) => byId.get(a.to).name.localeCompare(byId.get(b.to).name));
  const ambiguous = (GRAPH.ambiguous_calls || []).filter((call) => call.from === node.id);
  const dynamic = (GRAPH.dynamic_calls || []).filter((call) => call.from === node.id);
  const parseErrors = (GRAPH.parse_errors || []).filter((item) => item.node_id === node.id);
  details.innerHTML =
    '<div class="section">' +
      '<div class="title">' + escapeHTML(node.name) + '</div>' +
      '<span class="badge">' + node.degree + ' degree</span>' +
      (node.duplicate_name ? '<span class="badge">duplicate</span>' : '') +
      (node.parse_error ? '<span class="badge">parse error</span>' : '') +
      '<div class="kv">' +
        '<div>source</div><div>' + escapeHTML(node.real_source) + '</div>' +
        '<div>line</div><div>' + node.line + '</div>' +
        '<div>group</div><div>' + escapeHTML(node.source_group) + '</div>' +
      '</div>' +
    '</div>' +
    linkedButtons("Callers", callers, (edge) => edge.from) +
    linkedButtons("Callees", callees, (edge) => edge.to) +
    '<div class="section">' +
      '<div class="brand">Unresolved</div>' +
      '<div class="subtle">' + ambiguous.length + ' ambiguous, ' + dynamic.length + ' dynamic</div>' +
      ambiguous.slice(0, 12).map((call) => '<div class="badge">' + escapeHTML(call.name) + ':' + call.line + '</div>').join("") +
      dynamic.slice(0, 12).map((call) => '<div class="badge">' + escapeHTML(call.reason) + ':' + call.line + '</div>').join("") +
      parseErrors.map((item) => '<div class="subtle">' + escapeHTML(item.error) + '</div>').join("") +
    '</div>';
}

function render() {
  renderMatches();
  renderDetails();
  draw();
}

function selectNode(id) {
  const changed = state.selected !== id;
  state.selected = id;
  if (changed && state.radius !== "all") {
    restartSimulation();
  }
  render();
}

matches.addEventListener("click", (event) => {
  const button = event.target.closest("button[data-id]");
  if (!button) return;
  selectNode(button.dataset.id);
});

details.addEventListener("click", (event) => {
  const button = event.target.closest("button[data-id]");
  if (!button) return;
  selectNode(button.dataset.id);
});

search.addEventListener("input", () => {
  state.query = search.value;
  renderMatches();
});

radius.addEventListener("change", () => {
  state.radius = radius.value;
  restartSimulation();
  draw();
});

reheat.addEventListener("click", () => {
  restartSimulation();
  draw();
});

reset.addEventListener("click", () => {
  state.selected = null;
  state.scale = 1;
  state.tx = 0;
  state.ty = 0;
  search.value = "";
  state.query = "";
  radius.value = "all";
  state.radius = "all";
  state.direction = "both";
  document.querySelectorAll(".segmented button").forEach((button) => button.classList.toggle("active", button.dataset.direction === "both"));
  restartSimulation();
  render();
});

document.querySelector(".segmented").addEventListener("click", (event) => {
  const button = event.target.closest("button[data-direction]");
  if (!button) return;
  state.direction = button.dataset.direction;
  document.querySelectorAll(".segmented button").forEach((item) => item.classList.toggle("active", item === button));
  restartSimulation();
  draw();
});

function pickNodeAt(event) {
  const rect = canvas.getBoundingClientRect();
  const point = screenToWorld(event.clientX - rect.left, event.clientY - rect.top);
  const subgraph = visibleGraph();
  let best = null;
  let bestDistance = Infinity;
  for (const node of subgraph.nodes) {
    const distance = Math.hypot(point.x - node.x, point.y - node.y);
    if (distance < Math.max(10, node.r + 6) && distance < bestDistance) {
      best = node;
      bestDistance = distance;
    }
  }
  return {node: best, point};
}

canvas.addEventListener("click", (event) => {
  if (state.suppressClick) {
    state.suppressClick = false;
    return;
  }
  if (state.dragStart && Math.hypot(event.clientX - state.dragStart.x, event.clientY - state.dragStart.y) > 4) return;
  const picked = pickNodeAt(event);
  if (picked.node) {
    selectNode(picked.node.id);
  }
});

canvas.addEventListener("mousedown", (event) => {
  const picked = pickNodeAt(event);
  if (picked.node) {
    state.nodeDrag = {
      id: picked.node.id,
      moved: false,
      startX: event.clientX,
      startY: event.clientY,
    };
    picked.node.fixed = true;
    picked.node.fx = picked.point.x;
    picked.node.fy = picked.point.y;
    reheatSimulation(0.28);
    return;
  }
  state.dragging = true;
  state.dragStart = {x: event.clientX, y: event.clientY, tx: state.tx, ty: state.ty};
});

window.addEventListener("mousemove", (event) => {
  if (state.nodeDrag) {
    const node = byId.get(state.nodeDrag.id);
    if (!node) return;
    const rect = canvas.getBoundingClientRect();
    const point = screenToWorld(event.clientX - rect.left, event.clientY - rect.top);
    node.fixed = true;
    node.fx = point.x;
    node.fy = point.y;
    node.x = point.x;
    node.y = point.y;
    node.vx = 0;
    node.vy = 0;
    if (Math.hypot(event.clientX - state.nodeDrag.startX, event.clientY - state.nodeDrag.startY) > 3) {
      state.nodeDrag.moved = true;
    }
    reheatSimulation(0.36);
    draw();
    return;
  }
  if (!state.dragging) return;
  state.tx = state.dragStart.tx + event.clientX - state.dragStart.x;
  state.ty = state.dragStart.ty + event.clientY - state.dragStart.y;
  draw();
});

window.addEventListener("mouseup", () => {
  if (state.nodeDrag) {
    const moved = state.nodeDrag.moved;
    const nodeID = state.nodeDrag.id;
    const node = byId.get(state.nodeDrag.id);
    if (node) {
      node.fixed = false;
      node.fx = undefined;
      node.fy = undefined;
    }
    state.nodeDrag = null;
    if (moved) {
      state.suppressClick = true;
      selectNode(nodeID);
    }
    reheatSimulation(0.18);
  }
  state.dragging = false;
});

canvas.addEventListener("wheel", (event) => {
  event.preventDefault();
  const rect = canvas.getBoundingClientRect();
  const before = screenToWorld(event.clientX - rect.left, event.clientY - rect.top);
  const factor = event.deltaY < 0 ? 1.12 : 0.89;
  state.scale = Math.max(0.35, Math.min(3.2, state.scale * factor));
  const after = screenToWorld(event.clientX - rect.left, event.clientY - rect.top);
  state.tx += (after.x - before.x) * state.scale;
  state.ty += (after.y - before.y) * state.scale;
  draw();
}, {passive: false});

window.addEventListener("resize", resize);
resize();
render();
</script>
</body>
</html>
`
}
