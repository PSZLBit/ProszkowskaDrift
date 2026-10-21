package protocol

// HelloData is sent by a controller immediately after connecting.
type HelloData struct {
	DeviceToken string `json:"deviceToken"`
	AppVersion  string `json:"appVersion,omitempty"`
}

// InputData is a controller input payload. Pointers distinguish missing fields
// from a legitimate zero value.
type InputData struct {
	Steer    *float64 `json:"steer"`
	Throttle *float64 `json:"throttle"`
	TS       *int64   `json:"ts,omitempty"`
}

// PingData is the application-level controller heartbeat.
type PingData struct {
	ClientTime int64 `json:"clientTime"`
}

// LeaveData is an optional clean controller leave.
type LeaveData struct{}

// WelcomeData is the controller's assigned slot and current public state.
type WelcomeData struct {
	Slot  int    `json:"slot"`
	Label string `json:"label"`
	Color string `json:"color"`
	Phase string `json:"phase"`
}

// PhaseData is the public round state. TimeLeftMs is authoritative from the
// game; the server only caches and forwards it.
type PhaseData struct {
	Phase      string `json:"phase"`
	TimeLeftMs int64  `json:"timeLeftMs"`
	RoundID    int64  `json:"roundId"`
}

// TaskData is the player's current parking task.
type TaskData struct {
	TaskID     int64  `json:"taskId"`
	SpotLabel  string `json:"spotLabel"`
	SpotColor  string `json:"spotColor"`
	Hint       string `json:"hint"`
	DeadlineMs int64  `json:"deadlineMs"`
}

// TaskResultData is the result of one completed or skipped task.
type TaskResultData struct {
	TaskID           int64  `json:"taskId"`
	Points           int64  `json:"points"`
	TimeScore        int64  `json:"timeScore"`
	PrecisionScore   int64  `json:"precisionScore"`
	CollisionPenalty int64  `json:"collisionPenalty"`
	Rating           string `json:"rating"`
	TotalScore       int64  `json:"totalScore"`
}

// TaskResultWithSlot is the game-to-server task result.
type TaskResultWithSlot struct {
	Slot int `json:"slot"`
	TaskResultData
}

// RoundScore is one player's public round summary.
type RoundScore struct {
	Slot       int   `json:"slot"`
	Score      int64 `json:"score"`
	Tasks      int   `json:"tasks"`
	Collisions int   `json:"collisions"`
}

// RoundResultData is the public result of a round.
type RoundResultData struct {
	RoundID    int64        `json:"roundId"`
	WinnerSlot int          `json:"winnerSlot"`
	Scores     []RoundScore `json:"scores"`
}

// ErrorData is a protocol error payload.
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PongData answers a controller ping.
type PongData struct {
	ClientTime int64 `json:"clientTime"`
	ServerTime int64 `json:"serverTime"`
}

// GameHelloData is the server's acknowledgement of an authenticated game
// connection. The game proves possession of the token in the connection URL,
// so the token itself is not echoed back.
type GameHelloData struct {
	ProtocolVersion int `json:"protocolVersion"`
}

// ConfigData is sent to the game on connect and after a reset.
type ConfigData struct {
	RoundDurationMs int64 `json:"roundDurationMs"`
	TasksPerPlayer  int   `json:"tasksPerPlayer"`
	MaxPlayers      int   `json:"maxPlayers"`
}

// PlayerJoinedData tells the game that a slot became controllable.
type PlayerJoinedData struct {
	Slot  int    `json:"slot"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// PlayerLeftData tells the game that a slot stopped being controllable.
type PlayerLeftData struct {
	Slot   int    `json:"slot"`
	Reason string `json:"reason"`
}

// GameInputData is a validated input forwarded to the game.
type GameInputData struct {
	Slot     int     `json:"slot"`
	Seq      int64   `json:"seq"`
	Steer    float64 `json:"steer"`
	Throttle float64 `json:"throttle"`
}

// HostCommandData is a server-authorised host command.
type HostCommandData struct {
	Command string `json:"command"`
	Slot    *int   `json:"slot,omitempty"`
}

// HostRequestData is the game's request for a host command.
type HostRequestData struct {
	Command string `json:"command"`
	Slot    *int   `json:"slot,omitempty"`
}

// HostDeniedData tells the game why a host request was rejected.
type HostDeniedData struct {
	Command string `json:"command"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GameReadyData is the game's first application message.
type GameReadyData struct {
	GameVersion string `json:"gameVersion"`
	MapVersion  string `json:"mapVersion"`
}

// PlayerFeedbackData is the per-player feedback message. It is sent Game ->
// Server and then Server -> the owning Controller.
type PlayerFeedbackData struct {
	Slot  int       `json:"slot"`
	Task  *TaskData `json:"task,omitempty"`
	Score int64     `json:"score"`
	State string    `json:"state"`
}

// HeartbeatData is the game's one-second liveness message.
type HeartbeatData struct {
	GameTimeMs int64 `json:"gameTimeMs"`
}

// Player left reasons.
const (
	LeftDisconnect = "disconnect"
	LeftLeave      = "leave"
	LeftTimeout    = "timeout"
)
