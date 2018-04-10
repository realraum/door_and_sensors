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
