package utils

import (
	"./logevent"
	"errors"
	"github.com/codegangsta/inject"
)

type TypeFilterConfig interface {
	TypeConfig
	Event(logevent.LogEvent) logevent.LogEvent
}

type FilterConfig struct {
	CommonConfig
}

type FilterHandler interface{}

var (
	mapFilterHandler = map[string]FilterHandler{}
)

func RegistFilterHandler(name string, handler FilterHandler) {
	mapFilterHandler[name] = handler
}

func (t *Config) RunFilters() (err error) {
	_, err = t.Injector.Invoke(t.runFilters)
	return
}

func (c *Config) runFilters(inchan InChan, outchan OutChan) (err error) {
	filters, err := c.getFilters()
	if err != nil {
		return
	}

	go func() {
		for {
			select {
			case event := <-inchan:
				for _, filter := range filters {
					event = filter.Event(event)
				}
				outchan <- event
			}
		}
	}()
	return
}

func (c *Config) getFilters() (filters []TypeFilterConfig, err error) {
	for _, confraw := range c.FilterRaw {
		handler, ok := mapFilterHandler[confraw["type"].(string)]
		if !ok {
			err = errors.New(confraw["type"].(string))
			return
		}

		inj := inject.New()
		inj.SetParent(c)
		inj.Map(&confraw)

		refvs, err := inj.Invoke(handler)
		if err != nil {
			return []TypeFilterConfig{}, err
		}

		for _, refv := range refvs {
			if !refv.CanInterface() {
				continue
			}
			if conf, ok := refv.Interface().(TypeFilterConfig); ok {
				conf.SetInjector(inj)
				filters = append(filters, conf)
			}
		}
	}
	return
}
