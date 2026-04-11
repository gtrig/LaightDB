import { Routes, Route, Navigate } from "react-router-dom";
import Layout from "./components/Layout";
import Dashboard from "./components/Dashboard";
import SearchPanel from "./components/SearchPanel";
import StoreForm from "./components/StoreForm";
import ContextDetail from "./components/ContextDetail";
import CollectionsPage from "./components/CollectionsPage";
import CollectionBrowse from "./components/CollectionBrowse";
import SystemPage from "./components/SystemPage";

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="search" element={<SearchPanel />} />
        <Route path="store" element={<StoreForm />} />
        <Route path="collections" element={<CollectionsPage />} />
        <Route path="collections/:name" element={<CollectionBrowse />} />
        <Route path="system" element={<SystemPage />} />
        <Route path="contexts/:id" element={<ContextDetail />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
