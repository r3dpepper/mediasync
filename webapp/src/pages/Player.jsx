import { useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useStore } from '../store/useStore'
import videojs from 'video.js'
import 'video.js/dist/video-js.css'

export default function Player() {
  const { hash } = useParams()
  const navigate = useNavigate()
  const videoRef = useRef(null)
  const playerRef = useRef(null)
  const { serverUrl } = useStore()

  useEffect(() => {
    if (!videoRef.current) return

    const streamUrl = `${serverUrl}/api/stream/${hash}`

    // Initialize Video.js player
    const player = videojs(videoRef.current, {
      controls: true,
      autoplay: true,
      fluid: true,
      responsive: true,
      preload: 'auto',
      html5: {
        hls: {
          enableLowInitialPlaylist: true,
          smoothQualityChange: true
        }
      }
    })

    player.src({
      src: streamUrl,
      type: 'video/mp4'
    })

    // Handle errors
    player.on('error', () => {
      console.error('Video playback error')
    })

    // Handle playback end
    player.on('ended', () => {
      navigate('/timeline')
    })

    playerRef.current = player

    return () => {
      if (playerRef.current) {
        playerRef.current.dispose()
      }
    }
  }, [hash, serverUrl, navigate])

  return (
    <div className="fixed inset-0 bg-black z-50">
      <div className="h-full flex items-center justify-center">
        <video
          ref={videoRef}
          className="video-js vjs-big-play-centered"
        />
      </div>
      
      {/* Back button */}
      <button
        onClick={() => navigate('/timeline')}
        className="absolute top-4 left-4 btn btn-secondary"
      >
        ← Back
      </button>
    </div>
  )
}
