# Private Media Ecosystem - Universal Web App

**One web app that works everywhere**: macOS (server management), Android (mobile upload), and Fire TV (playback).

## 🌐 Features

### Works On All Platforms
- **macOS** - Full server management dashboard
- **Android** - Touch-optimized upload and browsing
- **Fire TV** - D-pad navigation and video playback
- **Any Browser** - Responsive design adapts automatically

### Core Functionality
✅ **Auto Server Discovery** - Finds server on local network
✅ **Drag & Drop Upload** - Easy file uploads with progress
✅ **Hash Verification** - Client-side SHA-256 before upload
✅ **Timeline View** - Chronological media grid
✅ **Video Playback** - Full-screen player with Video.js
✅ **Sync Status** - Real-time upload tracking
✅ **Responsive UI** - Adapts to screen size and input method
✅ **Zero Installation** - Just open browser, no app needed

## 🚀 Quick Start

### 1. Install Dependencies

```bash
cd webapp
npm install
```

### 2. Start Development Server

```bash
npm run dev
```

Opens at `http://localhost:3000` with hot reload.

### 3. Build for Production

```bash
npm run build
```

Outputs to `dist/` directory.

### 4. Serve Production Build

```bash
# Option 1: Built-in preview
npm run preview

# Option 2: Serve from Go server (recommended)
# Copy dist/ contents to server's static folder
cp -r dist/* ../server/static/

# Then access via server
# http://localhost:8080/
```

## 📁 Project Structure

```
webapp/
├── index.html                 # Entry HTML
├── package.json               # Dependencies
├── vite.config.js             # Vite config (dev server, proxy)
├── tailwind.config.js         # Tailwind CSS config
│
├── src/
│   ├── main.jsx               # React entry point
│   ├── App.jsx                # Main app with routing
│   ├── index.css              # Global styles + Tailwind
│   │
│   ├── store/
│   │   └── useStore.js        # Zustand global state
│   │
│   ├── api/
│   │   └── client.js          # Axios API client
│   │
│   ├── utils/
│   │   └── hash.js            # SHA-256 file hashing
│   │
│   ├── components/
│   │   └── Layout.jsx         # Navigation layout
│   │
│   └── pages/
│       ├── Login.jsx          # Server discovery
│       ├── Dashboard.jsx      # Stats overview
│       ├── Timeline.jsx       # Media grid
│       ├── Upload.jsx         # Drag-drop upload
│       ├── Player.jsx         # Video playback
│       ├── ServerManagement.jsx # Backup controls
│       └── Settings.jsx       # Device info
│
└── dist/                      # Production build (generated)
```

## 🎨 Platform Detection

The app automatically detects the platform and adjusts:

```javascript
// Detect Fire TV
if (userAgent.includes('AFTMM') || screenWidth >= 1920) {
  platform = 'tv'
  // Shows: D-pad navigation, bottom nav, large touch targets
}

// Detect Mobile
else if (/Android|iPhone|iPad/i.test(userAgent)) {
  platform = 'mobile'
  // Shows: Bottom nav, swipe gestures, optimized grid
}

// Default to Desktop
else {
  platform = 'desktop'
  // Shows: Sidebar nav, multi-column layout
}
```

## 📱 Usage Examples

### macOS (Server Management)
1. Open Safari/Chrome: `http://localhost:8080`
2. Auto-discovers local server
3. View dashboard with stats
4. Manage backups
5. Monitor upload queue

### Android (Mobile Upload)
1. Open Chrome on phone
2. Connect to `http://media-server.local:8080`
3. Tap "Discover Server"
4. Take photo or select from gallery
5. Drag-drop to upload
6. Hash verification automatic
7. Confirms before delete

### Fire TV (Playback)
1. Open Silk Browser on Fire TV
2. Navigate to `http://media-server.local:8080`
3. Use D-pad to navigate timeline
4. Select video to play
5. Full-screen playback with controls
6. Auto-returns to timeline when done

## 🔧 Configuration

### Vite Proxy (Development)

`vite.config.js` proxies API calls to backend:

```javascript
proxy: {
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true
  }
}
```

This allows development on `http://localhost:3000` while backend runs on `:8080`.

### Production Deployment

**Option 1: Serve from Go Backend**

Update Go server to serve static files:

```go
// In server/internal/api/server.go
router.Static("/", "./static")
```

Copy built files:
```bash
npm run build
cp -r dist/* ../server/static/
```

**Option 2: Separate Web Server**

Use nginx, Apache, or any static file server:

```nginx
server {
    listen 80;
    server_name media-server.local;
    
    root /path/to/webapp/dist;
    index index.html;
    
    location /api {
        proxy_pass http://localhost:8080;
    }
}
```

## 📊 State Management

Using Zustand for global state:

```javascript
const useStore = create((set, get) => ({
  serverUrl: null,
  isConnected: false,
  platform: 'desktop',
  
  // Actions
  checkConnection: async () => { /*...*/ },
  detectPlatform: () => { /*...*/ },
  registerDevice: async () => { /*...*/ }
}))
```

Persisted to localStorage:
- Server URL
- Device ID
- Platform detection

## 🎯 Key Features Implementation

### Hash Verification

```javascript
// 1. Compute hash client-side
const hash = await computeFileHash(file)

// 2. Upload with hash
await mediaApi.uploadFile(file, { local_hash: hash })

// 3. Server returns its hash
// 4. Verify match before marking synced
```

### Drag & Drop Upload

Using `react-dropzone`:

```javascript
const { getRootProps, getInputProps } = useDropzone({
  onDrop: (files) => {
    files.forEach(file => processUpload(file))
  },
  accept: {
    'image/*': ['.jpg', '.png'],
    'video/*': ['.mp4', '.mov']
  }
})
```

### Video Playback

Using Video.js with range request support:

```javascript
const player = videojs(videoRef.current, {
  controls: true,
  autoplay: true,
  fluid: true
})

player.src({
  src: `${serverUrl}/api/stream/${hash}`,
  type: 'video/mp4'
})
```

## 🎨 Styling

### Tailwind CSS

Utility-first CSS with custom components:

```css
.btn { @apply px-4 py-2 rounded-lg font-medium }
.btn-primary { @apply bg-blue-600 hover:bg-blue-700 }
.card { @apply bg-slate-800 rounded-xl shadow-lg }
```

### Dark Mode Only

Designed for dark mode with slate color palette for comfortable viewing.

### Responsive Grid

Adapts columns based on screen size:

```javascript
// Mobile: 2 columns
// Tablet: 3 columns  
// Desktop: 4-8 columns
// Fire TV: 4 columns (large targets)
```

## 🔌 API Integration

All endpoints from server documented in API client:

```javascript
mediaApi.getTimeline({ limit: 100, offset: 0 })
mediaApi.uploadFile(file, metadata, onProgress)
mediaApi.getStatus(hash)
mediaApi.verifyUpload(hash, data)
```

## 🐛 Troubleshooting

### Cannot Connect to Server

1. Check server is running: `curl http://localhost:8080/api/health`
2. Check firewall allows port 8080
3. Try manual IP: `http://192.168.1.X:8080`
4. Clear browser cache

### Upload Fails

1. Check file size < 5GB
2. Verify server has disk space
3. Check network connection
4. Look at browser console for errors

### Video Won't Play

1. Ensure Video.js loaded (check console)
2. Check server `/api/stream/:hash` works
3. Try different browser
4. Check video codec (H.264 recommended)

### Fire TV Navigation Issues

1. Ensure D-pad focus highlights visible
2. Tab through elements to test focus
3. Check `focusable` class applied
4. Use `tabIndex` for custom elements

## 📦 Dependencies

- **React 18** - UI framework
- **React Router** - Navigation
- **Zustand** - State management
- **Axios** - HTTP client
- **Tailwind CSS** - Styling
- **Video.js** - Video player
- **react-dropzone** - File uploads
- **crypto-js** - SHA-256 hashing
- **lucide-react** - Icons
- **date-fns** - Date formatting

## 🚀 Performance

- **Lazy Loading** - Images load as scrolled
- **Code Splitting** - Routes loaded on demand
- **Optimized Build** - Vite produces minimal bundles
- **HTTP/2** - Parallel resource loading
- **Service Worker** - Optional PWA support

## 📱 Progressive Web App (Optional)

Add manifest.json and service worker for installable web app:

```json
{
  "name": "Private Media",
  "short_name": "Media",
  "start_url": "/",
  "display": "standalone",
  "icons": [...]
}
```

Users can "Add to Home Screen" on mobile devices.

## 🔒 Security

- **Local Network Only** - No internet exposure
- **Client-Side Hashing** - Verify before upload
- **HTTPS Ready** - Works with TLS certificates
- **No Cookies** - State in localStorage only

## 📝 License

MIT License
