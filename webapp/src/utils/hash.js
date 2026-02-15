import CryptoJS from 'crypto-js'

export async function computeFileHash(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    
    reader.onload = (e) => {
      try {
        const wordArray = CryptoJS.lib.WordArray.create(e.target.result)
        const hash = CryptoJS.SHA256(wordArray).toString()
        resolve(hash)
      } catch (error) {
        reject(error)
      }
    }
    
    reader.onerror = (error) => reject(error)
    reader.readAsArrayBuffer(file)
  })
}

export function verifyHash(hash1, hash2) {
  return hash1.toLowerCase() === hash2.toLowerCase()
}
