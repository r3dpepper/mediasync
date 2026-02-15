import { useState, useEffect, useRef } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { mediaApi } from '../api/client'
import { useStore } from '../store/useStore'
import { X, ChevronLeft, ChevronRight } from 'lucide-react'

export default function Viewer() {
  const { hash } = useParams()
  const navigate = useNavigate()
  const { serverUrl } = useStore()
  const [items, setItems] = useState([])
  const [currentIndex, setCurrentIndex] = useState(0)
  const [loading, setLoading] = useState(true)
  const videoRef = useRef(null)

  useEffect(() => {
    loadTimeline()
  }, [])

  useEffect(() => {
    if (items.length > 0 && hash) {
      const index = items.findIndex(item => item.file_hash === hash)
      if (index !== -1) setCurrentIndex(index)
    }
  }, [items, hash])

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'ArrowRight') goNext()
      if (e.key === 'ArrowLeft') goPrev()
      if (e.key === 'Escape') navigate('/timeline')
      
      // Handle play/pause for Fire TV remote and spacebar
      if ((e.key === ' ' || e.key === 'MediaPlayPause' || e.keyCode === 179) && videoRef.current) {
        e.preventDefault()
        if (videoRef.current.paused) {
          videoRef.current.play()
        } else {
          videoRef.current.pause()
        }
      }
      
      // Handle separate play/pause buttons
      if ((e.key === 'MediaPlay' || e.keyCode === 415) && videoRef.current) {
        e.preventDefault()
        videoRef.current.play()
      }
      if ((e.key === 'MediaPause' || e.keyCode === 19) && videoRef.current) {
        e.preventDefault()
        videoRef.current.pause()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [currentIndex, items])

  const loadTimeline = async () => {
    try {
      const response = await mediaApi.getTimeline({ limit: 1000, offset: 0, sort: 'desc' })
      setItems(response.data.items)
    } catch (err) {
      console.error('Failed to load timeline:', err)
    } finally {
      setLoading(false)
    }
  }

  const goNext = () => {
    if (currentIndex < items.length - 1) {
      setCurrentIndex(currentIndex + 1)
    }
  }

  const goPrev = () => {
    if (currentIndex > 0) {
      setCurrentIndex(currentIndex - 1)
    }
  }

  if (loading || items.length === 0) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>
  }

  const currentItem = items[currentIndex]

  return (
    <div className="fixed inset-0 bg-black z-50 flex items-center justify-center">
      <button
        onClick={() => navigate('/timeline')}
        className="absolute top-4 right-4 z-10 p-2 bg-black/50 rounded-full hover:bg-black/70"
      >
        <X className="w-6 h-6" />
      </button>

      <button
        onClick={goPrev}
        disabled={currentIndex === 0}
        className="absolute left-4 z-10 p-2 bg-black/50 rounded-full hover:bg-black/70 disabled:opacity-30"
      >
        <ChevronLeft className="w-8 h-8" />
      </button>

      <button
        onClick={goNext}
        disabled={currentIndex === items.length - 1}
        className="absolute right-4 z-10 p-2 bg-black/50 rounded-full hover:bg-black/70 disabled:opacity-30"
      >
        <ChevronRight className="w-8 h-8" />
      </button>

      <div className="w-full h-full flex items-center justify-center">
        {currentItem.media_type === 'video' ? (
          <video
            ref={videoRef}
            key={currentItem.file_hash}
            src={`${serverUrl}/api/stream/${currentItem.file_hash}?transcode=true`}
            className="w-full h-full"
            style={{ objectFit: 'contain' }}
            controls
            playsInline
            preload="auto"
          />
        ) : (
          <img
            key={currentItem.file_hash}
            src={`${serverUrl}/api/stream/${currentItem.file_hash}`}
            alt={currentItem.original_filename}
            className="w-full h-full object-contain"
          />
        )}
      </div>

      <div className="absolute bottom-4 left-1/2 transform -translate-x-1/2 text-white bg-black/50 px-4 py-2 rounded">
        {currentIndex + 1} / {items.length} - {currentItem.original_filename}
      </div>
    </div>
  )
}
