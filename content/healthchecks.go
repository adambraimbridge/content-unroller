package content

import (
	"fmt"
	"net/http"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
	"github.com/pkg/errors"
)

type ServiceConfig struct { // both
	ContentStoreAppName        string
	ContentStoreAppHealthURI   string
	ContentPreviewAppName      string
	ContentPreviewAppHealthURI string
	HTTPClient                 *http.Client
}

func (sc *ServiceConfig) GtgCheck() gtg.Status { // both
	contentStoreCheck := func() gtg.Status {
		msg, err := sc.checkServiceAvailability(sc.ContentStoreAppName, sc.ContentStoreAppHealthURI)
		if err != nil {
			return gtg.Status{GoodToGo: false, Message: msg}
		}
		return gtg.Status{GoodToGo: true}
	}
	contentPreviewCheck := func() gtg.Status {
		msg, err := sc.checkServiceAvailability(sc.ContentPreviewAppName, sc.ContentPreviewAppHealthURI)
		if err != nil {
			return gtg.Status{GoodToGo: false, Message: msg}
		}

		return gtg.Status{GoodToGo: true}
	}
	return gtg.FailFastParallelCheck([]gtg.StatusChecker{
		contentStoreCheck,
		contentPreviewCheck,
	})()
}

func (sc *ServiceConfig) ContentStoreCheck() fthealth.Check {  // both
	return fthealth.Check{
		ID:               fmt.Sprintf("check-connect-%s", sc.ContentStoreAppName),
		Name:             fmt.Sprintf("Check connectivity to %s", sc.ContentStoreAppName),
		Severity:         1,
		BusinessImpact:   "Unrolled images and dynamic content won't be available",
		TechnicalSummary: fmt.Sprintf(`Cannot connect to %v.`, sc.ContentStoreAppName),
		PanicGuide:       "https://dewey.in.ft.com/runbooks/contentreadapi",
		Checker: func() (string, error) {
			return sc.checkServiceAvailability(sc.ContentStoreAppName, sc.ContentStoreAppHealthURI)
		},
	}
}

func (sc *ServiceConfig) ContentPreviewCheck() fthealth.Check { // preview
	return fthealth.Check{
		ID:               fmt.Sprintf("check-connect-%s", sc.ContentPreviewAppName),
		Name:             fmt.Sprintf("Check connectivity to %s", sc.ContentPreviewAppName),
		Severity:         1,
		BusinessImpact:   "Unrolled dynamic content won't be available",
		TechnicalSummary: fmt.Sprintf(`Cannot connect to %v.`, sc.ContentPreviewAppName),
		PanicGuide:       "https://dewey.in.ft.com/runbooks/content-preview",
		Checker: func() (string, error) {
			return sc.checkServiceAvailability(sc.ContentPreviewAppName, sc.ContentPreviewAppHealthURI)
		},
	}
}

func (sc *ServiceConfig) checkServiceAvailability(serviceName string, healthURI string) (string, error) { // both
	req, err := http.NewRequest(http.MethodGet, healthURI, nil)
	resp, err := sc.HTTPClient.Do(req)
	if err != nil {
		return "Error", errors.Errorf("%s service is unreachable: %v", serviceName, err)
	}
	if resp.StatusCode != http.StatusOK {
		return "Error", errors.Errorf("%s service is not responding with OK. Status=%d", serviceName, resp.StatusCode)
	}
	return "Ok", nil
}
