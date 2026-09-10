package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	propertyerrors "github.com/example/property-error-capture"
)

func main() {
	client, err := propertyerrors.NewClient(os.Getenv("INFRAI_API_KEY"), nil)
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /property-errors", captureHandler(client))
	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("property error service listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func captureHandler(client *propertyerrors.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var failure propertyerrors.Failure
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&failure); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		capture, err := propertyerrors.Classify(failure)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		data, err := client.Capture(r.Context(), capture, "property-error:"+failure.EventID)
		if err != nil {
			status := http.StatusBadGateway
			var apiErr *propertyerrors.APIError
			if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
				status = apiErr.Status
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"grouping_key": capture.Fingerprint, "data": json.RawMessage(data)})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
