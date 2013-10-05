// (c) Bernhard Tittelbach, 2013

package main

import (
    pubsub "github.com/tuxychandru/pubsub"
    r3events "svn.spreadspace.org/realraum/go.svn/r3-eventbroker_zmq/r3events"
    )

func ParseZMQr3Event(lines [][]byte, ps *pubsub.PubSub) {
    evnt, pubsubcat, err := r3events.UnmarshalByteByte2Event(lines)
    if err != nil { return }
    ps.Pub(evnt, pubsubcat)
}
