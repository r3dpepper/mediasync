import { useStore } from '../store/useStore'
import { Monitor, Smartphone, Tv } from 'lucide-react'

export default function Settings() {
  const { platform, deviceName, serverUrl } = useStore()

  const getPlatformIcon = () => {
    switch (platform) {
      case 'tv': return <Tv className="w-8 h-8" />
      case 'mobile': return <Smartphone className="w-8 h-8" />
      default: return <Monitor className="w-8 h-8" />
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">Settings</h1>

      <div className="card p-6">
        <h2 className="text-xl font-semibold mb-4">Device Information</h2>
        <div className="space-y-4">
          <div className="flex items-center gap-4">
            <div className="text-blue-400">
              {getPlatformIcon()}
            </div>
            <div>
              <div className="font-medium">{deviceName || 'Web Client'}</div>
              <div className="text-sm text-slate-400">Platform: {platform}</div>
            </div>
          </div>
          
          <div>
            <div className="text-sm text-slate-400 mb-1">Server URL</div>
            <div className="font-mono text-sm">{serverUrl}</div>
          </div>
          
          <div>
            <div className="text-sm text-slate-400 mb-1">User Agent</div>
            <div className="text-xs text-slate-500 break-all">
              {navigator.userAgent}
            </div>
          </div>
        </div>
      </div>

      <div className="card p-6">
        <h2 className="text-xl font-semibold mb-4">About</h2>
        <div className="space-y-2 text-slate-300">
          <div>Private Media Ecosystem</div>
          <div className="text-sm text-slate-400">Version 1.0.0</div>
          <div className="text-sm text-slate-400">
            Universal web interface for managing your private media server
          </div>
        </div>
      </div>
    </div>
  )
}
