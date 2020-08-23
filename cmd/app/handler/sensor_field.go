package handler

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	ml "github.com/mycontroller-org/backend/pkg/model"
	sml "github.com/mycontroller-org/backend/pkg/model/sensor"
)

func registerSensorFieldRoutes(router *mux.Router) {
	router.HandleFunc("/api/sensorfield", listSensorFields).Methods(http.MethodGet)
	router.HandleFunc("/api/sensorfield/{id}", getSensorField).Methods(http.MethodGet)
	router.HandleFunc("/api/sensorfield", updateSensorField).Methods(http.MethodPost)
}

func listSensorFields(w http.ResponseWriter, r *http.Request) {
	findMany(w, r, ml.EntitySensorField, &[]sml.SensorField{})
}

func getSensorField(w http.ResponseWriter, r *http.Request) {
	findOne(w, r, ml.EntitySensorField, &sml.SensorField{})
}

func updateSensorField(w http.ResponseWriter, r *http.Request) {
	bwFunc := func(d interface{}, f *[]ml.Filter) error {
		e := d.(*sml.SensorField)
		if e.ID == "" {
			return errors.New("ID should not be an empty")
		}
		return nil
	}
	saveEntity(w, r, ml.EntitySensorField, &sml.SensorField{}, bwFunc)
}
