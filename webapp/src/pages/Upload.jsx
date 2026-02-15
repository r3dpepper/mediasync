import { useState, useCallback } from 'react'
import { useDropzone } from 'react-dropzone'
import { Upload as UploadIcon, CheckCircle, XCircle, Loader } from 'lucide-react'
import { mediaApi } from '../api/client'
import { useStore } from '../store/useStore'
import { computeFileHash } from '../utils/hash'

export default function Upload() {
  const [uploads, setUploads] = useState([])
  const { deviceId, serverUrl } = useStore()

  const onDrop = useCallback((acceptedFiles) => {
    const newUploads = acceptedFiles.map(file => ({
      id: Math.random().toString(36),
      file,
      status: 'pending', // 'pending' | 'hashing' | 'uploading' | 'verifying' | 'complete' | 'error'
      progress: 0,
      hash: null,
      error: null
    }))
    
    setUploads(prev => [...prev, ...newUploads])
    
    // Start uploading immediately
    newUploads.forEach(upload => processUpload(upload))
  }, [])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'image/*': ['.jpg', '.jpeg', '.png', '.gif', '.webp'],
      'video/*': ['.mp4', '.mov', '.avi', '.mkv']
    }
  })

  const processUpload = async (upload) => {
    try {
      // 1. Compute hash
      updateUpload(upload.id, { status: 'hashing' })
      const hash = await computeFileHash(upload.file)
      updateUpload(upload.id, { hash, status: 'uploading' })
      
      // 2. Upload file
      const metadata = {
        original_filename: upload.file.name,
        timestamp_taken: new Date().toISOString(),
        destination_path: '/uploads',
        device_id: deviceId,
        local_hash: hash,
        exif_data: null
      }
      
      const response = await mediaApi.uploadFile(
        upload.file,
        metadata,
        (progress) => {
          updateUpload(upload.id, { progress })
        }
      )
      
      // 3. Verify hash
      updateUpload(upload.id, { status: 'verifying' })
      
      if (response.data.file_hash === hash) {
        updateUpload(upload.id, { 
          status: 'complete',
          progress: 100
        })
      } else {
        throw new Error('Hash mismatch - upload corrupted')
      }
      
    } catch (error) {
      updateUpload(upload.id, {
        status: 'error',
        error: error.message
      })
    }
  }

  const updateUpload = (id, updates) => {
    setUploads(prev => prev.map(u => 
      u.id === id ? { ...u, ...updates } : u
    ))
  }

  const removeUpload = (id) => {
    setUploads(prev => prev.filter(u => u.id !== id))
  }

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">Upload Media</h1>

      {/* Drop Zone */}
      <div
        {...getRootProps()}
        className={`card p-12 border-2 border-dashed transition-all cursor-pointer ${
          isDragActive 
            ? 'border-blue-500 bg-blue-900/20' 
            : 'border-slate-600 hover:border-slate-500'
        }`}
      >
        <input {...getInputProps()} />
        <div className="text-center space-y-4">
          <UploadIcon className="w-16 h-16 mx-auto text-slate-400" />
          {isDragActive ? (
            <p className="text-lg">Drop files here...</p>
          ) : (
            <>
              <p className="text-lg">Drag & drop photos or videos here</p>
              <p className="text-sm text-slate-400">or click to browse</p>
            </>
          )}
          <p className="text-xs text-slate-500">
            Supported: JPG, PNG, GIF, MP4, MOV, AVI
          </p>
        </div>
      </div>

      {/* Upload Queue */}
      {uploads.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-xl font-semibold">Upload Queue</h2>
          {uploads.map(upload => (
            <UploadItem
              key={upload.id}
              upload={upload}
              onRemove={() => removeUpload(upload.id)}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function UploadItem({ upload, onRemove }) {
  const getStatusColor = () => {
    switch (upload.status) {
      case 'complete': return 'text-green-400'
      case 'error': return 'text-red-400'
      default: return 'text-blue-400'
    }
  }

  const getStatusIcon = () => {
    switch (upload.status) {
      case 'complete':
        return <CheckCircle className="w-5 h-5 text-green-400" />
      case 'error':
        return <XCircle className="w-5 h-5 text-red-400" />
      default:
        return <Loader className="w-5 h-5 text-blue-400 animate-spin" />
    }
  }

  const getStatusText = () => {
    switch (upload.status) {
      case 'pending': return 'Queued...'
      case 'hashing': return 'Computing hash...'
      case 'uploading': return `Uploading... ${upload.progress}%`
      case 'verifying': return 'Verifying...'
      case 'complete': return 'Complete ✓'
      case 'error': return `Error: ${upload.error}`
      default: return upload.status
    }
  }

  return (
    <div className="card p-4">
      <div className="flex items-center gap-4">
        <div className="flex-shrink-0">
          {getStatusIcon()}
        </div>
        
        <div className="flex-1 min-w-0">
          <div className="font-medium truncate">{upload.file.name}</div>
          <div className={`text-sm ${getStatusColor()}`}>
            {getStatusText()}
          </div>
          
          {upload.status === 'uploading' && (
            <div className="mt-2 w-full bg-slate-700 rounded-full h-2">
              <div
                className="bg-blue-500 h-2 rounded-full transition-all"
                style={{ width: `${upload.progress}%` }}
              />
            </div>
          )}
        </div>
        
        <div className="text-sm text-slate-400">
          {(upload.file.size / 1024 / 1024).toFixed(1)} MB
        </div>
        
        {(upload.status === 'complete' || upload.status === 'error') && (
          <button
            onClick={onRemove}
            className="text-slate-400 hover:text-white"
          >
            ×
          </button>
        )}
      </div>
    </div>
  )
}
