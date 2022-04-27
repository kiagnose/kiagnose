package runner

import (
	"errors"
	"strings"
)

type status struct {
	failureReasons []string
	results        map[string]string
}

func newStatus() *status {
	return &status{failureReasons: []string{}, results: map[string]string{}}
}

func (s *status) appendFailureReason(err error) {
	s.failureReasons = append(s.failureReasons, err.Error())
}

func (s *status) FailureReason() error {
	if len(s.failureReasons) > 0 {
		return errors.New(strings.Join(s.failureReasons, ", "))
	}
	return nil
}

func (s *status) Succeeded() bool {
	return len(s.failureReasons) == 0
}

func (s *status) setResults(newResults map[string]string) {
	s.results = newResults
}

func (s *status) Results() map[string]string {
	return s.results
}

func (s *status) StringsMap() map[string]string {
	return s.results
}
