// (c) Bernhard Tittelbach, 2016

package r3events

type YamahaIRCmd struct {
	Cmd string
	Ts  int64
}

type SendRF433Code struct {
	Code [3]byte
	Ts   int64
}

type SetRF433Delay struct {
	Location string
	DelayMs  int64
}

type RestartPipeLEDs struct {
}

type SetPipeLEDsPattern struct {
	Pattern string `json:"pattern"`
	Arg     int64  `json:"arg,omitempty"`
	Arg1    int64  `json:"arg1,omitempty"`
}

type LightCtrlActionOnName struct {
	Name   string
	Action string
}

type FancyLight struct {
	R    uint16 `json:"r,omitempty"`
	G    uint16 `json:"g,omitempty"`
	B    uint16 `json:"b,omitempty"`
	CW   uint16 `json:"cw,omitempty"`
	WW   uint16 `json:"ww,omitempty"`
	Fade struct {
		Duration uint16 `json:"duration,omitempty"`
	} `json:"fade,omitempty"`
	Flash struct {
		Repetitions uint16 `json:"repetitions,omitempty"`
	} `json:"flash,omitempty"`
}
