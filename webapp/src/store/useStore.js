import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import api from '../api/client'
import config from '../config'

export const useStore = create(
  persist(
    (set, get) => ({
      // Server connection
      serverUrl: config.getApiUrl(),
      isConnected: false,
      serverInfo: null,
      
      // Platform detection
      platform: 'desktop', // 'desktop' | 'mobile' | 'tv'
      
      // User/Device info
      deviceId: null,
      deviceName: null,
      
      // Media state
      currentMedia: null,
      uploadQueue: [],
      syncStatus: {},
      
      // UI state
      sidebarOpen: true,
      
      // Actions
      setServerUrl: (url) => set({ serverUrl: url }),
      
      checkConnection: async () => {
        try {
          const response = await api.get('/health')
          set({ 
            isConnected: true, 
            serverInfo: response.data 
          })
          return true
        } catch (error) {
          set({ isConnected: false })
          return false
        }
      },
      
      detectPlatform: () => {
        const ua = navigator.userAgent
        
        // Detect Fire TV
        if (ua.includes('AFTMM') || ua.includes('AFTB') || 
            ua.includes('AFT') || window.innerWidth >= 1920) {
          set({ platform: 'tv' })
          return
        }
        
        // Detect mobile
        if (/Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(ua)) {
          set({ platform: 'mobile' })
          return
        }
        
        // Default to desktop
        set({ platform: 'desktop' })
      },
      
      registerDevice: async (deviceName, deviceType) => {
        const deviceId = get().deviceId || generateDeviceId()
        
        try {
          const response = await api.post('/devices/register', {
            device_id: deviceId,
            device_name: deviceName,
            device_type: deviceType || get().platform,
            app_version: '1.0.0'
          })
          
          set({ 
            deviceId,
            deviceName,
            isConnected: true 
          })
          
          return response.data
        } catch (error) {
          console.error('Device registration failed:', error)
          throw error
        }
      },
      
      addToUploadQueue: (files) => {
        const queue = get().uploadQueue
        set({ uploadQueue: [...queue, ...files] })
      },
      
      removeFromUploadQueue: (fileId) => {
        const queue = get().uploadQueue.filter(f => f.id !== fileId)
        set({ uploadQueue: queue })
      },
      
      updateSyncStatus: (hash, status) => {
        const syncStatus = get().syncStatus
        set({ 
          syncStatus: { 
            ...syncStatus, 
            [hash]: status 
          } 
        })
      },
      
      toggleSidebar: () => set({ sidebarOpen: !get().sidebarOpen }),
      
      setCurrentMedia: (media) => set({ currentMedia: media }),
    }),
    {
      name: 'media-storage',
      partialize: (state) => ({
        serverUrl: state.serverUrl,
        deviceId: state.deviceId,
        deviceName: state.deviceName,
        platform: state.platform,
      })
    }
  )
)

function generateDeviceId() {
  return 'web-' + Math.random().toString(36).substring(2, 15) + 
         Math.random().toString(36).substring(2, 15)
}
