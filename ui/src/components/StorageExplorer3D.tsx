import { Suspense, useEffect, useRef, useState, useCallback } from "react";
import { Canvas, useFrame, useThree } from "@react-three/fiber";
import { OrbitControls, Html, Line } from "@react-three/drei";
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
        const d2 = dx * dx + dy * dy + dz * dz;
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

  const positions = new Map<string, [number, number, number]>();
  for (const n of fNodes) positions.set(n.id, [n.x, n.y, n.z]);
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
  const meshRef = useRef<THREE.Mesh>(null!);
  const color = collectionColor(node.collection);
  useFrame(({ clock }) => {
    if (meshRef.current) {
      meshRef.current.position.y = pos[1] + Math.sin(clock.elapsedTime * 0.8 + pos[0]) * 0.3;
    }
  });
  return (
    <mesh
      ref={meshRef}
      position={pos}
      onPointerOver={(e) => { e.stopPropagation(); onHover(node.id); }}
      onPointerOut={() => onHover(null)}
      onClick={(e) => { e.stopPropagation(); onClick(node.id); }}
    >
      <sphereGeometry args={[1.2, 16, 16]} />
      <meshStandardMaterial color={color} roughness={0.3} metalness={0.4} />
    </mesh>
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
          <Line
            key={e.edge_id}
            points={[src, tgt]}
            color="#4f46e5"
            lineWidth={1}
            transparent
            opacity={0.35}
          />
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

// ─── Loading / error helpers ──────────────────────────────────────────────────

function StatusBox({ children }: { children: React.ReactNode }) {
  return (
    <div style={{
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      height: "100%",
      color: "var(--on-surface-variant)",
      fontSize: 14,
    }}>
      {children}
    </div>
  );
}

// ─── Main component ────────────────────────────────────────────────────────────

type Tab = "graph" | "engine";

/** One perspective per tab; avoids remounting Canvas when switching tabs (WebGL context loss). */
function CameraRig({ tab }: { tab: Tab }) {
  const { camera } = useThree();
  useEffect(() => {
    if (tab === "graph") {
      camera.position.set(0, 0, 160);
      if ("fov" in camera) (camera as THREE.PerspectiveCamera).fov = 50;
    } else {
      camera.position.set(60, 60, 120);
      if ("fov" in camera) (camera as THREE.PerspectiveCamera).fov = 45;
    }
    camera.updateProjectionMatrix();
  }, [tab, camera]);
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
  const positionsRef = useRef<Map<string, [number, number, number]>>(new Map());
  const [positions, setPositions] = useState<Map<string, [number, number, number]>>(new Map());

  const loadGraph = useCallback(async () => {
    setLoadingGraph(true);
    setGraphError(null);
    try {
      const data = await getGraphOverview({ limit: 500 });
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
    height: "calc(100vh - 220px)",
    minHeight: 420,
    borderRadius: 12,
    overflow: "hidden",
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
          onClick={tab === "graph" ? loadGraph : loadDiag}
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

      {/* Single viewport: one WebGL Canvas shared across tabs to avoid context loss when switching. */}
      <div style={canvasStyle}>
        {tab === "graph" && loadingGraph && <StatusBox>Computing force layout…</StatusBox>}
        {tab === "graph" && graphError && <StatusBox>Error: {graphError}</StatusBox>}
        {tab === "graph" && !loadingGraph && !graphError && overview && overview.nodes.length === 0 && (
          <StatusBox>No graph data yet. Store some context entries and link them.</StatusBox>
        )}

        {tab === "engine" && loadingDiag && <StatusBox>Loading diagnostics…</StatusBox>}
        {tab === "engine" && diagError && <StatusBox>Error: {diagError}</StatusBox>}

        {tab === "graph" && !loadingGraph && !graphError && overview && overview.truncated && overview.nodes.length > 0 && (
          <div
            style={{
              position: "absolute",
              bottom: 12,
              left: "50%",
              transform: "translateX(-50%)",
              zIndex: 2,
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

        {showCanvas && (
          <Canvas
            camera={{ position: [0, 0, 160], fov: 50 }}
            style={{ position: "absolute", inset: 0, width: "100%", height: "100%" }}
          >
            <CameraRig tab={tab} />
            <Suspense fallback={null}>
              {showGraphCanvas && overview && (
                <GraphScene overview={overview} positions={positions} />
              )}
              {showEngineCanvas && diag && <EngineScene diag={diag} />}
              <OrbitControls enableDamping dampingFactor={0.08} />
            </Suspense>
          </Canvas>
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
