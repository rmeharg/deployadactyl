package prechecker_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/compozed/deployadactyl/config"
	. "github.com/compozed/deployadactyl/controller/deployer/prechecker"
	S "github.com/compozed/deployadactyl/structs"
	"github.com/compozed/deployadactyl/test/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prechecker", func() {
	Describe("AssertAllFoundationsUp", func() {
		var (
			httpStatus        int
			err               error
			foundationApiURLs []string
			prechecker        Prechecker
			eventManager      *mocks.EventManager
			configServer      *httptest.Server
			environment       config.Environment
			event             S.Event
		)

		BeforeEach(func() {
			foundationApiURLs = []string{}

			eventManager = &mocks.EventManager{}
			prechecker = Prechecker{EventManager: eventManager}

			configServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				foundationApiURLs = append(foundationApiURLs, r.URL.Path)
				w.WriteHeader(httpStatus)
			}))
			environment = config.Environment{
				Foundations: []string{configServer.URL},
			}
		})

		AfterEach(func() {
			configServer.Close()
		})

		Context("when no foundations are given", func() {
			It("returns an error and emits an event", func() {
				environment.Foundations = nil

				precheckerEventData := S.PrecheckerEventData{
					Environment: environment,
					Description: "no foundations configured",
				}
				event = S.Event{
					Type: "validate.foundationsUnavailable",
					Data: precheckerEventData,
				}
				eventManager.EmitCall.Returns.Error = nil

				Expect(prechecker.AssertAllFoundationsUp(environment)).ToNot(Succeed())
				Expect(eventManager.EmitCall.Received.Event).To(Equal(event))
			})
		})

		Context("when all foundations return a 200 OK", func() {
			It("returns a nil error", func() {
				httpStatus = http.StatusOK

				err = prechecker.AssertAllFoundationsUp(environment)
				Expect(err).ToNot(HaveOccurred())

				Expect(foundationApiURLs).To(ConsistOf("/v2/info"))
			})
		})

		Context("when a foundation returns a 500 internal server error", func() {
			It("returns an error and emits an event", func() {
				precheckerEventData := S.PrecheckerEventData{
					Environment: environment,
					Description: "deploy aborted, one or more CF foundations unavailable",
				}
				event = S.Event{
					Type: "validate.foundationsUnavailable",
					Data: precheckerEventData,
				}
				eventManager.EmitCall.Returns.Error = nil

				httpStatus = http.StatusInternalServerError
				err = prechecker.AssertAllFoundationsUp(environment)
				Expect(err).To(HaveOccurred())

				Expect(foundationApiURLs).To(ConsistOf("/v2/info"))
				Expect(eventManager.EmitCall.Received.Event).To(Equal(event))
			})
		})

		Context("when a foundation returns a 404 not found", func() {
			It("returns an error and emits an event", func() {
				precheckerEventData := S.PrecheckerEventData{
					Environment: environment,
					Description: "deploy aborted, one or more CF foundations unavailable",
				}
				event = S.Event{
					Type: "validate.foundationsUnavailable",
					Data: precheckerEventData,
				}
				eventManager.EmitCall.Returns.Error = nil

				httpStatus = http.StatusNotFound

				err = prechecker.AssertAllFoundationsUp(environment)
				Expect(err).To(HaveOccurred())

				Expect(eventManager.EmitCall.Received.Event).To(Equal(event))
			})
		})
	})
})