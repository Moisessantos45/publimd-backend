package routes

import (
	"publimd/config"
	"publimd/config/db"
	"publimd/internal/features/auth"
	"publimd/internal/features/collaborator"
	"publimd/internal/features/permissions"
	"publimd/internal/features/post"
	"publimd/internal/features/user"
	"publimd/internal/shared/embeddings"
	"publimd/internal/shared/middleware"
	"publimd/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

var (
	maker   *utils.PasetoMaker
	authUc  *auth.AuthUseCase
	userUc  user.UserService
	ucPost  post.PostService
	ucCols  collaborator.CollaboratorService
	ucPerms permissions.PostPermissionChecker
)

func Init() {
	rd := config.Rdb
	maker = utils.NewPasetoMaker()
	clientHTPP := embeddings.NewClient()

	aRp := auth.NewPostgresRepository(db.DB)
	authUc = auth.NewAuthUseCase(aRp, rd, maker)

	uRp := user.NewPostgresRepository(db.DB)
	userUc = user.NewUserUseCase(uRp, authUc)

	cRp := collaborator.NewPostgresRepository(db.DB)
	ucPerms = permissions.NewPermissionUseCase(permissions.NewCollaboratorRepoAdapter(cRp))

	pRp := post.NewPostgresRepository(db.DB)
	ucPost = post.NewPostUseCase(pRp, userUc, ucPerms, clientHTPP)

	ucCols = collaborator.NewCollaboratorUsecase(cRp, rd, maker, ucPerms, ucPost)
}

func GetSocketDeps() (post.PostService, permissions.PostPermissionChecker) {
	return ucPost, ucPerms
}

func authMiddleware() gin.HandlerFunc {
	return middleware.AuthMiddleware(maker, config.Rdb)
}

func preAuthMiddleware() gin.HandlerFunc {
	return middleware.PreAuthMiddleware(config.Rdb)
}
