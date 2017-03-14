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
	Pattern          string `json:"pattern,omitempty"`
	Hue              *int64 `json:"hue,omitempty"`
	Brightness       *int64 `json:"brightness,omitempty"`
	Speed            *int64 `json:"speed,omitempty"`
	EffectBrightness *int64 `json:"effectbrightness,omitempty"`
	EffectHue        *int64 `json:"effecthue,omitempty"`
	Arg              *int64 `json:"arg,omitempty"`
	Arg1             *int64 `json:"arg1,omitempty"`
}

type LightCtrlActionOnName struct {
	Name   string
	Action string
}

type FancyLight struct {
	R    *uint16 `json:"r,omitempty"`
	G    *uint16 `json:"g,omitempty"`
	B    *uint16 `json:"b,omitempty"`
	CW   *uint16 `json:"cw,omitempty"`
	WW   *uint16 `json:"ww,omitempty"`
	Fade *struct {
		Duration uint32   `json:"duration,omitempty"`
		Cc       []string `json:"cc,omitempty"`
	} `json:"fade,omitempty"`
	Flash *struct {
		Repetitions uint16   `json:"repetitions,omitempty"`
		Period      uint16   `json:"period,omitempty"`
		Cc          []string `json:"cc,omitempty"`
	} `json:"flash,omitempty"`
}

type CeilingScript struct {
	Script string `json:"script"`
}
