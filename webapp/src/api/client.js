import axios from 'axios'
import config from '../config'

const api = axios.create({
  baseURL: config.getApiUrl() + '/api',
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Request interceptor
api.interceptors.request.use(
  (config) => {
    // Add auth token if available
    const token = localStorage.getItem('auth_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Handle unauthorized
      localStorage.removeItem('auth_token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default api

// Helper functions for common operations
export const mediaApi = {
  // Health & Stats
  getHealth: () => api.get('/health'),
  getStats: () => api.get('/stats'),
  
  // Timeline
  getTimeline: (params) => api.get('/timeline', { params }),
  
  // Upload
  uploadFile: (file, metadata, onProgress) => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('metadata', JSON.stringify(metadata))
    
    return api.post('/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (progressEvent) => {
        if (onProgress) {
          const percentCompleted = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total
          )
          onProgress(percentCompleted)
        }
      }
    })
  },
  
  // Sync
  getStatus: (hash) => api.get(`/status/${hash}`),
  verifyUpload: (hash, data) => api.post(`/verify/${hash}`, data),
  markDeletedLocal: (hash, data) => api.delete(`/local/${hash}`, { data }),
  
  // Browse
  getRoots: () => api.get('/browse/roots'),
  browsePath: (path, params) => api.get(`/browse/${path}`, { params }),
  
  // Metadata
  getMetadata: (hash) => api.get(`/metadata/${hash}`),
  
  // Search
  search: (query, params) => api.get('/search', { 
    params: { q: query, ...params } 
  }),
  
  // Backup
  getBackupStatus: () => api.get('/backup/status'),
  startBackup: (jobType) => api.post('/backup/start', { 
    job_type: jobType,
    priority: 'normal' 
  }),
  
  // Resync
  startResync: (scanPath = '/', dryRun = false) => api.post('/resync', {
    scan_path: scanPath,
    dry_run: dryRun,
    recompute_hashes: false
  }),
  
  // Maintenance
  truncateDatabase: () => api.post('/maintenance/truncate'),
  
  // Configuration
  getConfig: () => api.get('/config'),
  updateConfig: (config) => api.put('/config', config),
}
