// (c) Bernhard Tittelbach, 2013

package r3events

type PresenceUpdate struct {
    Present bool
    Ts int64
}

type SomethingReallyIsMoving struct {
    Movement bool
    Confidence uint8
    Ts int64
}

type TempSensorSpike struct {
    Sensorindex int
    Value float64    
    Ts int64
}

type DustSensorSpike struct {
    Sensorindex int
    Value int64
    Ts int64
}