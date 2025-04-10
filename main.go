package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/serialize/json"
	"github.com/lonng/nano/session"

	encodingjson "encoding/json"
)

type (
	Room struct {
		group *nano.Group
	}

	// RoomManager represents a component that contains a bundle of room
	RoomManager struct {
		component.Base
		timer *scheduler.Timer
		rooms map[int]*Room
	}

	// UserMessage represents a message that user sent
	UserMessage struct {
		Name      string `json:"name"`
		Content   string `json:"content"`
		Timestamp string `json:"timestamp"`
	}

	InitClientMessage struct {
		AllMessage []UserMessage `json:"allMessage"`
	}

	// NewUser message will be received when new user join room
	NewUser struct {
		Content string `json:"content"`
	}

	// AllMembers contains all members uid
	AllMembers struct {
		Members []int64 `json:"members"`
	}

	AllMembersNickname struct {
		MembersNickName []string `json:"membersnickname"`
	}

	// JoinResponse represents the result of joining room
	JoinResponse struct {
		Code   int    `json:"code"`
		Result string `json:"result"`
	}

	Stats struct {
		component.Base
		timer         *scheduler.Timer
		outboundBytes int
		inboundBytes  int
	}

	JoinMessage struct {
		Nickname string `json:"nickname"`
	}
)

func (stats *Stats) outbound(s *session.Session, msg *pipeline.Message) error {
	stats.outboundBytes += len(msg.Data)
	return nil
}

func (stats *Stats) inbound(s *session.Session, msg *pipeline.Message) error {
	stats.inboundBytes += len(msg.Data)
	return nil
}

func (stats *Stats) AfterInit() {
	stats.timer = scheduler.NewTimer(time.Minute, func() {
		println("OutboundBytes", stats.outboundBytes)
		println("InboundBytes", stats.outboundBytes)
	})
}

func (st *Stats) Nil(s *session.Session, msg []byte) error {
	return nil
}

const (
	testRoomID = 1
	roomIDKey  = "ROOM_ID"
)

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: map[int]*Room{},
	}
}

// AfterInit component lifetime callback
func (mgr *RoomManager) AfterInit() {
	// 监听会话关闭事件
	session.Lifetime.OnClosed(func(s *session.Session) {
		if !s.HasKey(roomIDKey) {
			return
		}

		// 获取当前房间
		room := s.Value(roomIDKey).(*Room)

		// 广播一条消息通知其他成员用户已经离开
		leaveMessage := fmt.Sprintf("%s 已经离开聊天室", s.String("nickname")) // 使用s.ID()作为用户标识
		room.group.Broadcast("onUserLeft", &NewUser{Content: leaveMessage})

		// 从房间中移除该会话
		room.group.Leave(s)
		delete(idNickNameMap, int(s.ID()))
		var nicknamearray []string
		for _, value := range idNickNameMap {
			nicknamearray = append(nicknamearray, value)
		}
		room.group.Broadcast("onUpdateList", &AllMembersNickname{MembersNickName: nicknamearray})
	})

	// 定时打印房间人数
	mgr.timer = scheduler.NewTimer(time.Minute, func() {
		for roomId, room := range mgr.rooms {
			println(fmt.Sprintf("UserCount: RoomID=%d, Time=%s, Count=%d",
				roomId, time.Now().String(), room.group.Count()))
		}
	})
}

var idNickNameMap = make(map[int]string)

// Join room
func (mgr *RoomManager) Join(s *session.Session, msg []byte) error {
	var joinMsg JoinMessage
	if err := encodingjson.Unmarshal(msg, &joinMsg); err != nil {
		return fmt.Errorf("failed to parse join message: %v", err)
	}

	// NOTE: join test room only in demo
	room, found := mgr.rooms[testRoomID]
	if !found {
		room = &Room{
			group: nano.NewGroup(fmt.Sprintf("room-%d", testRoomID)),
		}
		mgr.rooms[testRoomID] = room
	}

	fakeUID := s.ID() //just use s.ID as uid !!!
	s.Bind(fakeUID)   // binding session uids.Set(roomIDKey, room)
	s.Set(roomIDKey, room)
	s.Set("nickname", joinMsg.Nickname)
	idNickNameMap[int(fakeUID)] = joinMsg.Nickname

	room.group.Broadcast("onNewUser", &NewUser{Content: fmt.Sprintf("%s 加入聊天室", joinMsg.Nickname)})
	// new user join group
	room.group.Add(s) // add session to group
	// room.group.Broadcast("onUpdateList", &AllMembers{Members: room.group.Members()})
	var nicknamearray []string
	for _, value := range idNickNameMap {
		nicknamearray = append(nicknamearray, value)
	}
	room.group.Broadcast("onUpdateList", &AllMembersNickname{MembersNickName: nicknamearray})
	client_massage, _ := GetAllMessages()
	room.group.Broadcast("initMessage", &InitClientMessage{AllMessage: client_massage})
	return s.Response(&JoinResponse{Result: "success-2"})
}

// Message sync last message to all members
func (mgr *RoomManager) Message(s *session.Session, msg *UserMessage) error {
	if !s.HasKey(roomIDKey) {
		return fmt.Errorf("not join room yet")
	}
	room := s.Value(roomIDKey).(*Room)
	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-1-2 15:04:05")
	result := formattedTime
	msg.Timestamp = result
	InsertMessage(msg)
	return room.group.Broadcast("onMessage", msg)
}

func InsertMessage(msg *UserMessage) {
	// 打开 MySQL 数据库连接
	db, err := sql.Open("mysql", "blog:hgj1097229155@tcp(106.14.75.103:3306)/blog?charset=utf8")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 获取当前时间，如果没有提供 Timestamp 则使用当前时间
	if msg.Timestamp == "" {
		msg.Timestamp = time.Now().Format("2006-01-02 15:04:05")
	}

	// 插入消息到数据库
	query := `INSERT INTO client_message (name, content, timestamp) VALUES (?, ?, ?)`
	_, err = db.Exec(query, msg.Name, msg.Content, msg.Timestamp)
	if err != nil {
		log.Fatal(err)
	}

	// 输出成功信息
	fmt.Println("Message inserted successfully")
}

func GetAllMessages() ([]UserMessage, error) {
	// 打开数据库连接
	db, err := sql.Open("mysql", "blog:hgj1097229155@tcp(106.14.75.103:3306)/blog?charset=utf8")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// 查询所有消息
	rows, err := db.Query("SELECT name, content, timestamp FROM client_message")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 定义一个切片来存储所有的 UserMessage
	var messages []UserMessage

	// 遍历查询结果
	for rows.Next() {
		var msg UserMessage
		if err := rows.Scan(&msg.Name, &msg.Content, &msg.Timestamp); err != nil {
			return nil, err
		}
		fmt.Println(msg)
		// 将每条消息添加到切片中
		messages = append(messages, msg)
	}

	// 检查查询过程中是否有错误
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 返回所有的消息
	return messages, nil
}

func main() {
	components := &component.Components{}
	components.Register(
		NewRoomManager(),
		component.WithName("room"), // rewrite component and handler name
		component.WithNameFunc(strings.ToLower),
	)

	// traffic stats
	pip := pipeline.New()
	var stats = &Stats{}
	pip.Outbound().PushBack(stats.outbound)
	pip.Inbound().PushBack(stats.inbound)

	components.Register(stats, component.WithName("stats"))

	log.SetFlags(log.LstdFlags | log.Llongfile)
	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

	nano.Listen(":3250",
		nano.WithIsWebsocket(true),
		nano.WithPipeline(pip),
		nano.WithCheckOriginFunc(func(_ *http.Request) bool { return true }),
		nano.WithWSPath("/nano"),
		nano.WithDebugMode(),
		nano.WithSerializer(json.NewSerializer()), // override default serializer
		nano.WithComponents(components),
	)
}
