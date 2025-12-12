package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/yoockh/yoospeak/internal/services"
	"github.com/yoockh/yoospeak/internal/utils"
)

type WSHandler struct {
	sessions services.SessionService
	buffers  services.BufferService
	redis    *redis.Client
	upgrader websocket.Upgrader
}

func NewWSHandler(sessions services.SessionService, buffers services.BufferService, rdb *redis.Client) *WSHandler {
	return &WSHandler{
		sessions: sessions,
		buffers:  buffers,
		redis:    rdb,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // TODO: restrict origin in prod
		},
	}
}

type wsClientMsg struct {
	Type        string `json:"type"`
	SessionID   string `json:"session_id"`
	ChunkIndex  int64  `json:"chunk_index"`
	AudioBase64 string `json:"audio_base64"`
	AudioURL    string `json:"audio_url"`
	IsFinal     bool   `json:"is_final"`

	// pause/resume/end_session -> no fields
}

type wsServerMsg struct {
	Type string `json:"type"`

	// generic payload pass-through (workers publish JSON to Redis; WS forwards)
	// keep this minimal at handler layer
}

type wsConn struct {
	c  *websocket.Conn
	mu sync.Mutex
}

func (w *wsConn) writeText(b []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.c.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return w.c.WriteMessage(websocket.TextMessage, b)
}

func (h *WSHandler) SessionWS(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		writeError(c, utils.E(utils.CodeInvalidArgument, "WSHandler.SessionWS", "missing session_id", nil))
		return
	}

	// authorize session ownership
	sess, err := h.sessions.Get(c.Request.Context(), sessionID)
	if err != nil {
		writeError(c, err)
		return
	}
	if sess.UserID != userID {
		writeError(c, utils.E(utils.CodeForbidden, "WSHandler.SessionWS", "forbidden", nil))
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// upgrade already wrote response in most cases
		return
	}
	defer conn.Close()

	wc := &wsConn{c: conn}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Subscribe Redis -> WS
	respCh := "session:" + sessionID + ":response"
	statusCh := "session:" + sessionID + ":status"

	pubsub := h.redis.Subscribe(ctx, respCh, statusCh)
	defer pubsub.Close()

	// reader: WS -> Redis Stream (+ Mongo buffer insert)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, data, rerr := conn.ReadMessage()
			if rerr != nil {
				return
			}

			var msg wsClientMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				_ = wc.writeText([]byte(`{"type":"error","code":"INVALID_ARGUMENT","message":"invalid json"}`))
				continue
			}

			switch msg.Type {
			case "audio_chunk":
				// validate minimal
				if msg.ChunkIndex <= 0 {
					_ = wc.writeText([]byte(`{"type":"error","code":"INVALID_ARGUMENT","message":"chunk_index must be > 0"}`))
					continue
				}

				// accept either audio_base64 or audio_url
				var audioBase64Ptr *string
				var audioURLPtr *string
				if msg.AudioBase64 != "" {
					audioBase64Ptr = &msg.AudioBase64
				}
				if msg.AudioURL != "" {
					audioURLPtr = &msg.AudioURL
				}
				if audioBase64Ptr == nil && audioURLPtr == nil {
					_ = wc.writeText([]byte(`{"type":"error","code":"INVALID_ARGUMENT","message":"audio_base64 or audio_url required"}`))
					continue
				}

				// insert Mongo realtime_buffer (pending)
				_, err := h.buffers.InsertAudioChunk(ctx, sessionID, msg.ChunkIndex, audioURLPtr, audioBase64Ptr)
				if err != nil {
					_ = wc.writeText([]byte(`{"type":"error","code":"INTERNAL","message":"failed to insert buffer"}`))
					continue
				}

				// push to Redis Stream: audio:stream
				fields := map[string]any{
					"session_id":  sessionID,
					"chunk_index": strconv.FormatInt(msg.ChunkIndex, 10),
					"is_final":    strconv.FormatBool(msg.IsFinal),
					"ts_unix":     strconv.FormatInt(time.Now().UTC().Unix(), 10),
				}
				if audioBase64Ptr != nil {
					fields["audio_base64"] = *audioBase64Ptr
				}
				if audioURLPtr != nil {
					fields["audio_url"] = *audioURLPtr
				}

				if err := h.redis.XAdd(ctx, &redis.XAddArgs{
					Stream: "audio:stream",
					Values: fields,
				}).Err(); err != nil {
					_ = wc.writeText([]byte(`{"type":"error","code":"UNAVAILABLE","message":"failed to enqueue audio"}`))
					continue
				}

				// optional: immediate status ack
				_ = h.redis.Publish(ctx, statusCh, `{"type":"status","status":"processing","message":"audio chunk queued","chunk_index":`+strconv.FormatInt(msg.ChunkIndex, 10)+`}`).Err()

			case "pause":
				_ = h.redis.Publish(ctx, statusCh, `{"type":"status","status":"paused","message":"paused"}`).Err()

			case "resume":
				_ = h.redis.Publish(ctx, statusCh, `{"type":"status","status":"ready","message":"resumed"}`).Err()

			case "end_session":
				_, _ = h.sessions.End(ctx, sessionID)
				_ = h.redis.Publish(ctx, statusCh, `{"type":"status","status":"ended","message":"session ended"}`).Err()
				return

			default:
				_ = wc.writeText([]byte(`{"type":"error","code":"INVALID_ARGUMENT","message":"unknown message type"}`))
			}
		}
	}()

	// writer: Redis Pub/Sub -> WS
	for {
		select {
		case <-readDone:
			return
		case <-ctx.Done():
			return
		default:
			m, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				return
			}
			// forward as-is (payload expected JSON string)
			if werr := wc.writeText([]byte(m.Payload)); werr != nil {
				return
			}
		}
	}
}
