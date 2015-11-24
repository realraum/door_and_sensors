// (c) Bernhard Tittelbach, 2013

package r3events

type PresenceUpdate struct {
	Present bool
	Ts      int64
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
