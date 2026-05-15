package db

import "publimd/internal/shared/models"

func SeedDatabase() error {
	var count int64
	if err := DB.Model(&models.StatePost{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		if err := DB.Create(&models.StatePosts).Error; err != nil {
			return err
		}
	}

	if err := DB.Model(&models.Permission{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		if err := DB.Create(&models.Permissions).Error; err != nil {
			return err
		}
	}

	return nil
}
