package notifier

import (
	"fmt"

	"github.com/containrrr/shoutrrr"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Send(urlValue, title, message string) error {
	body := fmt.Sprintf("%s\n%s", title, message)
	return shoutrrr.Send(urlValue, body)
}
