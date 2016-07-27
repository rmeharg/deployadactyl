package bluegreen_test

import (
	"bytes"
	"errors"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/randomizer"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/compozed/deployadactyl/test/mocks"
	"github.com/op/go-logging"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bluegreen", func() {

	var (
		environmentName string
		domainName      string
		appName         string
		appPath         string
		org             string
		space           string
		pushOutput      string
		loginOutput     string
		username        string
		password        string
		pusherFactory   *mocks.PusherFactory
		pushers         []*mocks.Pusher
		log             *logging.Logger
		blueGreen       BlueGreen
		environment     config.Environment
		deploymentInfo  S.DeploymentInfo
		buffer          *bytes.Buffer
	)

	BeforeEach(func() {
		environmentName = "environmentName-" + randomizer.StringRunes(10)
		domainName = "domainName-" + randomizer.StringRunes(10)
		appName = "appName-" + randomizer.StringRunes(10)
		appPath = "appPath-" + randomizer.StringRunes(10)
		org = "org-" + randomizer.StringRunes(10)
		space = "space-" + randomizer.StringRunes(10)
		pushOutput = "pushOutput-" + randomizer.StringRunes(10)
		loginOutput = "loginOutput-" + randomizer.StringRunes(10)
		username = "username-" + randomizer.StringRunes(10)
		password = "password-" + randomizer.StringRunes(10)
		buffer = &bytes.Buffer{}

		pusherFactory = &mocks.PusherFactory{}
		pushers = nil

		log = logger.DefaultLogger(GinkgoWriter, logging.DEBUG, "test")

		blueGreen = BlueGreen{pusherFactory, log}

		environment = config.Environment{
			Name:   environmentName,
			Domain: domainName,
		}

		deploymentInfo = S.DeploymentInfo{
			Username: username,
			Password: password,
			Org:      org,
			Space:    space,
			AppName:  appName,
		}
	})

	Context("when any logins fail", func() {
		It("should not start to deploy", func() {
			environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}

			for index, _ := range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)

				if index == 0 {
					pusher.LoginCall.Write.Output = loginOutput
					pusher.LoginCall.Returns.Error = errors.New("bork")
				} else {
					pusher.LoginCall.Write.Output = loginOutput
					pusher.LoginCall.Returns.Error = nil
				}

				pusher.CleanUpCall.Returns.Error = nil
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, buffer)).ToNot(Succeed())

			for i, pusher := range pushers {
				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(environment.Foundations[i]))
				Expect(pusher.LoginCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			}

			Expect(buffer.String()).To(Equal(loginOutput + loginOutput))
		})
	})

	Context("when pushes are successful", func() {
		It("can push an app to a single foundation", func() {
			foundationURL := "foundationURL-" + randomizer.StringRunes(10)
			environment.Foundations = []string{foundationURL}

			pusher := &mocks.Pusher{}
			pushers = append(pushers, pusher)
			pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)

			pusher.LoginCall.Write.Output = loginOutput
			pusher.LoginCall.Returns.Error = nil
			pusher.PushCall.Write.Output = pushOutput
			pusher.PushCall.Returns.Error = nil
			pusher.FinishPushCall.Returns.Error = nil
			pusher.CleanUpCall.Returns.Error = nil

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, buffer)).To(Succeed())

			Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(foundationURL))
			Expect(pusher.LoginCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))
			Expect(pusher.PushCall.Received.FoundationURL).To(Equal(foundationURL))
			Expect(pusher.PushCall.Received.Domain).To(Equal(domainName))
			Expect(pusher.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			Expect(pusher.FinishPushCall.Received.FoundationURL).To(Equal(foundationURL))
			Expect(pusher.FinishPushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))

			Expect(buffer.String()).To(Equal(loginOutput + pushOutput))
		})

		It("can push an app to multiple foundations", func() {
			environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}

			for range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)

				pusher.LoginCall.Write.Output = loginOutput
				pusher.LoginCall.Returns.Error = nil
				pusher.PushCall.Write.Output = pushOutput
				pusher.PushCall.Returns.Error = nil
				pusher.FinishPushCall.Returns.Error = nil
				pusher.CleanUpCall.Returns.Error = nil
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, buffer)).To(Succeed())

			for i, pusher := range pushers {
				foundationURL := environment.Foundations[i]

				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(pusher.LoginCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(pusher.PushCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(pusher.PushCall.Received.Domain).To(Equal(domainName))
				Expect(pusher.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.FinishPushCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(pusher.FinishPushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			}

			Expect(buffer.String()).To(Equal(loginOutput + pushOutput + loginOutput + pushOutput))
		})
	})

	Context("when at least one push is unsuccessful", func() {
		It("should rollback all recent pushes", func() {
			environment.Foundations = []string{randomizer.StringRunes(10), randomizer.StringRunes(10)}

			for index, _ := range environment.Foundations {
				pusher := &mocks.Pusher{}
				pushers = append(pushers, pusher)
				pusherFactory.CreatePusherCall.Returns.Pushers = append(pusherFactory.CreatePusherCall.Returns.Pushers, pusher)

				pusher.LoginCall.Write.Output = loginOutput
				pusher.LoginCall.Returns.Error = nil

				if index == 0 {
					pusher.PushCall.Write.Output = pushOutput
					pusher.PushCall.Returns.Error = nil
				} else {
					pusher.PushCall.Write.Output = pushOutput
					pusher.PushCall.Returns.Error = errors.New("bork")
				}

				pusher.UnpushCall.Returns.Error = nil
				pusher.CleanUpCall.Returns.Error = nil
			}

			Expect(blueGreen.Push(environment, appPath, deploymentInfo, buffer)).ToNot(Succeed())

			for i, pusher := range pushers {
				foundationURL := environment.Foundations[i]

				Expect(pusher.LoginCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(pusher.LoginCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.PushCall.Received.AppPath).To(Equal(appPath))
				Expect(pusher.PushCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(pusher.PushCall.Received.Domain).To(Equal(domainName))
				Expect(pusher.PushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
				Expect(pusher.UnpushCall.Received.FoundationURL).To(Equal(foundationURL))
				Expect(pusher.UnpushCall.Received.DeploymentInfo).To(Equal(deploymentInfo))
			}

			Expect(buffer.String()).To(Equal(loginOutput + pushOutput + loginOutput + pushOutput))
		})
	})
})