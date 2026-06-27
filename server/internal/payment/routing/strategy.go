package routing

import (
	"fmt"
	"sort"
)

type Strategy interface {
	Select(routes []Route) (Route, error)
}

type PriorityStrategy struct{}

func NewPriorityStrategy() *PriorityStrategy {
	return &PriorityStrategy{}
}

func (s *PriorityStrategy) Select(routes []Route) (Route, error) {
	enabled := make([]Route, 0, len(routes))

	for _, route := range routes {
		if route.Enabled {
			enabled = append(enabled, route)
		}
	}

	if len(enabled) == 0 {
		return Route{}, fmt.Errorf("no enabled payment route found")
	}

	sort.SliceStable(enabled, func(i, j int) bool {
		return enabled[i].Priority < enabled[j].Priority
	})

	return enabled[0], nil
}
