import { Routes, Route, Navigate } from "react-router-dom";
import Layout from "./components/Layout";
import Dashboard from "./components/Dashboard";
import SearchPanel from "./components/SearchPanel";
import StoreForm from "./components/StoreForm";
import ContextDetail from "./components/ContextDetail";

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="search" element={<SearchPanel />} />
        <Route path="store" element={<StoreForm />} />
        <Route path="contexts/:id" element={<ContextDetail />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
