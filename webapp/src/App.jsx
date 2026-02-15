import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { useEffect } from 'react'
import { useStore } from './store/useStore'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Timeline from './pages/Timeline'
import Upload from './pages/Upload'
import Player from './pages/Player'
import Viewer from './pages/Viewer'
import ServerManagement from './pages/ServerManagement'
import Settings from './pages/Settings'
import Login from './pages/Login'

function App() {
  const { serverUrl, isConnected, checkConnection, detectPlatform } = useStore()

  useEffect(() => {
    // Detect platform on mount
    detectPlatform()
    
    // Check server connection
    if (serverUrl) {
      checkConnection()
    }
  }, [serverUrl, checkConnection, detectPlatform])

  // Show login if not connected
  if (!isConnected) {
    return <Login />
  }

  return (
    <Router future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/timeline" element={<Timeline />} />
          <Route path="/upload" element={<Upload />} />
          <Route path="/player/:hash" element={<Player />} />
          <Route path="/viewer/:hash" element={<Viewer />} />
          <Route path="/server" element={<ServerManagement />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Layout>
    </Router>
  )
}

export default App
