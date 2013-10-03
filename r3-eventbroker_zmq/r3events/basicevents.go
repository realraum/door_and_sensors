// (c) Bernhard Tittelbach, 2013

package r3events


type DoorLockUpdate struct {
    DoorID int
    Locked bool
    Ts int64
}

type DoorAjarUpdate struct {
    DoorID int
    Shut bool
    Ts int64
}

type DoorCommandEvent struct {
    Command string
    Using string
    Who string
    Ts int64
}

type ButtonPressUpdate struct {
    Buttonindex int
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