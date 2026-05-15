package models

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

type Auth struct {
	ID               uint64 `json:"id" gorm:"primaryKey"`
	Email            string `json:"email" gorm:"unique;not null"`
	PasswordHash     string `json:"-" gorm:"not null"`
	TwoFactorSecret  string `json:"-" gorm:"type:varchar(26)"`
	TwoFactorEnabled bool   `json:"two_factor_enabled" gorm:"default:false"`
	FirstSession     bool   `json:"first_session" gorm:"default:true"`
	FullProfile      bool   `json:"full_profile" gorm:"default:false"`
	EmailConfirmed   bool   `json:"email_confirmed" gorm:"default:false"`

	Token string `json:"token" gorm:"-"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type User struct {
	ID       uint64 `json:"id" gorm:"primaryKey"`
	Name     string `json:"name" gorm:"not null;type:varchar(100)"`
	LastName string `json:"last_name" gorm:"not null;type:varchar(100)"`
	Bio      string `json:"bio" gorm:"type:text"`
	Avatar   string `json:"avatar" gorm:"type:text"`
	AuthID   uint64 `json:"auth_id" gorm:"unique;not null"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Auth Auth `json:"-" gorm:"foreignKey:AuthID"`
}

type UserBasicInfo struct {
	ID     uint64 `json:"id" gorm:"column:id"`
	AuthID uint64 `json:"auth_id" gorm:"column:auth_id"`
}

type PostInfoBasic struct {
	ID   uint64 `json:"id" gorm:"column:id"`
	Slug string `json:"slug" gorm:"column:slug"`
}

type StatePost struct {
	ID   uint64 `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"not null;type:varchar(50)"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Post struct {
	ID              uint64           `json:"id" gorm:"primaryKey"`
	Slug            string           `json:"slug" gorm:"unique;not null;type:varchar(150)"`
	Title           string           `json:"title" gorm:"not null;type:varchar(200)"`
	Content         string           `json:"content" gorm:"type:text"`
	AuthorID        uint64           `json:"author_id" gorm:"not null"`
	Tags            JSONStringArray  `json:"tags" gorm:"type:jsonb;default:'[]'"`
	Category        string           `json:"category" gorm:"type:varchar(100)"`
	StateID         uint64           `json:"state_id" gorm:"not null"`
	Embedding       *pgvector.Vector `json:"-" gorm:"column:embedding;type:vector(768)"`
	SearchVector    string           `json:"-" gorm:"->;type:tsvector"`
	FuzzyShort      string           `json:"-" gorm:"->;type:text"`
	ContentClean    string           `json:"-" gorm:"column:content_clean;type:text"`
	IsCollaborative bool             `json:"is_collaborative" gorm:"-"`
	PermissionID    uint64           `json:"permission_id" gorm:"-"`
	CreatedAt       time.Time        `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time        `json:"updated_at" gorm:"autoUpdateTime"`

	Author User      `json:"-" gorm:"foreignKey:AuthorID"`
	State  StatePost `json:"-" gorm:"foreignKey:StateID"`
}

type Permission struct {
	ID   uint64 `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"not null;type:varchar(50)"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Collaborator struct {
	ID           uint64 `json:"id" gorm:"primaryKey"`
	UserID       uint64 `json:"user_id" gorm:"not null"`
	PostID       uint64 `json:"post_id" gorm:"not null"`
	PermissionID uint64 `json:"permission_id" gorm:"not null"`
	Confirmed    bool   `json:"confirmed" gorm:"default:false"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	User       User       `json:"-" gorm:"foreignKey:UserID"`
	Post       Post       `json:"-" gorm:"foreignKey:PostID"`
	Permission Permission `json:"-" gorm:"foreignKey:PermissionID"`
}

type Like struct {
	ID     uint64 `json:"id" gorm:"primaryKey"`
	UserID uint64 `json:"user_id" gorm:"not null"`
	PostID uint64 `json:"post_id" gorm:"not null"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	User User `json:"user" gorm:"foreignKey:UserID"`
	Post Post `json:"post" gorm:"foreignKey:PostID"`
}

type Comment struct {
	ID       uint64 `json:"id" gorm:"primaryKey"`
	Content  string `json:"content" gorm:"type:text"`
	AuthorID uint64 `json:"author_id" gorm:"not null"`
	PostID   uint64 `json:"post_id" gorm:"not null"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Author User `json:"author" gorm:"foreignKey:AuthorID"`
	Post   Post `json:"post" gorm:"foreignKey:PostID"`
}

var Models = []any{
	&Auth{},
	&User{},
	&Post{},
	&StatePost{},
	&Permission{},
	&Collaborator{},
	&Like{},
	&Comment{},
}

// agregar esytado que sea para cuando el usuario crea el post y solo lo pueden ver las personas que tenga el link asi como youtube que puedes colocar un video como privado y solo lo pueden ver las personas que tengan el link, esto es para que el usuario pueda compartir su post con otras personas sin necesidad de publicarlo completamente, esto se puede llamar "unlisted" o "privado con enlace" o algo similar
var StatePosts = []StatePost{
	{Name: "draft"},
	{Name: "published"},
	{Name: "archived"},
	{Name: "private"},
	{Name: "unlisted"},
}

var Permissions = []Permission{
	{ID: 1, Name: "read"},
	{ID: 2, Name: "write"},
	{ID: 3, Name: "manage"},
	{ID: 4, Name: "admin"},
}
