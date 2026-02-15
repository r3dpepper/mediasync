import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { mediaApi } from '../api/client'
import { HardDrive, Image, Video, Upload, Server } from 'lucide-react'

export default function Dashboard() {
  const [stats, setStats] = useState(null)
  const [health, setHealth] = useState(null)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [statsRes, healthRes] = await Promise.all([
        mediaApi.getStats(),
        mediaApi.getHealth()
      ])
      setStats(statsRes.data)
      setHealth(healthRes.data)
    } catch (error) {
      console.error('Failed to load dashboard:', error)
    }
  }

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
  }

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">Dashboard</h1>

      {/* Server Status */}
      {health && (
        <div className="card p-6">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-semibold mb-1">Server Status</h2>
              <p className="text-slate-400">Version {health.version}</p>
            </div>
            <div className={`px-4 py-2 rounded-full ${
              health.status === 'healthy' 
                ? 'bg-green-900/20 text-green-400' 
                : 'bg-red-900/20 text-red-400'
            }`}>
              {health.status === 'healthy' ? '● Online' : '● Offline'}
            </div>
          </div>
        </div>
      )}

      {/* Stats */}
      {stats && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <StatCard
              icon={<HardDrive />}
              label="Total Size"
              value={formatBytes(stats.total_size_bytes)}
              color="blue"
            />
            <StatCard
              icon={<Image />}
              label="Images"
              value={stats.files_by_type.image || 0}
              color="green"
            />
            <StatCard
              icon={<Video />}
              label="Videos"
              value={stats.files_by_type.video || 0}
              color="purple"
            />
          </div>

          {/* Quick Actions */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Link to="/timeline" className="card p-6 hover:bg-slate-700 transition-colors">
              <Image className="w-8 h-8 mb-3 text-blue-400" />
              <h3 className="text-lg font-semibold mb-1">View Timeline</h3>
              <p className="text-slate-400">Browse your media chronologically</p>
            </Link>
            
            <Link to="/upload" className="card p-6 hover:bg-slate-700 transition-colors">
              <Upload className="w-8 h-8 mb-3 text-green-400" />
              <h3 className="text-lg font-semibold mb-1">Upload Media</h3>
              <p className="text-slate-400">Add photos and videos</p>
            </Link>
          </div>
        </>
      )}
    </div>
  )
}

function StatCard({ icon, label, value, color }) {
  const colors = {
    blue: 'text-blue-400',
    green: 'text-green-400',
    purple: 'text-purple-400'
  }

  return (
    <div className="card p-6">
      <div className={`w-12 h-12 rounded-lg bg-${color}-900/20 flex items-center justify-center mb-3 ${colors[color]}`}>
        {icon}
      </div>
      <div className="text-2xl font-bold">{value}</div>
      <div className="text-sm text-slate-400">{label}</div>
    </div>
  )
}
