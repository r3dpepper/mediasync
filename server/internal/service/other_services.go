package service

import (
	"github.com/yourusername/private-media-ecosystem/internal/db"
	"gorm.io/gorm"
)

// DeviceService handles device registration
type DeviceService struct {
	db *gorm.DB
}

func NewDeviceService(database *gorm.DB) *DeviceService {
	return &DeviceService{db: database}
}

func (s *DeviceService) Register(deviceID, deviceName, deviceType, appVersion string) (*db.Device, error) {
	// Check if device already exists
	var device db.Device
	err := s.db.Where("device_id = ?", deviceID).First(&device).Error

	if err == gorm.ErrRecordNotFound {
		// Create new device
		device = db.Device{
			DeviceID:    deviceID,
			DeviceName:  deviceName,
			DeviceType:  deviceType,
			AppVersion:  &appVersion,
			CanUpload:   true,
			CanDownload: true,
			CanDelete:   false,
			IsActive:    true,
		}
		device.FirstSeen = device.CreatedAt
		device.LastSeen = device.CreatedAt

		if err := s.db.Create(&device).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// Update existing device
		device.LastSeen = device.UpdatedAt
		device.AppVersion = &appVersion
		s.db.Save(&device)
	}

	return &device, nil
}

func (s *DeviceService) GetByDeviceID(deviceID string) (*db.Device, error) {
	var device db.Device
	err := s.db.Where("device_id = ? AND is_active = ?", deviceID, true).First(&device).Error
	return &device, err
}

func (s *DeviceService) UpdateLastSeen(deviceID uint) error {
	return s.db.Model(&db.Device{}).
		Where("id = ?", deviceID).
		Update("last_seen", gorm.Expr("CURRENT_TIMESTAMP")).Error
}

func (s *DeviceService) ListAll() ([]db.Device, error) {
	var devices []db.Device
	err := s.db.Where("is_active = ?", true).Find(&devices).Error
	return devices, err
}
