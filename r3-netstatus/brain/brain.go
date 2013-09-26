// (c) Bernhard Tittelbach, 2013

package brain

import "errors"

type informationtuple struct {
    name string
    value interface{}
}

type informationretrievalpath struct {
    name string
    returnpath chan interface{}
}

type hippocampus map[string]interface{}

type Brain struct {
    storeTuple chan informationtuple
    retrieveValue chan informationretrievalpath
    shutdown chan bool
}

func New() *Brain {
    b := new(Brain)
    b.storeTuple = make(chan informationtuple)
    b.retrieveValue = make(chan informationretrievalpath)
    go b.runBrain()
    return b
}

func (b *Brain) runBrain() {
    var h hippocampus = make(hippocampus)
    for {
        select {
            case newtuple := <- b.storeTuple:
                h[newtuple.name] = newtuple.value

            case retrievvalue := <- b.retrieveValue:
                v, e := h[retrievvalue.name]
                if e {
                    retrievvalue.returnpath <- v
                } else {
                    retrievvalue.returnpath <- nil
                }

            case <- b.shutdown:
                break
        }
    }
}

func (b *Brain) Shutdown() {
    b.shutdown <- true
}

func (b *Brain) Oboite(name string, value interface{}) {
    b.storeTuple <- informationtuple{name, value}
}

func (b *Brain) OmoiDashite(name string) (interface{}, error) {
    rc := make(chan interface{})
    b.retrieveValue <- informationretrievalpath{name, rc}
    v := <- rc
    if v == nil {
        return v, errors.New("name not in brain")
    }
    return v, nil
}

func (b *Brain) OmoiDashiteBool(name string) (bool, error) {
    v, e := b.OmoiDashite(name)
    if e != nil {
        return false, e
    }
    vc, ok := v.(bool)
    if !ok {
        return false, errors.New(name + " does not have type bool")
    }
    return vc, nil
}

func (b *Brain) OmoiDashiteInt(name string) (int, error) {
    v, e := b.OmoiDashite(name)
    if e != nil {
        return 0, e
    }
    vc, ok := v.(int)
    if !ok {
        return 0, errors.New(name + " does not have type int")
    }
    return vc, nil
}

func (b *Brain) OmoiDashiteFloat(name string) (float64, error) {
    v, e := b.OmoiDashite(name)
    if e != nil {
        return 0, e
    }
    vc, ok := v.(float64)
    if !ok {
        return 0, errors.New(name + " does not have type float64")
    }
    return vc, nil
}

func (b *Brain) OmoiDashiteString(name string) (string, error) {
    v, e := b.OmoiDashite(name)
    if e != nil {
        return "", e
    }
    vc, ok := v.(string)
    if !ok {
        return "", errors.New(name + " does not have type string")
    }
    return vc, nil
}