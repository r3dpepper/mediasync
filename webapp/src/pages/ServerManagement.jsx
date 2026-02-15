import { useState, useEffect } from 'react'
import { useStore } from '../store/useStore'
import { mediaApi } from '../api/client'
import { Database, HardDrive, RefreshCw, Settings as SettingsIcon, Trash2 } from 'lucide-react'

export function ServerManagement() {
  const [backupStatus, setBackupStatus] = useState(null)
  const [loading, setLoading] = useState(false)
  const [resyncLoading, setResyncLoading] = useState(false)
  const [resyncResult, setResyncResult] = useState(null)
  const [truncateLoading, setTruncateLoading] = useState(false)
  const [config, setConfig] = useState(null)
  const [editingConfig, setEditingConfig] = useState(false)
  const [configForm, setConfigForm] = useState({})

  useEffect(() => {
    loadBackupStatus()
    loadConfig()
  }, [])

  const loadBackupStatus = async () => {
    try {
      const response = await mediaApi.getBackupStatus()
      setBackupStatus(response.data)
    } catch (error) {
      console.error('Failed to load backup status:', error)
    }
  }

  const loadConfig = async () => {
    try {
      const response = await mediaApi.getConfig()
      setConfig(response.data)
      setConfigForm(response.data)
    } catch (error) {
      console.error('Failed to load config:', error)
    }
  }

  const startBackup = async () => {
    setLoading(true)
    try {
      await mediaApi.startBackup('incremental')
      await loadBackupStatus()
    } catch (error) {
      console.error('Failed to start backup:', error)
    } finally {
      setLoading(false)
    }
  }

  const startResync = async () => {
    setResyncLoading(true)
    setResyncResult(null)
    try {
      const response = await mediaApi.startResync()
      setResyncResult(response.data)
    } catch (error) {
      console.error('Failed to start resync:', error)
      setResyncResult({ error: error.message })
    } finally {
      setResyncLoading(false)
    }
  }

  const truncateDatabase = async () => {
    if (!confirm('Are you sure? This will delete all media records from the database. Files will not be deleted.')) {
      return
    }
    setTruncateLoading(true)
    try {
      const response = await mediaApi.truncateDatabase()
      alert(response.data.message)
    } catch (error) {
      alert('Failed to truncate database: ' + error.message)
    } finally {
      setTruncateLoading(false)
    }
  }

  const saveConfig = async () => {
    try {
      const response = await mediaApi.updateConfig(configForm)
      alert(response.data.message)
      setConfig(response.data.config)
      setEditingConfig(false)
    } catch (error) {
      alert('Failed to save config: ' + error.message)
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">Server Management</h1>

      {config && (
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-xl font-semibold">Configuration</h2>
              <p className="text-slate-400">Server settings</p>
            </div>
            <button
              onClick={() => editingConfig ? saveConfig() : setEditingConfig(true)}
              className="btn btn-primary flex items-center gap-2"
            >
              <SettingsIcon className="w-4 h-4" />
              {editingConfig ? 'Save' : 'Edit'}
            </button>
          </div>

          {editingConfig ? (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">Primary Storage Path</label>
                <input
                  type="text"
                  value={configForm.Storage?.PrimaryPath || ''}
                  onChange={(e) => setConfigForm({...configForm, Storage: {...configForm.Storage, PrimaryPath: e.target.value}})}
                  className="w-full px-3 py-2 bg-slate-800 rounded border border-slate-700"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Backup Path</label>
                <input
                  type="text"
                  value={configForm.Storage?.BackupPath || ''}
                  onChange={(e) => setConfigForm({...configForm, Storage: {...configForm.Storage, BackupPath: e.target.value}})}
                  className="w-full px-3 py-2 bg-slate-800 rounded border border-slate-700"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Server Port</label>
                <input
                  type="number"
                  value={configForm.Server?.Port || 8080}
                  onChange={(e) => setConfigForm({...configForm, Server: {...configForm.Server, Port: parseInt(e.target.value)}})}
                  className="w-full px-3 py-2 bg-slate-800 rounded border border-slate-700"
                />
              </div>
            </div>
          ) : (
            <div className="space-y-2 text-slate-300">
              <div>Primary Path: {config.Storage?.PrimaryPath}</div>
              <div>Backup Path: {config.Storage?.BackupPath || 'Not set'}</div>
              <div>Port: {config.Server?.Port}</div>
            </div>
          )}
        </div>
      )}

      <div className="card p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-xl font-semibold">Scan Files</h2>
            <p className="text-slate-400">Scan storage for new media files</p>
          </div>
          <button
            onClick={startResync}
            disabled={resyncLoading}
            className="btn btn-primary flex items-center gap-2"
          >
            <RefreshCw className={`w-4 h-4 ${resyncLoading ? 'animate-spin' : ''}`} />
            {resyncLoading ? 'Scanning...' : 'Scan Now'}
          </button>
        </div>

        {resyncResult && (
          <div className={`mt-4 p-4 rounded-lg ${
            resyncResult.error ? 'bg-red-900/20 text-red-400' : 'bg-green-900/20 text-green-400'
          }`}>
            {resyncResult.error ? (
              <p>Error: {resyncResult.error}</p>
            ) : (
              <p>✓ {resyncResult.message}</p>
            )}
          </div>
        )}
      </div>

      <div className="card p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-xl font-semibold">Truncate Database</h2>
            <p className="text-slate-400">Clear all media records (files not deleted)</p>
          </div>
          <button
            onClick={truncateDatabase}
            disabled={truncateLoading}
            className="btn bg-red-600 hover:bg-red-700 flex items-center gap-2"
          >
            <Trash2 className="w-4 h-4" />
            {truncateLoading ? 'Truncating...' : 'Truncate'}
          </button>
        </div>
      </div>

      <div className="card p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-xl font-semibold">Backup</h2>
            <p className="text-slate-400">Manage incremental backups</p>
          </div>
          <button
            onClick={startBackup}
            disabled={loading}
            className="btn btn-primary flex items-center gap-2"
          >
            <HardDrive className="w-4 h-4" />
            Start Backup
          </button>
        </div>

        {backupStatus && (
          <div className="mt-4 p-4 bg-slate-900 rounded-lg">
            <pre className="text-sm text-slate-300">
              {JSON.stringify(backupStatus, null, 2)}
            </pre>
          </div>
        )}
      </div>
    </div>
  )
}

export function Settings() {
  const { platform } = useStore()

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">Settings</h1>

      <div className="card p-6">
        <h2 className="text-xl font-semibold mb-4">Platform Info</h2>
        <div className="space-y-2 text-slate-300">
          <div>Platform: {platform}</div>
          <div>User Agent: {navigator.userAgent}</div>
        </div>
      </div>
    </div>
  )
}

export default ServerManagement
