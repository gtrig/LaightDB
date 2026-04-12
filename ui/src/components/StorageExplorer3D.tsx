import { Suspense, useEffect, useMemo, useRef, useState, useCallback } from "react";
import { Canvas, useFrame, useThree } from "@react-three/fiber";
import { OrbitControls, Html } from "@react-three/drei";
import * as THREE from "three";
import { useNavigate } from "react-router-dom";
import { getGraphOverview, getStorageDiagnostics } from "../api";
import type { GraphOverview, OverviewNode, StorageDiagnostics } from "../types";

// ─── Colours ────────────────────────────────────────────────────────────────

const PALETTE = [
  "#6366f1", "#8b5cf6", "#ec4899", "#14b8a6", "#f59e0b",
  "#10b981", "#3b82f6", "#f97316", "#06b6d4", "#84cc16",
];

function collectionColor(col: string): string {
  if (!col) return PALETTE[0];
  let h = 0;
  for (let i = 0; i < col.length; i++) h = (h * 31 + col.charCodeAt(i)) >>> 0;
  return PALETTE[h % PALETTE.length];
}

/** Edge segment using classic `THREE.Line` (drei `<Line />` uses fat Line2 + screen resolution; keep edges simple). */
function GraphEdgeLine({
  from,
  to,
}: {
  from: [number, number, number];
  to: [number, number, number];
}) {
  const line = useMemo(() => {
    const geometry = new THREE.BufferGeometry().setFromPoints([
      new THREE.Vector3(...from),
      new THREE.Vector3(...to),
    ]);
    const material = new THREE.LineBasicMaterial({
      color: "#4f46e5",
      transparent: true,
      opacity: 0.45,
    });
    return new THREE.Line(geometry, material);
  }, [from, to]);

  useEffect(() => {
    return () => {
      line.geometry.dispose();
      (line.material as THREE.Material).dispose();
    };
  }, [line]);

  return <primitive object={line} />;
}

// ─── Force-directed layout ────────────────────────────────────────────────────
//
// d3-force-3d ships ESM; pull the functions we need directly.
// We run the simulation synchronously for N ticks before first render,
// so the initial layout is stable and there's no visible jitter.

interface ForceNode {
  id: string;
  x: number;
  y: number;
  z: number;
  vx: number;
  vy: number;
  vz: number;
}

interface ForceLink {
  source: ForceNode;
  target: ForceNode;
}

function buildForceLayout(
  nodes: OverviewNode[],
  edges: GraphOverview["edges"],
  ticks = 300,
): Map<string, [number, number, number]> {
  const nodeMap = new Map<string, ForceNode>();
  const fNodes: ForceNode[] = nodes.map((n) => {
    const fn: ForceNode = {
      id: n.id,
      x: (Math.random() - 0.5) * 100,
      y: (Math.random() - 0.5) * 100,
      z: (Math.random() - 0.5) * 100,
      vx: 0,
      vy: 0,
      vz: 0,
    };
    nodeMap.set(n.id, fn);
    return fn;
  });

  const fLinks: ForceLink[] = edges.flatMap((e) => {
    const s = nodeMap.get(e.from_id);
    const t = nodeMap.get(e.to_id);
    return s && t ? [{ source: s, target: t }] : [];
  });

  const alpha = { value: 1 };
  const alphaDecay = 1 - Math.pow(0.001, 1 / ticks);
  const k = Math.sqrt(fNodes.length > 0 ? (200 * 200) / fNodes.length : 1);

  for (let tick = 0; tick < ticks; tick++) {
    alpha.value *= 1 - alphaDecay;
    const a = alpha.value;

    // Repulsion (Barnes-Hut approximated as O(n²) for simplicity)
    for (let i = 0; i < fNodes.length; i++) {
      for (let j = i + 1; j < fNodes.length; j++) {
        const ni = fNodes[i];
        const nj = fNodes[j];
        const dx = nj.x - ni.x || 1e-6;
        const dy = nj.y - ni.y || 1e-6;
        const dz = nj.z - ni.z || 1e-6;
        const d2 = Math.max(dx * dx + dy * dy + dz * dz, 1e-2);
        const f = (k * k) / d2;
        ni.vx -= dx * f * a;
        ni.vy -= dy * f * a;
        ni.vz -= dz * f * a;
        nj.vx += dx * f * a;
        nj.vy += dy * f * a;
        nj.vz += dz * f * a;
      }
    }

    // Attraction along edges
    const linkStrength = 1 / Math.max(fLinks.length, 1);
    for (const lk of fLinks) {
      const dx = lk.target.x - lk.source.x;
      const dy = lk.target.y - lk.source.y;
      const dz = lk.target.z - lk.source.z;
      const d = Math.sqrt(dx * dx + dy * dy + dz * dz) || 1;
      const f = ((d - k) / d) * linkStrength * a;
      lk.source.vx += dx * f;
      lk.source.vy += dy * f;
      lk.source.vz += dz * f;
      lk.target.vx -= dx * f;
      lk.target.vy -= dy * f;
      lk.target.vz -= dz * f;
    }

    // Gravity toward origin
    for (const n of fNodes) {
      n.vx -= n.x * 0.01 * a;
      n.vy -= n.y * 0.01 * a;
      n.vz -= n.z * 0.01 * a;
    }

    // Integrate
    for (const n of fNodes) {
      n.x += n.vx *= 0.4;
      n.y += n.vy *= 0.4;
      n.z += n.vz *= 0.4;
    }
  }

  // Fit into a modest radius so nodes stay inside the default perspective frustum (camera ~z=220, fov 50).
  let maxR = 0;
  for (const n of fNodes) {
    const r = Math.hypot(n.x, n.y, n.z);
    if (r > maxR) maxR = r;
  }
  const targetRadius = 55;
  const scale = maxR > 1e-6 ? targetRadius / maxR : 1;

  const positions = new Map<string, [number, number, number]>();
  for (const n of fNodes) {
    const x = Number.isFinite(n.x) ? n.x * scale : 0;
    const y = Number.isFinite(n.y) ? n.y * scale : 0;
    const z = Number.isFinite(n.z) ? n.z * scale : 0;
    positions.set(n.id, [x, y, z]);
  }
  return positions;
}

// ─── Graph node sphere ────────────────────────────────────────────────────────

function GraphNode({
  node,
  pos,
  onHover,
  onClick,
}: {
  node: OverviewNode;
  pos: [number, number, number];
  onHover: (id: string | null) => void;
  onClick: (id: string) => void;
}) {
  const groupRef = useRef<THREE.Group>(null!);
  const color = collectionColor(node.collection);
  const coreColor = useMemo(() => new THREE.Color(color), [color]);
  useFrame(({ clock }) => {
    if (groupRef.current) {
      groupRef.current.position.y = pos[1] + Math.sin(clock.elapsedTime * 0.8 + pos[0]) * 0.3;
    }
  });
  return (
    <group
      ref={groupRef}
      position={pos}
      onPointerOver={(e) => { e.stopPropagation(); onHover(node.id); }}
      onPointerOut={() => onHover(null)}
      onClick={(e) => { e.stopPropagation(); onClick(node.id); }}
    >
      {/* Light outer shell — reads as a crisp rim against the dark scene */}
      <mesh renderOrder={0}>
        <sphereGeometry args={[1.42, 32, 32]} />
        <meshStandardMaterial
          color="#f1f5f9"
          metalness={0.35}
          roughness={0.38}
          emissive="#e2e8f0"
          emissiveIntensity={0.22}
        />
      </mesh>
      {/* Saturated core with clearcoat for a subtle “glass / 3D” highlight */}
      <mesh renderOrder={1}>
        <sphereGeometry args={[1.12, 32, 32]} />
        <meshPhysicalMaterial
          color={color}
          metalness={0.55}
          roughness={0.22}
          clearcoat={0.55}
          clearcoatRoughness={0.28}
          emissive={coreColor}
          emissiveIntensity={0.18}
        />
      </mesh>
    </group>
  );
}

// ─── Graph scene ─────────────────────────────────────────────────────────────

function GraphScene({
  overview,
  positions,
}: {
  overview: GraphOverview;
  positions: Map<string, [number, number, number]>;
}) {
  const navigate = useNavigate();
  const [hovered, setHovered] = useState<string | null>(null);

  const { gl } = useThree();
  useEffect(() => {
    gl.domElement.style.cursor = hovered ? "pointer" : "default";
  }, [hovered, gl]);

  return (
    <>
      <ambientLight intensity={0.5} />
      <pointLight position={[80, 80, 80]} intensity={1.2} />
      <pointLight position={[-80, -80, 40]} intensity={0.5} color="#a78bfa" />

      {overview.edges.map((e) => {
        const src = positions.get(e.from_id);
        const tgt = positions.get(e.to_id);
        if (!src || !tgt) return null;
        return (
          <GraphEdgeLine key={e.edge_id} from={src} to={tgt} />
        );
      })}

      {overview.nodes.map((n) => {
        const pos = positions.get(n.id);
        if (!pos) return null;
        return (
          <group key={n.id}>
            <GraphNode
              node={n}
              pos={pos}
              onHover={setHovered}
              onClick={(id) => navigate(`/contexts/${encodeURIComponent(id)}`)}
            />
            {hovered === n.id && (
              <Html position={pos} center distanceFactor={60}>
                <div
                  style={{
                    background: "rgba(15,15,25,0.92)",
                    border: "1px solid rgba(99,102,241,0.5)",
                    borderRadius: 8,
                    padding: "6px 10px",
                    fontSize: 12,
                    color: "#e2e8f0",
                    pointerEvents: "none",
                    whiteSpace: "nowrap",
                    maxWidth: 240,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                  }}
                >
                  <div style={{ fontWeight: 600, color: "#a5b4fc", marginBottom: 2 }}>
                    {n.collection || "(no collection)"}
                  </div>
                  <div style={{ opacity: 0.85 }}>{n.label || n.id}</div>
                </div>
              </Html>
            )}
          </group>
        );
      })}
    </>
  );
}

// ─── Engine layout scene ──────────────────────────────────────────────────────

function EngineBox({
  label,
  sublabel,
  position,
  scale,
  color,
  emissive,
}: {
  label: string;
  sublabel: string;
  position: [number, number, number];
  scale: [number, number, number];
  color: string;
  emissive?: string;
}) {
  const meshRef = useRef<THREE.Mesh>(null!);
  const [hovered, setHovered] = useState(false);
  const { gl } = useThree();
  useEffect(() => {
    gl.domElement.style.cursor = hovered ? "help" : "default";
  }, [hovered, gl]);

  return (
    <group position={position}>
      <mesh
        ref={meshRef}
        scale={scale}
        onPointerOver={() => setHovered(true)}
        onPointerOut={() => setHovered(false)}
      >
        <boxGeometry args={[1, 1, 1]} />
        <meshStandardMaterial
          color={color}
          emissive={emissive ?? color}
          emissiveIntensity={hovered ? 0.5 : 0.15}
          roughness={0.35}
          metalness={0.5}
          transparent
          opacity={0.88}
        />
      </mesh>
      <Html position={[0, scale[1] / 2 + 1.5, 0]} center>
        <div
          style={{
            background: "rgba(10,10,20,0.88)",
            border: "1px solid rgba(148,163,184,0.2)",
            borderRadius: 6,
            padding: "4px 8px",
            fontSize: 11,
            color: "#cbd5e1",
            pointerEvents: "none",
            whiteSpace: "nowrap",
            textAlign: "center",
          }}
        >
          <div style={{ fontWeight: 700, color: "#f8fafc", letterSpacing: "0.03em" }}>{label}</div>
          <div style={{ opacity: 0.7, marginTop: 1 }}>{sublabel}</div>
        </div>
      </Html>
    </group>
  );
}

function EngineScene({ diag }: { diag: StorageDiagnostics }) {
  const walKB = (diag.wal_bytes / 1024).toFixed(1);
  const walH = Math.max(2, Math.log2(diag.wal_bytes + 2) * 1.5);
  const memH = Math.max(2, Math.log2(diag.mem_entries + 2) * 2);

  // Floor dimensions
  const maxSSTBytes = diag.sstables.reduce((m, s) => Math.max(m, s.bytes), 1);

  return (
    <>
      <ambientLight intensity={0.45} />
      <pointLight position={[60, 80, 60]} intensity={1.5} color="#ffffff" />
      <pointLight position={[-60, 30, -40]} intensity={0.6} color="#818cf8" />

      {/* Floor grid */}
      <gridHelper args={[200, 40, "#334155", "#1e293b"]} position={[0, -1, 0]} />

      {/* WAL block */}
      <EngineBox
        label="WAL"
        sublabel={`${walKB} KB`}
        position={[-30, walH / 2 - 1, 0]}
        scale={[12, walH, 12]}
        color="#f59e0b"
        emissive="#d97706"
      />

      {/* MemTable block */}
      <EngineBox
        label="MemTable"
        sublabel={`${diag.mem_entries} entries`}
        position={[0, memH / 2 - 1, 0]}
        scale={[14, memH, 14]}
        color="#6366f1"
        emissive="#4f46e5"
      />

      {/* SSTable columns */}
      {diag.sstables.map((sst, i) => {
        const sstKB = (sst.bytes / 1024).toFixed(1);
        const sstH = Math.max(2, Math.log2(sst.bytes + 2) / Math.log2(maxSSTBytes + 2) * 30 + 2);
        const x = 26 + i * 16;
        return (
          <EngineBox
            key={sst.path}
            label={`SST #${sst.seq}`}
            sublabel={`${sstKB} KB`}
            position={[x, sstH / 2 - 1, 0]}
            scale={[12, sstH, 12]}
            color="#10b981"
            emissive="#059669"
          />
        );
      })}

      {diag.sstables.length === 0 && (
        <Html position={[30, 5, 0]} center>
          <div style={{ color: "#64748b", fontSize: 12, fontStyle: "italic", pointerEvents: "none" }}>
            No SST files yet
          </div>
        </Html>
      )}
    </>
  );
}

// ─── WebGL fallbacks (embedded browsers / context loss) ───────────────────────

function GraphOverviewFallback({ overview }: { overview: GraphOverview }) {
  const navigate = useNavigate();
  return (
    <div
      style={{
        width: "100%",
        maxHeight: 280,
        overflow: "auto",
        textAlign: "left",
        borderTop: "1px solid rgba(148,163,184,0.25)",
        marginTop: 12,
        paddingTop: 12,
      }}
    >
      <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8, color: "var(--on-surface)" }}>
        Graph data (list view)
      </div>
      <ul style={{ margin: 0, paddingLeft: 18, fontSize: 13, lineHeight: 1.45 }}>
        {overview.nodes.map((n) => (
          <li key={n.id} style={{ marginBottom: 6 }}>
            <button
              type="button"
              onClick={() => navigate(`/contexts/${encodeURIComponent(n.id)}`)}
              style={{
                background: "none",
                border: "none",
                padding: 0,
                cursor: "pointer",
                color: "var(--primary)",
                textDecoration: "underline",
                font: "inherit",
              }}
            >
              {n.label || n.id}
            </button>
            <span style={{ color: "var(--on-surface-variant)", marginLeft: 6 }}>
              ({n.collection || "—"})
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function EngineOverviewFallback({ diag }: { diag: StorageDiagnostics }) {
  return (
    <div
      style={{
        width: "100%",
        maxHeight: 280,
        overflow: "auto",
        textAlign: "left",
        borderTop: "1px solid rgba(148,163,184,0.25)",
        marginTop: 12,
        paddingTop: 12,
      }}
    >
      <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 8, color: "var(--on-surface)" }}>
        Engine layout (text view)
      </div>
      <div style={{ fontSize: 13, lineHeight: 1.5, color: "var(--on-surface-variant)" }}>
        <div>WAL: {(diag.wal_bytes / 1024).toFixed(1)} KB</div>
        <div>MemTable: {diag.mem_entries} entries</div>
        <div style={{ marginTop: 8 }}>SSTables:</div>
        <ul style={{ margin: "4px 0 0 18px", padding: 0 }}>
          {diag.sstables.length === 0 ? (
            <li>None yet</li>
          ) : (
            diag.sstables.map((s) => (
              <li key={s.path}>
                #{s.seq} — {(s.bytes / 1024).toFixed(1)} KB — {s.path}
              </li>
            ))
          )}
        </ul>
      </div>
    </div>
  );
}

// ─── Main component ────────────────────────────────────────────────────────────

type Tab = "graph" | "engine";

/** One perspective per tab; avoids remounting Canvas when switching tabs (WebGL context loss). */
function CameraRig({ tab }: { tab: Tab }) {
  const { camera } = useThree();
  useEffect(() => {
    const p = camera as THREE.PerspectiveCamera;
    p.near = 0.1;
    p.far = 100000;
    if (tab === "graph") {
      camera.position.set(0, 0, 220);
      p.fov = 50;
    } else {
      camera.position.set(60, 60, 120);
      p.fov = 45;
    }
    p.updateProjectionMatrix();
  }, [tab, camera]);
  return null;
}

/** Solid background + context-loss reporting (embedded browsers / GPU limits often lose WebGL). */
function SceneSetup({ onContextLost }: { onContextLost: () => void }) {
  const { gl, scene } = useThree();
  useEffect(() => {
    scene.background = new THREE.Color(0x0f172a);
    gl.setClearColor(0x0f172a, 1);
  }, [gl, scene]);

  useEffect(() => {
    const el = gl.domElement;
    const onLost = (e: Event) => {
      e.preventDefault();
      onContextLost();
    };
    el.addEventListener("webglcontextlost", onLost);
    return () => el.removeEventListener("webglcontextlost", onLost);
  }, [gl, onContextLost]);

  return null;
}

export default function StorageExplorer3D() {
  const [tab, setTab] = useState<Tab>("graph");
  const [overview, setOverview] = useState<GraphOverview | null>(null);
  const [diag, setDiag] = useState<StorageDiagnostics | null>(null);
  const [graphError, setGraphError] = useState<string | null>(null);
  const [diagError, setDiagError] = useState<string | null>(null);
  const [loadingGraph, setLoadingGraph] = useState(true);
  const [loadingDiag, setLoadingDiag] = useState(true);
  const [webglLost, setWebglLost] = useState(false);
  /** Bumps to remount `<Canvas>` after WebGL context loss or flaky GPU stacks. */
  const [canvasMountKey, setCanvasMountKey] = useState(0);
  const positionsRef = useRef<Map<string, [number, number, number]>>(new Map());
  const [positions, setPositions] = useState<Map<string, [number, number, number]>>(new Map());

  const loadGraph = useCallback(async () => {
    setLoadingGraph(true);
    setGraphError(null);
    try {
      const raw = await getGraphOverview({ limit: 500 });
      const data: GraphOverview = {
        nodes: raw?.nodes ?? [],
        edges: raw?.edges ?? [],
        truncated: Boolean(raw?.truncated),
      };
      const pos = buildForceLayout(data.nodes, data.edges);
      positionsRef.current = pos;
      setPositions(pos);
      setOverview(data);
    } catch (e) {
      setGraphError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoadingGraph(false);
    }
  }, []);

  const loadDiag = useCallback(async () => {
    setLoadingDiag(true);
    setDiagError(null);
    try {
      setDiag(await getStorageDiagnostics());
    } catch (e) {
      setDiagError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoadingDiag(false);
    }
  }, []);

  useEffect(() => { void loadGraph(); }, [loadGraph]);
  useEffect(() => { void loadDiag(); }, [loadDiag]);

  const canvasStyle: React.CSSProperties = {
    position: "relative",
    width: "100%",
    flexShrink: 0,
    minHeight: 420,
    height: "max(420px, min(70dvh, calc(100vh - 200px)))",
    borderRadius: 12,
    overflow: "hidden",
    isolation: "isolate",
    background: "linear-gradient(160deg, #0a0a14 0%, #0f172a 60%, #1a1040 100%)",
    border: "1px solid var(--outline-variant)",
  };

  const showGraphCanvas =
    tab === "graph" &&
    !loadingGraph &&
    !graphError &&
    overview !== null &&
    overview.nodes.length > 0;

  const showEngineCanvas =
    tab === "engine" &&
    !loadingDiag &&
    !diagError &&
    diag !== null;

  const showCanvas = showGraphCanvas || showEngineCanvas;

  const explorerStatusText: string | null =
    tab === "graph"
      ? loadingGraph
        ? "Computing force layout…"
        : graphError
          ? `Error: ${graphError}`
          : overview && overview.nodes.length === 0
            ? "No graph data yet. Store some context entries and link them."
            : null
      : loadingDiag
        ? "Loading diagnostics…"
        : diagError
          ? `Error: ${diagError}`
          : null;

  const showExplorerStatusOverlay = !showCanvas && !webglLost && explorerStatusText !== null;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
      <div style={{ display: "flex", alignItems: "baseline", justifyContent: "space-between" }}>
        <div>
          <h1 style={{ fontFamily: "var(--font-headline)", fontSize: 24, fontWeight: 800, margin: 0 }}>
            3D Explorer
          </h1>
          <p style={{ color: "var(--on-surface-variant)", marginTop: 4, fontSize: 14 }}>
            Interactive 3D views of your context graph and storage engine internals.
          </p>
        </div>
        <button
          onClick={() => {
            setWebglLost(false);
            setCanvasMountKey((k) => k + 1);
            if (tab === "graph") void loadGraph();
            else void loadDiag();
          }}
          style={{
            background: "var(--surface-container-high)",
            border: "1px solid var(--outline-variant)",
            borderRadius: "var(--radius)",
            color: "var(--on-surface)",
            fontSize: 13,
            padding: "7px 14px",
            cursor: "pointer",
          }}
        >
          Refresh
        </button>
      </div>

      {/* Tab bar */}
      <div style={{ display: "flex", gap: 8 }}>
        {(["graph", "engine"] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            style={{
              padding: "8px 20px",
              borderRadius: "var(--radius)",
              border: "1px solid var(--outline-variant)",
              background: tab === t ? "var(--primary)" : "var(--surface-container)",
              color: tab === t ? "var(--on-primary)" : "var(--on-surface-variant)",
              fontWeight: 600,
              fontSize: 13,
              cursor: "pointer",
              transition: "all 0.15s",
            }}
          >
            {t === "graph" ? "Context Graph" : "Engine Layout"}
          </button>
        ))}
      </div>

      {/* Single viewport: one WebGL Canvas shared across tabs. Canvas is painted first; overlays use z-index so status text is never covered. */}
      <div style={canvasStyle}>
        {showCanvas && !webglLost && (
          <Canvas
            key={canvasMountKey}
            camera={{ position: [0, 0, 220], fov: 50, near: 0.1, far: 100000 }}
            gl={{
              alpha: false,
              antialias: true,
              powerPreference: "default",
              failIfMajorPerformanceCaveat: false,
            }}
            dpr={[1, 1.5]}
            style={{ position: "absolute", inset: 0, zIndex: 1, width: "100%", height: "100%" }}
            onCreated={({ scene }) => {
              scene.background = new THREE.Color(0x0f172a);
            }}
          >
            <SceneSetup onContextLost={() => setWebglLost(true)} />
            <CameraRig tab={tab} />
            <Suspense fallback={null}>
              {showGraphCanvas && overview && (
                <GraphScene overview={overview} positions={positions} />
              )}
              {showEngineCanvas && diag && <EngineScene diag={diag} />}
              <OrbitControls
                makeDefault
                enableDamping
                dampingFactor={0.08}
                target={[0, 0, 0]}
              />
            </Suspense>
          </Canvas>
        )}

        {webglLost && (
          <div
            role="alert"
            style={{
              position: "absolute",
              inset: 0,
              zIndex: 8,
              display: "flex",
              flexDirection: "column",
              alignItems: "stretch",
              justifyContent: "flex-start",
              gap: 0,
              padding: 20,
              overflow: "auto",
              background: "rgba(10, 10, 20, 0.96)",
              color: "var(--on-surface-variant)",
              fontSize: 14,
              textAlign: "center",
            }}
          >
            <div style={{ marginBottom: 12 }}>
              WebGL is unavailable or the context was lost (common in embedded browsers, remote preview, or after GPU sleep).
              You can still use the data below.
            </div>
            <button
              type="button"
              onClick={() => {
                setWebglLost(false);
                setCanvasMountKey((k) => k + 1);
                void loadGraph();
                void loadDiag();
              }}
              style={{
                alignSelf: "center",
                background: "var(--primary)",
                color: "var(--on-primary)",
                border: "none",
                borderRadius: "var(--radius)",
                padding: "8px 18px",
                fontWeight: 600,
                cursor: "pointer",
              }}
            >
              Retry 3D view
            </button>
            {tab === "graph" && overview && overview.nodes.length > 0 && (
              <GraphOverviewFallback overview={overview} />
            )}
            {tab === "engine" && diag && <EngineOverviewFallback diag={diag} />}
          </div>
        )}

        {showExplorerStatusOverlay && (
          <div
            role="status"
            style={{
              position: "absolute",
              inset: 0,
              zIndex: 4,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              padding: 24,
              color: "var(--on-surface-variant)",
              fontSize: 14,
              textAlign: "center",
            }}
          >
            {explorerStatusText}
          </div>
        )}

        {tab === "graph" && !loadingGraph && !graphError && overview && overview.truncated && overview.nodes.length > 0 && showCanvas && !webglLost && (
          <div
            style={{
              position: "absolute",
              bottom: 12,
              left: "50%",
              transform: "translateX(-50%)",
              zIndex: 6,
              background: "rgba(245,158,11,0.15)",
              border: "1px solid rgba(245,158,11,0.4)",
              borderRadius: 8,
              padding: "4px 12px",
              fontSize: 12,
              color: "#fbbf24",
              pointerEvents: "none",
            }}
          >
            Graph truncated at 500 nodes
          </div>
        )}
      </div>

      {/* Legend */}
      {tab === "engine" && diag && !loadingDiag && !diagError && (
        <div style={{
          display: "flex",
          gap: 24,
          flexWrap: "wrap",
          fontSize: 12,
          color: "var(--on-surface-variant)",
        }}>
          {[
            { color: "#f59e0b", label: "WAL — write-ahead log" },
            { color: "#6366f1", label: "MemTable — in-memory sorted buffer" },
            { color: "#10b981", label: "SSTables — immutable on-disk files" },
          ].map(({ color, label }) => (
            <div key={label} style={{ display: "flex", alignItems: "center", gap: 6 }}>
              <div style={{ width: 10, height: 10, borderRadius: 2, background: color }} />
              {label}
            </div>
          ))}
          <div style={{ marginLeft: "auto" }}>
            Height ∝ log(bytes) — hover boxes for details
          </div>
        </div>
      )}
      {tab === "graph" && overview && !loadingGraph && !graphError && overview.nodes.length > 0 && (
        <div style={{ fontSize: 12, color: "var(--on-surface-variant)" }}>
          {overview.nodes.length} nodes · {overview.edges.length} edges — click a node to open its detail page
        </div>
      )}
    </div>
  );
}
