// (c) Bernhard Tittelbach, 2013

package r3events


type DoorLockUpdate struct {
    Locked bool
    Ts int64
}

type DoorAjarUpdate struct {
    Shut bool
    Ts int64
}

type BackdoorAjarUpdate struct {
    Shut bool
    Ts int64
}

type DoorCommandEvent struct {
    Command string
    Using string
    Who string
    Ts int64
}

type DoorProblemEvent struct {
    Severity int
    Ts int64
}

type BoreDoomButtonPressEvent struct {
    Ts int64
}

type TempSensorUpdate struct {
    Sensorindex int
    Value float64
    Ts int64
}

type IlluminationSensorUpdate struct {
    Sensorindex int
    Value int64
    Ts int64
}

type TimeTick struct {
    Ts int64
}

type MovementSensorUpdate struct {
    Sensorindex int
    Ts int64
}