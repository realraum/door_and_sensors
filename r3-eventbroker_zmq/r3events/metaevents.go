// (c) Bernhard Tittelbach, 2013

package r3events

type PresenceUpdate struct {
    Present bool
    Ts int64
}

type SomethingReallyIsMoving struct {
    Movement bool
    Ts int64
}