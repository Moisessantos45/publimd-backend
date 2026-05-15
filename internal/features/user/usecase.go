package user

import (
	"context"
	"fmt"
	"log"
	"publimd/internal/features/auth"
	"publimd/internal/shared/models"
)

type UserUseCase struct {
	repo   UserRepository
	ucAuth auth.AuthService
}

func NewUserUseCase(repo UserRepository, ucAuth auth.AuthService) UserService {
	return &UserUseCase{repo: repo, ucAuth: ucAuth}
}

func (uc *UserUseCase) GetDashboardMetrics(ctx context.Context, authID uint64) (*UserDashboardMetrics, error) {
	if authID == 0 {
		return nil, fmt.Errorf("el AuthID no puede ser cero")
	}

	user, err := uc.repo.GetByAuthID(ctx, authID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener el usuario por AuthID: %v", err)
	}

	return uc.repo.GetDashboardMetrics(ctx, user.ID)
}

func (uc *UserUseCase) GetByID(ctx context.Context, id uint64) (*models.User, error) {
	if id == 0 {
		return nil, fmt.Errorf("el ID no puede ser cero")
	}

	return uc.repo.GetByID(ctx, id)
}

func (uc *UserUseCase) GetByAuthID(ctx context.Context, id uint64) (*models.User, error) {
	if id == 0 {
		return nil, fmt.Errorf("el ID no puede ser cero")
	}

	return uc.repo.GetByAuthID(ctx, id)
}

func (uc *UserUseCase) GetBasicInfoByID(ctx context.Context, id uint64) (*UserBasicInfo, error) {
	if id == 0 {
		return nil, fmt.Errorf("el ID no puede ser cero")
	}

	return uc.repo.GetBasicInfoByID(ctx, id)
}

func (uc *UserUseCase) Create(ctx context.Context, user *models.User) error {
	newUser, err := NewUser(user)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return err
	}

	log.Printf("Creating user with AuthID: %d", newUser.AuthID)

	err = uc.repo.WithTransaction(func(repo *PostgresRepository) error {
		if err := uc.repo.Create(ctx, newUser); err != nil {
			return err
		}

		if err := uc.ucAuth.ChangeCompletProfile(newUser.AuthID, true); err != nil {
			return fmt.Errorf("error al actualizar el perfil completo en auth: %v", err)
		}

		return nil
	})
	log.Printf("User created successfully with ID: %d", newUser.ID)

	*user = *newUser
	return err
}

func (uc *UserUseCase) Update(ctx context.Context, id uint64, user *models.User) error {
	if id == 0 {
		return fmt.Errorf("el ID no puede ser cero")
	}

	updatedData := BuildUserUpdateData(user)

	return uc.repo.Update(ctx, id, updatedData)
}
