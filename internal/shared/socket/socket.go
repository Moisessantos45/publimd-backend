package socket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"publimd/config"
	"publimd/internal/features/permissions"
	"publimd/internal/features/post"
	"publimd/internal/shared/utils"

	"github.com/gorilla/websocket"
)

var ctx = context.Background()

var slugRegex = regexp.MustCompile(`^[\p{L}0-9_-]+$`)

func isValidSlug(slug string) bool {
	if len(slug) == 0 || len(slug) > 255 {
		return false
	}
	return slugRegex.MatchString(slug)
}

type Client struct {
	conn          *websocket.Conn
	send          chan []byte
	server        *Server
	userID        int64
	authenticated bool
	currentPostID string
	sessionExp    time.Time
}

type IncomingMessage struct {
	Type          string `json:"type"`
	Kind          string `json:"kind,omitempty"`
	Token         string `json:"token,omitempty"`
	PostID        string `json:"postId,omitempty"`
	Content       string `json:"content,omitempty"`
	ClientVersion int64  `json:"clientVersion,omitempty"`
	Position      *int64 `json:"position,omitempty"`
	Length        *int64 `json:"length,omitempty"`
}

type OutgoingMessage struct {
	Type      string `json:"type"`
	Kind      string `json:"kind,omitempty"`
	Message   string `json:"message,omitempty"`
	PostID    string `json:"postId,omitempty"`
	UserID    int64  `json:"userId,omitempty"`
	Content   string `json:"content,omitempty"`
	Position  *int64 `json:"position,omitempty"`
	Length    *int64 `json:"length,omitempty"`
	Version   int64  `json:"version,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type SessionData struct {
	UserID    int64
	ExpiresAt time.Time
}

type Server struct {
	hub            *Hub
	store          PostStore
	allowedOrigins map[string]bool
	paseto         *utils.PasetoMaker
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 64 * 1024
	authTimeout    = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		allowed := map[string]bool{
			// "http://localhost:5173":     true,
		}
		return allowed[origin]
	},
}

func NewHandler(hub *Hub, allowedOrigins []string, postSvc post.PostService, checker permissions.PostPermissionChecker) *Server {
	originSet := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o != "" {
			originSet[o] = true
		}
	}
	return &Server{
		hub:            hub,
		store:          NewDBPostStore(postSvc, checker),
		allowedOrigins: originSet,
		paseto:         utils.NewPasetoMaker(),
	}
}

type PostStore interface {
	UserCanEditPost(ctx context.Context, userID int64, postID string) (bool, error)
	GetPostVersion(ctx context.Context, postID string) (int64, error)
	SavePostIfVersion(ctx context.Context, postID string, content string, expectedVersion int64, userID int64) (newVersion int64, err error)
	CanManageOrAuthor(ctx context.Context, userID int64, slug string) (bool, error)
	InvalidateCache(ctx context.Context, slug string) error
}

type DBPostStore struct {
	mu       sync.Mutex
	postSvc  post.PostService
	checker  permissions.PostPermissionChecker
	versions map[string]int64
}

func NewDBPostStore(postSvc post.PostService, checker permissions.PostPermissionChecker) *DBPostStore {
	return &DBPostStore{
		postSvc:  postSvc,
		checker:  checker,
		versions: make(map[string]int64),
	}
}

func (d *DBPostStore) resolvePostID(ctx context.Context, slug string) (uint64, error) {
	info, err := d.postSvc.GetBasicInfoBySlug(ctx, slug)
	if err != nil {
		return 0, fmt.Errorf("post no encontrado: %v", err)
	}
	return info.ID, nil
}

func (d *DBPostStore) UserCanEditPost(ctx context.Context, userID int64, slug string) (bool, error) {
	if !isValidSlug(slug) {
		return false, errors.New("slug invalido")
	}

	cacheKey := fmt.Sprintf("ws_perm:%s:%d", slug, userID)
	cachedPerm, err := config.Rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		return cachedPerm == "1", nil
	}

	postID, err := d.resolvePostID(ctx, slug)
	if err != nil {
		return false, err
	}

	isAuthor, err := d.checker.IsAuthor(ctx, postID, uint64(userID))
	if err != nil {
		return false, fmt.Errorf("error verificando autoría: %v", err)
	}

	var canEdit bool
	if isAuthor {
		canEdit = true
	} else {
		canEdit, err = d.checker.CanEditContent(ctx, uint64(userID), postID)
		if err != nil {
			return false, err
		}
	}

	val := "0"
	if canEdit {
		val = "1"
	}
	config.Rdb.Set(ctx, cacheKey, val, 10*time.Minute)

	return canEdit, nil
}

func (d *DBPostStore) CanManageOrAuthor(ctx context.Context, userID int64, slug string) (bool, error) {
	if !isValidSlug(slug) {
		return false, errors.New("slug invalido")
	}

	postID, err := d.resolvePostID(ctx, slug)
	if err != nil {
		return false, err
	}
	isAuthor, err := d.checker.IsAuthor(ctx, postID, uint64(userID))
	if err != nil {
		return false, err
	}
	if isAuthor {
		return true, nil
	}
	return d.checker.CanManagePost(ctx, uint64(userID), postID)
}

func (d *DBPostStore) InvalidateCache(ctx context.Context, slug string) error {
	if !isValidSlug(slug) {
		return errors.New("slug invalido")
	}
	pattern := fmt.Sprintf("ws_perm:%s:*", slug)
	keys, err := config.Rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return config.Rdb.Del(ctx, keys...).Err()
	}
	return nil
}

func (d *DBPostStore) GetPostVersion(ctx context.Context, slug string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	v, ok := d.versions[slug]
	if !ok {
		return 0, nil
	}
	return v, nil
}

func (d *DBPostStore) SavePostIfVersion(ctx context.Context, slug string, content string, expectedVersion int64, userID int64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	current := d.versions[slug]
	if current != expectedVersion {
		return current, errors.New("version conflict")
	}

	d.versions[slug] = current + 1
	return d.versions[slug], nil
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	select {}
}

func (h *Hub) Join(postID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[postID] == nil {
		h.rooms[postID] = make(map[*Client]bool)
	}
	h.rooms[postID][c] = true
}

func (h *Hub) LeaveAll(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for postID, clients := range h.rooms {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.rooms, postID)
		}
	}
}

func (h *Hub) BroadcastToPost(postID string, payload []byte, except *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.rooms[postID] {
		if client != except {
			select {
			case client.send <- payload:
			default:
			}
		}
	}
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	var conn *websocket.Conn
	var err error
	if len(s.allowedOrigins) > 0 {
		up := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return s.allowedOrigins[r.Header.Get("Origin")]
			},
		}
		conn, err = up.Upgrade(w, r, nil)
	} else {
		conn, err = upgrader.Upgrade(w, r, nil)
	}
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		server: s,
	}

	go client.writePump()
	client.readPump()
}

func (s *Server) HandleEditor(w http.ResponseWriter, r *http.Request) {
	s.HandleWS(w, r)
}

func (c *Client) readPump() {
	defer func() {
		c.server.hub.LeaveAll(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(authTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	authenticatedOnce := false

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg IncomingMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.writeJSON(OutgoingMessage{Type: "error", Message: "json invalido"})
			continue
		}

		if !authenticatedOnce {
			if msg.Type != "auth" {
				c.closeWithPolicy("debes autenticar primero")
				return
			}

			session, err := c.server.validateToken(ctx, msg.Token)
			if err != nil {
				c.closeWithPolicy("token invalido")
				return
			}

			c.userID = session.UserID
			c.sessionExp = session.ExpiresAt
			c.authenticated = true
			authenticatedOnce = true

			_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
			c.writeJSON(OutgoingMessage{
				Type:    "auth_ok",
				Message: "autenticado",
				UserID:  c.userID,
			})
			continue
		}

		if time.Now().After(c.sessionExp) {
			c.closeWithPolicy("sesion expirada")
			return
		}

		switch msg.Type {
		case "join_post":
			c.handleJoinPost(msg)
		case "edit":
			c.handleEdit(msg)
		case "save":
			c.handleSave(msg)
		case "refresh_auth":
			session, err := c.server.validateToken(ctx, msg.Token)
			if err != nil {
				c.closeWithPolicy("refresh invalido")
				return
			}
			c.sessionExp = session.ExpiresAt
			c.writeJSON(OutgoingMessage{Type: "refresh_ok", Message: "sesion renovada"})
		case "update_permissions":
			c.handleUpdatePermissions(msg)
		case "lock_save":
			c.handleLockSave(msg)
		case "unlock_save":
			c.handleUnlockSave(msg)
		default:
			c.writeJSON(OutgoingMessage{Type: "error", Message: "tipo de mensaje no soportado"})
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleJoinPost(msg IncomingMessage) {
	if msg.PostID == "" || !isValidSlug(msg.PostID) {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "postId invalido"})
		return
	}

	ok, err := c.server.store.UserCanEditPost(ctx, c.userID, msg.PostID)
	if err != nil {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "error verificando permisos"})
		return
	}
	if !ok {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "sin permisos para este post"})
		return
	}

	c.currentPostID = msg.PostID
	c.server.hub.Join(msg.PostID, c)

	version, _ := c.server.store.GetPostVersion(ctx, msg.PostID)

	c.writeJSON(OutgoingMessage{
		Type:    "join_ok",
		Message: "unido al post",
		PostID:  msg.PostID,
		Version: version,
	})
}

func (c *Client) handleUpdatePermissions(msg IncomingMessage) {
	if msg.PostID == "" || !isValidSlug(msg.PostID) {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "postId invalido"})
		return
	}

	ok, err := c.server.store.CanManageOrAuthor(ctx, c.userID, msg.PostID)
	if err != nil || !ok {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "sin permisos para actualizar"})
		return
	}

	err = c.server.store.InvalidateCache(ctx, msg.PostID)
	if err != nil {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "error al invalidar cache"})
		return
	}

	c.writeJSON(OutgoingMessage{Type: "update_permissions_ok", Message: "cache invalidado"})
}

func (c *Client) handleLockSave(msg IncomingMessage) {
	if c.currentPostID == "" || c.currentPostID != msg.PostID {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "no estas unido a ese post"})
		return
	}

	out := OutgoingMessage{
		Type:      "saving_in_progress",
		PostID:    msg.PostID,
		UserID:    c.userID,
		Timestamp: time.Now().Unix(),
	}
	b, _ := json.Marshal(out)
	c.server.hub.BroadcastToPost(msg.PostID, b, c)
}

func (c *Client) handleUnlockSave(msg IncomingMessage) {
	if c.currentPostID == "" || c.currentPostID != msg.PostID {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "no estas unido a ese post"})
		return
	}

	out := OutgoingMessage{
		Type:      "save_finished",
		PostID:    msg.PostID,
		UserID:    c.userID,
		Timestamp: time.Now().Unix(),
	}
	b, _ := json.Marshal(out)
	c.server.hub.BroadcastToPost(msg.PostID, b, c)
}

func (c *Client) handleEdit(msg IncomingMessage) {
	if c.currentPostID == "" || c.currentPostID != msg.PostID {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "no estas unido a ese post"})
		return
	}

	ok, err := c.server.store.UserCanEditPost(ctx, c.userID, msg.PostID)
	if err != nil || !ok {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "sin permisos"})
		return
	}

	if len(msg.Content) > 20000 {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "contenido demasiado grande"})
		return
	}

	out := OutgoingMessage{
		Type:      "post_edited",
		Kind:      msg.Kind,
		PostID:    msg.PostID,
		UserID:    c.userID,
		Content:   msg.Content,
		Version:   msg.ClientVersion,
		Timestamp: time.Now().Unix(),
		Position:  msg.Position,
		Length:    msg.Length,
	}

	b, _ := json.Marshal(out)
	c.server.hub.BroadcastToPost(msg.PostID, b, c)
}

func (c *Client) handleSave(msg IncomingMessage) {
	if c.currentPostID == "" || c.currentPostID != msg.PostID {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "no estas unido a ese post"})
		return
	}

	ok, err := c.server.store.UserCanEditPost(ctx, c.userID, msg.PostID)
	if err != nil || !ok {
		c.writeJSON(OutgoingMessage{Type: "error", Message: "sin permisos"})
		return
	}

	newVersion, err := c.server.store.SavePostIfVersion(ctx, msg.PostID, msg.Content, msg.ClientVersion, c.userID)
	if err != nil {
		c.writeJSON(OutgoingMessage{
			Type:    "conflict",
			Message: "conflicto de version",
			PostID:  msg.PostID,
			Version: newVersion,
		})
		return
	}

	c.writeJSON(OutgoingMessage{
		Type:    "save_ok",
		Message: "guardado",
		PostID:  msg.PostID,
		Version: newVersion,
	})

	out := OutgoingMessage{
		Type:      "post_saved",
		PostID:    msg.PostID,
		UserID:    c.userID,
		Content:   msg.Content,
		Version:   newVersion,
		Timestamp: time.Now().Unix(),
	}

	b, _ := json.Marshal(out)
	c.server.hub.BroadcastToPost(msg.PostID, b, c)
}

func (c *Client) writeJSON(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.send <- b:
	default:
	}
}

func (c *Client) closeWithPolicy(message string) {
	_ = c.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, message),
		time.Now().Add(writeWait),
	)
	_ = c.conn.Close()
}

func (s *Server) validateToken(ctx context.Context, token string) (*SessionData, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("token vacio")
	}
	cacheKey := fmt.Sprintf("ws_token:%s", token)
	cached, err := config.Rdb.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		parts := strings.Split(cached, ":")
		if len(parts) == 2 {
			uid, _ := strconv.ParseInt(parts[0], 10, 64)
			exp, _ := strconv.ParseInt(parts[1], 10, 64)
			return &SessionData{
				UserID:    uid,
				ExpiresAt: time.Unix(exp, 0),
			}, nil
		}
	}

	payload, err := s.paseto.VerifyToken(token)
	if err != nil {
		return nil, err
	}

	userID, err := strconv.ParseInt(payload.UserID, 10, 64)
	if err != nil {
		return nil, errors.New("user_id invalido")
	}

	dur := time.Until(payload.Exp)
	if dur > 10*time.Minute {
		dur = 10 * time.Minute
	}
	
	if dur > 0 {
		val := fmt.Sprintf("%d:%d", userID, payload.Exp.Unix())
		config.Rdb.Set(ctx, cacheKey, val, dur)
	}

	return &SessionData{
		UserID:    userID,
		ExpiresAt: payload.Exp,
	}, nil
}
