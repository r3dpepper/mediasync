// Central configuration for the webapp
const config = {
  // API Server URL - change this to your server's IP address
  apiUrl: import.meta.env.VITE_API_URL || 'http://192.168.12.152:8080',
  
  // Auto-detect based on hostname if not set
  getApiUrl: () => {
    if (import.meta.env.VITE_API_URL) {
      return import.meta.env.VITE_API_URL
    }
    
    const hostname = window.location.hostname
    if (hostname === 'localhost' || hostname === '127.0.0.1') {
      return 'http://localhost:8080'
    }
    
    return `http://${hostname}:8080`
  }
}

export default config
