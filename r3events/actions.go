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
