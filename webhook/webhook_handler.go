package webhook

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	types "github.com/sysincz/prometheus_webhook_logger/types"

	logrus "github.com/Sirupsen/logrus"
	template "github.com/prometheus/alertmanager/template"
)

//Handler A webhook handler with a "ServeHTTP" method:
type Handler struct {
	AlertsChannel chan types.Alert
}

// Handle webhook requests:
func (webhookHandler *Handler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	//prepare for request.Form
	err := request.ParseForm()
	if err != nil {
		panic(err)
	}

	// Read the request body:
	payload, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		log.WithFields(logrus.Fields{"error": err}).Error("Failed to read the request body")
		http.Error(responseWriter, "Failed to read the request body", http.StatusBadRequest)
		return
	}

	// Validate the payload:
	alerts, err := validatePayload(payload, request.Form)
	if err != nil {
		http.Error(responseWriter, "Failed to unmarshal the request-body into an alert", http.StatusBadRequest)
		return
	}

	// Send the alerts to the webhook-logger:
	for alertIndex, alert := range alerts {
		log.WithFields(logrus.Fields{"index": alertIndex, "status": alert.Status, "labels": alert.Labels, "URLValues": alert.URLValues}).Debug("Forwarding an alert to the logger")

		// Enrich the request with the remote-address:
		alert.Address = request.RemoteAddr

		// Put the alert onto the alerts-channel:
		webhookHandler.AlertsChannel <- alert
	}

}

//validatePayload Validate a webhook payload and return a list of Alerts:
func validatePayload(payload []byte, URLValues map[string][]string) ([]types.Alert, error) {

	// Make our response:
	alerts := make([]types.Alert, 0)

	// Make a new Prometheus data-structure to unmarshal the request body into:
	prometheusData := &template.Data{}

	// Unmarshal the request body into the alert:
	err := json.Unmarshal(payload, prometheusData)
	if err != nil {
		log.WithFields(logrus.Fields{"error": err, "payload": payload}).Error("Failed to unmarshal the request body into an alert")
		return alerts, err
	}

	log.WithFields(logrus.Fields{"payload": string(payload)}).Debug("Received a valid webhook alert")
	//First send resolved after that firing (fix change severity)
	//strings := []string{"resolved", "firing"}
	//for _, status := range strings {

	// Iterate over the list of alerts:
	for _, alertDetails := range prometheusData.Alerts {
		//if alertDetails.Status == status {
		// Make a new alert:
		alerts = append(alerts, types.Alert{
			status:            alertDetails.Status,
			labels:            alertDetails.Labels,
			annotations:       alertDetails.Annotations,
			startsAt:          alertDetails.StartsAt,
			endsAt:            alertDetails.EndsAt,
			generatorURL:      alertDetails.GeneratorURL
			receiver:          prometheusData.Receiver,
			groupLabels:       prometheusData.GroupLabels,
			commonLabels:      prometheusData.CommonLabels,
			commonAnnotations: prometheusData.CommonAnnotations,
			externalURL:       prometheusData.ExternalURL,
			urlValues:         URLValues,
		})
		//}

	}
	//}

	return alerts, nil
}
