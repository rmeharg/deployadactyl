package healthchecker_test

import (
	"errors"
	"fmt"
	"net/http"

	. "github.com/compozed/deployadactyl/eventmanager/handlers/healthchecker"
	"github.com/compozed/deployadactyl/mocks"
	"github.com/compozed/deployadactyl/randomizer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"

	I "github.com/compozed/deployadactyl/interfaces"
	"github.com/compozed/deployadactyl/state/push"
	"github.com/op/go-logging"
)

var _ = Describe("Healthchecker", func() {

	var (
		randomAppName       string
		randomFoundationURL string
		randomDomain        string
		randomEndpoint      string
		randomUsername      string
		randomPassword      string
		randomOrg           string
		randomSpace         string
		randomHostname      string
		randomEnvironment   string

		ievent        push.PushFinishedEvent
		healthchecker HealthChecker
		client        *mocks.Client
		courier       *mocks.Courier
		logBuffer     *Buffer
	)

	BeforeEach(func() {
		randomAppName = "randomAppName-" + randomizer.StringRunes(10)

		s := "random-" + randomizer.StringRunes(10)

		randomFoundationURL = fmt.Sprintf("https://api.cf.%s.com", s)
		randomDomain = fmt.Sprintf("apps.%s.com", s)
		randomHostname = randomAppName

		randomEndpoint = "/" + randomizer.StringRunes(10)

		randomUsername = "randomUsername" + randomizer.StringRunes(10)
		randomPassword = "randomPassword" + randomizer.StringRunes(10)
		randomOrg = "randomOrg" + randomizer.StringRunes(10)
		randomSpace = "randomSpace" + randomizer.StringRunes(10)
		randomEnvironment = "randomEnvironment" + randomizer.StringRunes(10)

		courier = &mocks.Courier{}
		client = &mocks.Client{}

		logBuffer = NewBuffer()
		ievent = push.PushFinishedEvent{
			TempAppWithUUID:     randomAppName,
			FoundationURL:       randomFoundationURL,
			Courier:             courier,
			HealthCheckEndpoint: randomEndpoint,
			Auth: I.Authorization{
				Username: randomUsername,
				Password: randomPassword,
			},
			CFContext: I.CFContext{
				Organization: randomOrg,
				Space:        randomSpace,
				Environment:  randomEnvironment,
			},
			Log: I.DeploymentLogger{Log: I.DefaultLogger(logBuffer, logging.DEBUG, "healthchecker_test")},
		}

		healthchecker = HealthChecker{
			OldURL:                  "api.cf",
			NewURL:                  "apps",
			SilentDeployURL:         "silentapps",
			SilentDeployEnvironment: "silentenvironment",
			Client:                  client,
		}

		client.GetCall.Returns.Response = http.Response{
			StatusCode: http.StatusOK,
			Body:       NewBuffer(),
		}
	})

	Describe("OnEvent", func() {
		Context("the new build application is healthy", func() {
			Context("the endpoint provided is valid", func() {
				It("does not return an error", func() {
					client.GetCall.Returns.Response = http.Response{StatusCode: http.StatusOK}

					err := healthchecker.PushFinishedEventHandler(ievent)

					Expect(err).ToNot(HaveOccurred())
				})

				It("maps a new temporary route", func() {
					healthchecker.PushFinishedEventHandler(ievent)

					Expect(courier.MapRouteCall.Received.AppName[0]).To(Equal(randomAppName))
					Expect(courier.MapRouteCall.Received.Domain[0]).To(Equal(randomDomain))
					Expect(courier.MapRouteCall.Received.Hostname[0]).To(Equal(randomAppName))
				})

				It("formats the foundation url", func() {
					healthchecker.PushFinishedEventHandler(ievent)

					Expect(client.GetCall.Received.URL).To(Equal(fmt.Sprintf("https://%s.%s%s", randomAppName, randomDomain, randomEndpoint)))
				})

				It("unmaps the temporary route", func() {
					healthchecker.PushFinishedEventHandler(ievent)

					Expect(courier.UnmapRouteCall.Received.AppName).To(Equal(randomAppName))
					Expect(courier.UnmapRouteCall.Received.Domain).To(Equal(randomDomain))
					Expect(courier.UnmapRouteCall.Received.Hostname).To(Equal(randomAppName))
				})

				It("deletes the temporary routes", func() {
					healthchecker.PushFinishedEventHandler(ievent)

					Expect(courier.DeleteRouteCall.Received.Domain).To(Equal(randomDomain))
					Expect(courier.DeleteRouteCall.Received.Hostname).To(Equal(randomHostname))
				})

				It("unmaps the temporary route before deleting it", func() {
					healthchecker.PushFinishedEventHandler(ievent)

					Expect(courier.UnmapRouteCall.OrderCalled < courier.DeleteRouteCall.OrderCalled).To(Equal(true))
				})

				It("prints success logs to the console", func() {
					client.GetCall.Returns.Response = http.Response{StatusCode: http.StatusOK}

					healthchecker.PushFinishedEventHandler(ievent)

					Eventually(logBuffer).Should(Say("starting health check"))
					Eventually(logBuffer).Should(Say("mapping temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("mapped temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("checking route https://%s.%s%s", randomAppName, randomDomain, randomEndpoint))
					Eventually(logBuffer).Should(Say("health check successful for https://%s.%s%s", randomAppName, randomDomain, randomEndpoint))
					Eventually(logBuffer).Should(Say("unmapping temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("unmapped temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("finished health check"))
				})

				It("maps route for silent deploy environment", func() {
					healthchecker = HealthChecker{
						OldURL:                  "api.cf",
						NewURL:                  "apps",
						SilentDeployURL:         "silentapps",
						SilentDeployEnvironment: randomEnvironment,
						Client:                  client,
					}

					healthchecker.PushFinishedEventHandler(ievent)

					Expect(courier.MapRouteCall.Received.Domain[0]).To(ContainSubstring("silentapps"))
					Eventually(logBuffer).Should(Say("finished health check"))
				})
			})

			Context("the endpoint provided is not valid", func() {
				BeforeEach(func() {
					client.GetCall.Returns.Response = http.Response{
						StatusCode: http.StatusNotFound,
						Body:       NewBuffer(),
					}
				})

				It("returns an error", func() {
					body := []byte("Could not find page")

					buf := NewBuffer()
					buf.Write(body)

					client.GetCall.Returns.Response = http.Response{
						StatusCode: http.StatusNotFound,
						Body:       buf,
					}

					err := healthchecker.PushFinishedEventHandler(ievent)

					Expect(err).To(MatchError(HealthCheckError{http.StatusNotFound, randomEndpoint, body}))
				})

				It("prints the endpoint error to the console", func() {
					healthchecker.PushFinishedEventHandler(ievent)

					Eventually(logBuffer).Should(Say("starting health check"))
					Eventually(logBuffer).Should(Say("mapping temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("mapped temporary route %s.%s", randomAppName, randomDomain))
					Eventually(logBuffer).Should(Say("checking route https://%s.%s%s", randomAppName, randomDomain, randomEndpoint))
					Eventually(logBuffer).Should(Say("health check failed"))
					Eventually(logBuffer).Should(Say(randomEndpoint))
				})
			})
		})

		Context("the new build application is not healthy", func() {
			It("returns an error", func() {
				client.GetCall.Returns.Response = http.Response{
					StatusCode: http.StatusNotFound,
					Body:       NewBuffer(),
				}

				err := healthchecker.PushFinishedEventHandler(ievent)
				Expect(err).To(MatchError(HealthCheckError{http.StatusNotFound, randomEndpoint, []byte{}}))
			})
		})

		Context("when mapping the temporary route fails", func() {
			It("returns an error", func() {
				courier.MapRouteCall.Returns.Output = append(courier.MapRouteCall.Returns.Output, []byte("map route output"))
				courier.MapRouteCall.Returns.Error = append(courier.MapRouteCall.Returns.Error, errors.New("map route error"))

				healthchecker.PushFinishedEventHandler(ievent)

				Eventually(logBuffer).Should(Say("mapping temporary route"))
				Eventually(logBuffer).Should(Say("failed to map temporary route"))
				Eventually(logBuffer).Should(Say("map route output"))
			})
		})

		Context("when the client fails to send the GET", func() {
			It("returns an error", func() {
				client.GetCall.Returns.Error = errors.New("client GET error")

				err := healthchecker.PushFinishedEventHandler(ievent)

				Expect(err).To(MatchError(ClientError{errors.New("client GET error")}))
			})

			It("prints the error to the logs", func() {
				client.GetCall.Returns.Error = errors.New("client GET error")

				healthchecker.PushFinishedEventHandler(ievent)

				Eventually(logBuffer).Should(Say("checking route"))
				Eventually(logBuffer).Should(Say("client GET error"))
			})
		})

		Context("when a health check endpoint is not provided", func() {
			It("returns nil", func() {
				ievent = push.PushFinishedEvent{
					Courier:             courier,
					TempAppWithUUID:     randomAppName,
					FoundationURL:       randomFoundationURL,
					HealthCheckEndpoint: "",
				}

				err := healthchecker.PushFinishedEventHandler(ievent)

				Expect(err).To(BeNil())
			})
		})

		Context("when unmapping the temporary route fails", func() {
			It("prints output to the logs", func() {
				courier.UnmapRouteCall.Returns.Output = []byte("unmap route output")
				courier.UnmapRouteCall.Returns.Error = errors.New("unmap route error")

				healthchecker.PushFinishedEventHandler(ievent)

				Eventually(logBuffer).Should(Say("unmapping temporary route"))
				Eventually(logBuffer).Should(Say("failed to unmap temporary route"))
				Eventually(logBuffer).Should(Say("unmap route output"))
				Eventually(logBuffer).Should(Say("finished"))
			})
		})
	})

	Describe("format of endpoint parameter", func() {
		Context("when the endpoint does not include a '/'", func() {
			It("adds the leading '/'", func() {
				endpoint := "health"

				healthchecker.Check(randomFoundationURL, endpoint, I.DeploymentLogger{Log: I.DefaultLogger(logBuffer, logging.DEBUG, "healthchecker_test")})

				Expect(client.GetCall.Received.URL).To(Equal(fmt.Sprintf("%s/%s", randomFoundationURL, endpoint)))
			})
		})

		Context("when the endpoint does include a '/'", func() {
			It("does not add the leading '/'", func() {
				endpoint := "/health"

				healthchecker.Check(randomFoundationURL, endpoint, I.DeploymentLogger{Log: I.DefaultLogger(logBuffer, logging.DEBUG, "healthchecker_test")})

				Expect(client.GetCall.Received.URL).To(Equal(fmt.Sprintf("%s%s", randomFoundationURL, endpoint)))
			})
		})
	})
})
