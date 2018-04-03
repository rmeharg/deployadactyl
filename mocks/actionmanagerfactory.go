package mocks

import (
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/structs"
)

// PushManager handmade mock for tests.
type PushManagerFactory struct {
	PusherCreatorCall struct {
		Called   bool
		Received struct {
			DeployEventData structs.DeployEventData
		}
		Returns struct {
			ActionCreator interfaces.ActionCreator
		}
	}
}

// CreatePusher mock method.

func (p *PushManagerFactory) PusherCreator(deployEventData structs.DeployEventData) interfaces.ActionCreator {
	p.PusherCreatorCall.Called = true
	p.PusherCreatorCall.Received.DeployEventData = deployEventData

	return p.PusherCreatorCall.Returns.ActionCreator
}

type StopManagerFactory struct {
	StopManagerCall struct {
		Called   bool
		Received struct {
			DeployEventData structs.DeployEventData
		}
		Returns struct {
			ActionCreater interfaces.ActionCreator
		}
	}
}

func (s *StopManagerFactory) StopManager(DeployEventData structs.DeployEventData) interfaces.ActionCreator {
	s.StopManagerCall.Called = true
	s.StopManagerCall.Received.DeployEventData = DeployEventData

	return s.StopManagerCall.Returns.ActionCreater
}
