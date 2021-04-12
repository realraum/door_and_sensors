// (c) Bernhard Tittelbach, 2013

package r3events

type PresenceUpdate struct {
	Present  bool
	InSpace1 bool
	InSpace2 bool
	Ts       int64
}

type SomethingReallyIsMoving struct {
	Movement   bool
	Confidence uint8
	Ts         int64
}

type MovementSum struct {
	Sensorindex     int
	NumEvents       int
	IntervalSeconds int
	Ts              int64
}

//Sent by the system to indicate that the system thinks everbody left, but is unsure
//and that is has started a timeout after which presence will be false if nobody presses a button
//	TYPE_PRESENCETIMEOUTSTART      string = "presencetimeoutstart"
type PresenceTimeoutStartEvent struct {
	TimeoutSec           int
	PresenceAfterTimeout bool  `json:",omitempty"`
	Ts                   int64 `json:",omitempty"`
}

type TempSensorSpike struct {
	Location string
	Value    float64
	Ts       int64
}

type DustSensorSpike struct {
	Location string
	Value    int64
	Ts       int64
}

type HumiditySensorSpike struct {
	Location string
	Percent  float64
	Ts       int64
}

type SensorLost struct {
	Topic         string
	LastSeen      int64
	UsualInterval int64
	Ts            int64
}

type DuskOrDawn struct {
	Event        string // Sunset, CivilDusk, NauticalDusk, AstronomicalDusk, AstronomicalDawn, NauticalDawn, CivilDawn, Sunrise
	HaveSunlight bool
	Ts           int64
}

type AggregateContactsensor struct {
	AllDoorsShut   bool
	AllWindowsShut bool
	AllDoorsLocked bool
	Ts             int64 `json:",omitempty"`
}
