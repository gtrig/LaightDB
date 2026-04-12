import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { AuthProvider } from "./hooks/useAuth";
import App from "./App";
import "./styles/global.css";

// R3F v9 still uses THREE.Clock; three r183 deprecates it. R3F v10 (Timer) is not yet Vite-safe with three@0.183 here.
{
  const needle = "THREE.THREE.Clock: This module has been deprecated";
  const wrap = (fn: (...args: unknown[]) => void) => (...args: unknown[]) => {
    if (typeof args[0] === "string" && args[0].includes(needle)) return;
    fn.apply(console, args);
  };
  console.warn = wrap(console.warn.bind(console));
  console.error = wrap(console.error.bind(console));
}

createRoot(document.getElementById("root")!).render(
  <BrowserRouter>
    <AuthProvider>
      <App />
    </AuthProvider>
  </BrowserRouter>,
);
