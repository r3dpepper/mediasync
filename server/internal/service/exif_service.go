package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	"github.com/yourusername/private-media-ecosystem/internal/db"
)

// ExifService handles EXIF metadata extraction
type ExifService struct{}

func NewExifService() *ExifService {
	return &ExifService{}
}

// ExtractMetadata extracts EXIF data from an image file
func (s *ExifService) ExtractMetadata(filePath string) (*db.ExifMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Only process image files
	if !s.isImageFile(filePath) {
		return nil, nil // Not an error, just not an image
	}

	x, err := exif.Decode(file)
	if err != nil {
		// Many images don't have EXIF data, this is not necessarily an error
		return nil, nil
	}

	metadata := &db.ExifMetadata{}

	// Extract camera information
	if cameraMake, err := x.Get(exif.Make); err == nil {
		if makeStr, err := cameraMake.StringVal(); err == nil {
			metadata.CameraMake = &makeStr
		}
	}

	if cameraModel, err := x.Get(exif.Model); err == nil {
		if modelStr, err := cameraModel.StringVal(); err == nil {
			metadata.CameraModel = &modelStr
		}
	}

	if lensModel, err := x.Get(exif.LensModel); err == nil {
		if lensStr, err := lensModel.StringVal(); err == nil {
			metadata.LensModel = &lensStr
		}
	}

	// Extract camera settings
	if fNumber, err := x.Get(exif.FNumber); err == nil {
		if num, denom, err := fNumber.Rat2(0); err == nil && denom != 0 {
			aperture := float64(num) / float64(denom)
			metadata.Aperture = &aperture
		}
	}

	if isoSpeed, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if iso, err := isoSpeed.Int(0); err == nil {
			metadata.ISO = &iso
		}
	}

	if focalLength, err := x.Get(exif.FocalLength); err == nil {
		if num, denom, err := focalLength.Rat2(0); err == nil && denom != 0 {
			focal := float64(num) / float64(denom)
			metadata.FocalLength = &focal
		}
	}

	if exposureTime, err := x.Get(exif.ExposureTime); err == nil {
		if num, denom, err := exposureTime.Rat2(0); err == nil {
			shutter := fmt.Sprintf("1/%d", denom/num)
			metadata.ShutterSpeed = &shutter
		}
	}

	if flash, err := x.Get(exif.Flash); err == nil {
		if flashVal, err := flash.Int(0); err == nil {
			flashStr := s.flashToString(flashVal)
			metadata.Flash = &flashStr
		}
	}

	// Extract GPS data
	lat, lon, err := x.LatLong()
	if err == nil {
		metadata.GPSLatitude = &lat
		metadata.GPSLongitude = &lon
	}

	if altitude, err := x.Get(exif.GPSAltitude); err == nil {
		if num, denom, err := altitude.Rat2(0); err == nil && denom != 0 {
			alt := float64(num) / float64(denom)
			metadata.GPSAltitude = &alt
		}
	}

	// GPS timestamp
	if gpsDate, err := x.Get(exif.GPSDateStamp); err == nil {
		if _, err := x.Get(exif.GPSTimeStamp); err == nil {
			if dateStr, _ := gpsDate.StringVal(); dateStr != "" {
				// Parse GPS timestamp
				if t, err := time.Parse("2006:01:02", dateStr); err == nil {
					metadata.GPSTimestamp = &t
				}
			}
		}
	}

	// Store raw EXIF as JSON for future use
	rawExif := make(map[string]interface{})
	walker := exifWalker{data: rawExif}
	x.Walk(walker)

	if rawJSON, err := json.Marshal(rawExif); err == nil {
		rawStr := string(rawJSON)
		metadata.RawExifJSON = &rawStr
	}

	return metadata, nil
}

type exifWalker struct {
	data map[string]interface{}
}

func (w exifWalker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	w.data[string(name)] = tag.String()
	return nil
}

func (s *ExifService) isImageFile(filePath string) bool {
	ext := strings.ToLower(filePath)
	return strings.HasSuffix(ext, ".jpg") ||
		strings.HasSuffix(ext, ".jpeg") ||
		strings.HasSuffix(ext, ".png") ||
		strings.HasSuffix(ext, ".tiff") ||
		strings.HasSuffix(ext, ".tif")
}

func (s *ExifService) flashToString(flash int) string {
	switch flash {
	case 0x0:
		return "No Flash"
	case 0x1:
		return "Flash Fired"
	case 0x5:
		return "Flash Fired, No Return Light"
	case 0x7:
		return "Flash Fired, Return Light Detected"
	case 0x9:
		return "Flash Fired, Compulsory"
	case 0xd:
		return "Flash Fired, Compulsory, No Return Light"
	case 0xf:
		return "Flash Fired, Compulsory, Return Light Detected"
	case 0x10:
		return "No Flash, Compulsory"
	case 0x18:
		return "No Flash, Auto"
	case 0x19:
		return "Flash Fired, Auto"
	case 0x1d:
		return "Flash Fired, Auto, No Return Light"
	case 0x1f:
		return "Flash Fired, Auto, Return Light Detected"
	case 0x20:
		return "No Flash Function"
	case 0x41:
		return "Flash Fired, Red-eye Reduction"
	case 0x45:
		return "Flash Fired, Red-eye Reduction, No Return Light"
	case 0x47:
		return "Flash Fired, Red-eye Reduction, Return Light Detected"
	case 0x49:
		return "Flash Fired, Compulsory, Red-eye Reduction"
	case 0x4d:
		return "Flash Fired, Compulsory, Red-eye Reduction, No Return Light"
	case 0x4f:
		return "Flash Fired, Compulsory, Red-eye Reduction, Return Light Detected"
	case 0x59:
		return "Flash Fired, Auto, Red-eye Reduction"
	case 0x5d:
		return "Flash Fired, Auto, Red-eye Reduction, No Return Light"
	case 0x5f:
		return "Flash Fired, Auto, Red-eye Reduction, Return Light Detected"
	default:
		return fmt.Sprintf("Unknown (%d)", flash)
	}
}

// GetTimestampTaken extracts the original timestamp from EXIF
func (s *ExifService) GetTimestampTaken(filePath string) (*time.Time, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	x, err := exif.Decode(file)
	if err != nil {
		return nil, err
	}

	// Try DateTimeOriginal first (most accurate)
	if dt, err := x.DateTime(); err == nil {
		return &dt, nil
	}

	// Fallback to file modification time
	if stat, err := os.Stat(filePath); err == nil {
		t := stat.ModTime()
		return &t, nil
	}

	return nil, fmt.Errorf("no timestamp found")
}
