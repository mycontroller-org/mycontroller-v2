package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	ml "github.com/mycontroller-org/mycontroller/pkg/model"
	srv "github.com/mycontroller-org/mycontroller/pkg/service"
)

func params(r *http.Request) ([]ml.Filter, *ml.Pagination, error) {
	f := mux.Vars(r)
	q := r.URL.Query()
	for k, v := range q {
		f[k] = v[0] // TODO: FIX this to fetch all the values
	}

	// get Pagination arguments
	// start with pagination default values
	p := ml.Pagination{
		Limit:  50,
		Offset: 0,
		SortBy: []ml.Sort{},
	}

	lFunc := func(k string) (int64, error) {
		if v, ok := f[k]; ok {
			vi, err := strconv.Atoi(v)
			if err != nil {
				return 0, err
			}
			return int64(vi), nil
		}
		return 0, fmt.Errorf("Key '%s' not found in the map", k)
	}

	v, err := lFunc("limit")
	if err == nil {
		p.Limit = v
	}

	v, err = lFunc("offset")
	if err == nil {
		p.Offset = v
	}

	// fetch sort options
	if sr, ok := f["sortBy"]; ok {
		s := &[]ml.Sort{}
		err := json.Unmarshal([]byte(sr), s)
		if err != nil {
			return nil, nil, err
		}
		p.SortBy = *s
	}
	// remove these keys from map
	delete(f, "limit")
	delete(f, "offset")
	delete(f, "sortBy")

	filters := make([]ml.Filter, 0)

	for k, v := range f {
		if k != "filter" {
			filters = append(filters, ml.Filter{
				Key:   k,
				Value: v,
			})
		}
	}

	if fj, ok := f["filter"]; ok {
		fs := &[]ml.Filter{}
		err := json.Unmarshal([]byte(fj), fs)
		if err != nil {
			return nil, nil, err
		}
		filters = append(filters, *fs...)
	}

	return filters, &p, nil
}

func findOne(w http.ResponseWriter, r *http.Request, en string, e interface{}) {
	w.Header().Set("Content-Type", "application/json")

	f, _, err := params(r)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = srv.STG.FindOne(en, f, e)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	od, err := json.Marshal(e)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(od)
}

func distinct(w http.ResponseWriter, r *http.Request, e string, fn string) {
	w.Header().Set("Content-Type", "application/json")

	f, _, err := params(r)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	rs, err := srv.STG.Distinct(e, fn, f)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	od, err := json.Marshal(rs)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(od)
}

func findMany(w http.ResponseWriter, r *http.Request, entityName string, entities interface{}) {
	w.Header().Set("Content-Type", "application/json")

	f, p, err := params(r)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = srv.STG.Find(entityName, f, *p, entities)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	od, err := json.Marshal(entities)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(od)
}

func saveEntity(w http.ResponseWriter, r *http.Request, en string, e interface{}, bwFunc func(e interface{}, f *[]ml.Filter) error) {
	w.Header().Set("Content-Type", "application/json")

	d, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	err = json.Unmarshal(d, &e)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	f := make([]ml.Filter, 0)
	if bwFunc != nil {
		err = bwFunc(e, &f)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	err = srv.STG.Upsert(en, f, e)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
