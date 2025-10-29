package closer

import (
	log "github.com/sirupsen/logrus"
)

var globalCloser = NewCloser()

type Closer struct {
	fns []func() error
}

func NewCloser() *Closer {
	return &Closer{}
}

func Add(f ...func() error) {
	globalCloser.Add(f...)
}

func CloseAll() {
	globalCloser.CloseAll()
}

func (c *Closer) Add(f ...func() error) {
	c.fns = append(c.fns, f...)
}

func (c *Closer) CloseAll() {
	errs := make([]error, len(c.fns))
	for _, f := range c.fns {
		errs = append(errs, f())
	}

	for _, err := range errs {
		if err != nil {
			log.Errorf("error returned from closer: %s", err)
		}
	}
}
