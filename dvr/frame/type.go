package frame

type Type byte

const (
	T_VideoI  Type = 0xFC
	T_VideoP  Type = 0xFD
	T_Audio   Type = 0xFA
	T_Picture Type = 0xFE
	T_Info    Type = 0xF9
)

var TMap = map[Type]func() Header{
	T_VideoI:  func() Header { return &HeaderVideoI{} },
	T_VideoP:  func() Header { return &HeaderVideoP{} },
	T_Audio:   func() Header { return &HeaderAudio{} },
	T_Picture: func() Header { return &HeaderPicture{} },
	T_Info:    func() Header { return &HeaderInfo{} },
}

var THMap = map[Type]string{
	T_VideoI:  "Key video frame",
	T_VideoP:  "Non-key video frame",
	T_Audio:   "Audio frame",
	T_Picture: "Picture frame",
	T_Info:    "Info frame",
}
