import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { mediaApi } from '../api/client'
import { useStore } from '../store/useStore'
import { Play, Image, Clock, MapPin } from 'lucide-react'
import { format } from 'date-fns'

export default function Timeline() {
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [filter, setFilter] = useState('all') // 'all' | 'image' | 'video'
  
  const navigate = useNavigate()
  const { serverUrl, platform } = useStore()

  useEffect(() => {
    loadTimeline()
  }, [filter])

  const loadTimeline = async () => {
    setLoading(true)
    setError(null)
    
    try {
      const params = {
        limit: 100,
        offset: 0,
        sort: 'desc'
      }
      
      if (filter !== 'all') {
        params.media_type = filter
      }
      
      const response = await mediaApi.getTimeline(params)
      setItems(response.data.items)
    } catch (err) {
      setError('Failed to load timeline: ' + err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleMediaClick = (item) => {
    navigate(`/viewer/${item.file_hash}`)
  }

  const getGridClass = () => {
    if (platform === 'tv') return 'grid-cols-4'
    if (platform === 'mobile') return 'grid-cols-2'
    return 'grid-cols-4 md:grid-cols-6 lg:grid-cols-8'
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="spinner"></div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-center py-12">
        <p className="text-red-400">{error}</p>
        <button onClick={loadTimeline} className="btn btn-primary mt-4">
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Timeline</h1>
        
        {/* Filters */}
        <div className="flex gap-2">
          <button
            onClick={() => setFilter('all')}
            className={`btn ${filter === 'all' ? 'btn-primary' : 'btn-secondary'}`}
          >
            All
          </button>
          <button
            onClick={() => setFilter('image')}
            className={`btn ${filter === 'image' ? 'btn-primary' : 'btn-secondary'}`}
          >
            <Image className="w-4 h-4" />
          </button>
          <button
            onClick={() => setFilter('video')}
            className={`btn ${filter === 'video' ? 'btn-primary' : 'btn-secondary'}`}
          >
            <Play className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="card p-4">
          <div className="text-2xl font-bold">{items.length}</div>
          <div className="text-sm text-slate-400">Media Files</div>
        </div>
      </div>

      {/* Media Grid */}
      {items.length === 0 ? (
        <div className="text-center py-12 text-slate-400">
          <p>No media files found</p>
        </div>
      ) : (
        <div className={`grid ${getGridClass()} gap-4`}>
          {items.map((item) => (
            <MediaThumbnail
              key={item.file_hash}
              item={item}
              serverUrl={serverUrl}
              onClick={() => handleMediaClick(item)}
              platform={platform}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function MediaThumbnail({ item, serverUrl, onClick, platform }) {
  const [imageError, setImageError] = useState(false)
  
  const thumbnailUrl = `${serverUrl}${item.thumbnail_url}?size=medium`
  
  return (
    <div
      onClick={onClick}
      className={`relative group cursor-pointer overflow-hidden rounded-lg bg-slate-800 aspect-square ${
        platform === 'tv' ? 'focusable' : ''
      }`}
      tabIndex={platform === 'tv' ? 0 : undefined}
    >
      {!imageError ? (
        <img
          src={thumbnailUrl}
          alt={item.original_filename}
          className="w-full h-full object-cover transition-transform group-hover:scale-105"
          onError={() => setImageError(true)}
          loading="lazy"
        />
      ) : (
        <div className="w-full h-full flex items-center justify-center">
          <Image className="w-12 h-12 text-slate-600" />
        </div>
      )}
      
      {/* Video indicator */}
      {item.media_type === 'video' && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity">
          <Play className="w-12 h-12 text-white" />
        </div>
      )}
      
      {/* Metadata overlay */}
      <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/80 to-transparent p-2 opacity-0 group-hover:opacity-100 transition-opacity">
        <div className="text-xs space-y-1">
          <div className="flex items-center gap-1 text-white">
            <Clock className="w-3 h-3" />
            {format(new Date(item.timestamp_taken), 'MMM d, yyyy')}
          </div>
          {item.location && (
            <div className="flex items-center gap-1 text-white">
              <MapPin className="w-3 h-3" />
              {item.location.name || 'Location'}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
