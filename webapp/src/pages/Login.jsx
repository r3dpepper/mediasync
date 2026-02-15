import { useState, useEffect } from 'react'
import { useStore } from '../store/useStore'
import { Wifi, Server, Loader } from 'lucide-react'

export default function Login() {
  const [discovering, setDiscovering] = useState(false)
  const [manualUrl, setManualUrl] = useState('')
  const [deviceName, setDeviceName] = useState('')
  const [error, setError] = useState(null)
  
  const { setServerUrl, registerDevice, checkConnection, platform } = useStore()

  useEffect(() => {
    // Auto-detect device name
    const ua = navigator.userAgent
    if (ua.includes('iPhone')) setDeviceName('iPhone')
    else if (ua.includes('iPad')) setDeviceName('iPad')
    else if (ua.includes('Android')) setDeviceName('Android Device')
    else if (ua.includes('Fire')) setDeviceName('Fire TV')
    else setDeviceName('Web Browser')
  }, [])

  const discoverServer = async () => {
    setDiscovering(true)
    setError(null)
    
    try {
      // Try common local addresses
      const commonAddresses = [
        'http://localhost:8080',
        'http://media-server.local:8080',
        'http://192.168.12.152:8080',
      ]
      
      for (const url of commonAddresses) {
        try {
          setServerUrl(url)
          const connected = await checkConnection()
          if (connected) {
            await registerDevice(deviceName, platform)
            return
          }
        } catch (e) {
          continue
        }
      }
      
      setError('No server found. Please enter server address manually.')
    } catch (err) {
      setError('Discovery failed: ' + err.message)
    } finally {
      setDiscovering(false)
    }
  }

  const connectManually = async () => {
    if (!manualUrl) {
      setError('Please enter server address')
      return
    }
    
    setError(null)
    try {
      setServerUrl(manualUrl)
      const connected = await checkConnection()
      if (connected) {
        await registerDevice(deviceName, platform)
      } else {
        setError('Could not connect to server')
      }
    } catch (err) {
      setError('Connection failed: ' + err.message)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-900 via-blue-900 to-slate-900 p-4">
      <div className="card p-8 max-w-md w-full">
        <div className="text-center mb-8">
          <div className="w-16 h-16 bg-blue-600 rounded-full flex items-center justify-center mx-auto mb-4">
            <Server className="w-8 h-8" />
          </div>
          <h1 className="text-3xl font-bold mb-2">Private Media</h1>
          <p className="text-slate-400">Connect to your media server</p>
          <p className="text-sm text-slate-500 mt-1">Platform: {platform}</p>
        </div>

        <div className="space-y-6">
          {/* Auto-discover */}
          <div>
            <button
              onClick={discoverServer}
              disabled={discovering}
              className="btn btn-primary w-full flex items-center justify-center gap-2"
            >
              {discovering ? (
                <>
                  <Loader className="w-5 h-5 animate-spin" />
                  Discovering...
                </>
              ) : (
                <>
                  <Wifi className="w-5 h-5" />
                  Discover Server
                </>
              )}
            </button>
          </div>

          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-slate-700"></div>
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="px-2 bg-slate-800 text-slate-400">or</span>
            </div>
          </div>

          {/* Manual entry */}
          <div className="space-y-3">
            <input
              type="text"
              value={deviceName}
              onChange={(e) => setDeviceName(e.target.value)}
              placeholder="Device Name"
              className="input"
            />
            
            <input
              type="text"
              value={manualUrl}
              onChange={(e) => setManualUrl(e.target.value)}
              placeholder="http://192.168.1.100:8080"
              className="input"
            />
            
            <button
              onClick={connectManually}
              className="btn btn-secondary w-full"
            >
              Connect Manually
            </button>
          </div>

          {error && (
            <div className="bg-red-900/20 border border-red-700 text-red-300 px-4 py-3 rounded-lg text-sm">
              {error}
            </div>
          )}
        </div>

        <div className="mt-8 text-center text-sm text-slate-500">
          <p>Make sure your server is running on the local network</p>
        </div>
      </div>
    </div>
  )
}
