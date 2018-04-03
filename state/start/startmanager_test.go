package start_test

import (
	"bytes"

	"github.com/compozed/deployadactyl/state/start"
	"github.com/compozed/deployadactyl/structs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io"
	"reflect"

	"github.com/compozed/deployadactyl/controller/deployer/bluegreen"
	"github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/logger"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"
	"github.com/go-errors/errors"
	"github.com/onsi/gomega/gbytes"
	"github.com/op/go-logging"
)

var _ = Describe("Startmanager", func() {
	var (
		response     io.ReadWriter
		startManager interfaces.ActionCreator
		creator      *mocks.Creator
	)
	BeforeEach(func() {
		logBuffer := bytes.NewBuffer([]byte{})
		log := logger.DefaultLogger(logBuffer, logging.DEBUG, "deployer tests")
		response = gbytes.NewBuffer()
		creator = &mocks.Creator{}
		startManager = start.StartManager{
			CourierCreator: creator,
			Logger:         logger.DeploymentLogger{log, randomizer.StringRunes(10)},
			DeployEventData: structs.DeployEventData{
				DeploymentInfo: &structs.DeploymentInfo{},
				Response:       response,
			},
		}
	})
	Describe("Create", func() {
		Context("when courier build succeeds", func() {
			It("should return a Starter object", func() {
				env := structs.Environment{}
				foundationURL := "foundation url"
				starter, _ := startManager.Create(env, response, foundationURL)

				Expect(reflect.TypeOf(starter)).Should(Equal(reflect.TypeOf(&start.Starter{})))

			})

			It("should return a Starter object with correct data", func() {
				env := structs.Environment{
					Name: "myEnv",
				}
				foundationURL := "foundation url"
				deploymentInfo := structs.DeploymentInfo{
					AppName:  "myApp",
					Username: "bob",
					Password: "password",
				}
				*startManager.(start.StartManager).DeployEventData.DeploymentInfo = deploymentInfo
				starter, _ := startManager.Create(env, response, foundationURL)

				starterData := starter.(*start.Starter)
				Expect(starterData.CFContext.Application).Should(Equal("myApp"))
				Expect(starterData.CFContext.Environment).Should(Equal("myEnv"))
				Expect(starterData.Authorization.Username).Should(Equal("bob"))
				Expect(starterData.Authorization.Password).Should(Equal("password"))
				Expect(starterData.FoundationURL).Should(Equal(foundationURL))

			})
		})

		Context("when courier build failed", func() {
			It("should return an error", func() {
				creator.CourierCreatorFn = func() (interfaces.Courier, error) {
					return nil, errors.New("a test error")
				}

				env := structs.Environment{}
				foundationURL := "foundation url"
				_, err := startManager.Create(env, response, foundationURL)
				Expect(err).ShouldNot(BeNil())
				Expect(err.Error()).Should(ContainSubstring("a test error"))

			})
		})
	})
	Describe("InitiallyError", func() {
		It("should return LoginErrors", func() {
			errors := []error{errors.New("first error")}
			err := startManager.InitiallyError(errors)

			Expect(reflect.TypeOf(err)).Should(Equal(reflect.TypeOf(bluegreen.LoginError{})))
		})
	})
	Describe("ExecuteError", func() {
		It("should return StartError", func() {
			errs := []error{errors.New("first error")}
			err := startManager.ExecuteError(errs)

			Expect(reflect.TypeOf(err)).Should(Equal(reflect.TypeOf(bluegreen.StartError{})))
		})
	})
	Describe("UndoError", func() {
		It("should return RollbackStartError", func() {
			errs := []error{errors.New("first error")}
			executeErrors := []error{errors.New("execute error")}

			err := startManager.UndoError(executeErrors, errs)

			Expect(reflect.TypeOf(err)).Should(Equal(reflect.TypeOf(bluegreen.RollbackStartError{})))
		})
	})
	Describe("SuccessError", func() {
		It("should return FinishStartError", func() {
			errors := []error{errors.New("first error")}
			err := startManager.SuccessError(errors)

			Expect(reflect.TypeOf(err)).Should(Equal(reflect.TypeOf(bluegreen.FinishStartError{})))
		})
	})
})
