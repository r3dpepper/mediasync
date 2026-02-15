import { Link, useLocation } from 'react-router-dom'
import { useStore } from '../store/useStore'
import { Home, Image, Upload as UploadIcon, Settings, Server, Menu } from 'lucide-react'

export default function Layout({ children }) {
  const location = useLocation()
  const { platform, sidebarOpen, toggleSidebar } = useStore()

  const navItems = [
    { path: '/', icon: Home, label: 'Dashboard' },
    { path: '/timeline', icon: Image, label: 'Timeline' },
    { path: '/upload', icon: UploadIcon, label: 'Upload', hideOnTv: true },
    { path: '/server', icon: Server, label: 'Server', hideOnMobile: true },
    { path: '/settings', icon: Settings, label: 'Settings' },
  ]

  const filteredNavItems = navItems.filter(item => {
    if (platform === 'tv' && item.hideOnTv) return false
    if (platform === 'mobile' && item.hideOnMobile) return false
    return true
  })

  // Mobile/TV: Bottom navigation
  if (platform === 'mobile' || platform === 'tv') {
    return (
      <div className="min-h-screen bg-slate-900 pb-20">
        <div className="container mx-auto px-4 py-6">
          {children}
        </div>
        
        {/* Bottom Navigation */}
        <nav className="fixed bottom-0 left-0 right-0 bg-slate-800 border-t border-slate-700">
          <div className="flex justify-around py-3">
            {filteredNavItems.map(item => (
              <Link
                key={item.path}
                to={item.path}
                className={`flex flex-col items-center gap-1 px-4 py-2 rounded-lg transition-colors ${
                  platform === 'tv' ? 'focusable' : ''
                } ${
                  location.pathname === item.path
                    ? 'text-blue-400 bg-blue-900/20'
                    : 'text-slate-400 hover:text-white'
                }`}
              >
                <item.icon className="w-6 h-6" />
                <span className="text-xs">{item.label}</span>
              </Link>
            ))}
          </div>
        </nav>
      </div>
    )
  }

  // Desktop: Sidebar navigation
  return (
    <div className="min-h-screen bg-slate-900 flex">
      {/* Sidebar */}
      <aside className={`bg-slate-800 border-r border-slate-700 transition-all duration-300 ${
        sidebarOpen ? 'w-64' : 'w-20'
      }`}>
        <div className="p-4">
          <div className="flex items-center justify-between mb-8">
            {sidebarOpen && (
              <h1 className="text-xl font-bold">Private Media</h1>
            )}
            <button
              onClick={toggleSidebar}
              className="p-2 hover:bg-slate-700 rounded-lg"
            >
              <Menu className="w-6 h-6" />
            </button>
          </div>
          
          <nav className="space-y-2">
            {filteredNavItems.map(item => (
              <Link
                key={item.path}
                to={item.path}
                className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                  location.pathname === item.path
                    ? 'bg-blue-900/20 text-blue-400'
                    : 'text-slate-400 hover:bg-slate-700 hover:text-white'
                }`}
              >
                <item.icon className="w-5 h-5 flex-shrink-0" />
                {sidebarOpen && (
                  <span className="font-medium">{item.label}</span>
                )}
              </Link>
            ))}
          </nav>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">
        <div className="container mx-auto px-6 py-8">
          {children}
        </div>
      </main>
    </div>
  )
}
