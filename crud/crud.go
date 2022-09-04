package crud

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bluebrown/dqlite-kubernetes-demo/model"
	"github.com/felixge/httpsnoop"
	"github.com/julienschmidt/httprouter"
	"k8s.io/klog/v2"
)

func New(ctx context.Context, db *sql.DB) *httprouter.Router {
	queries := model.New(db)
	router := httprouter.New()

	router.GET("/healthz", healthz())
	router.GET("/readyz", readyz(db))

	router.POST("/authors", loggingMw(create(queries, "/authors/%d")))
	router.GET("/authors", loggingMw(list(queries)))
	router.PUT("/authors/:id", loggingMw(update(queries)))
	router.DELETE("/authors/:id", loggingMw(delete(queries)))

	return router
}

// health checks

func healthz() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {}
}

func readyz(db *sql.DB) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if _, err := db.QueryContext(r.Context(), "select 1;"); err != nil {
			klog.ErrorS(err, "could not run query for readiness probe")
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}
}

// crud

func create(q *model.Queries, locStr string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		var payload model.CreateAuthorParams
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			errorResponse(w, r, http.StatusBadRequest, err)
			return
		}
		id, err := q.CreateAuthor(r.Context(), payload)
		if err != nil {
			errorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Location", fmt.Sprintf(locStr, id))
		w.WriteHeader(http.StatusCreated)
	}
}

func list(q *model.Queries) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		data, err := q.ListAuthors(r.Context())
		if err != nil {
			errorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
		response(w, r, http.StatusOK, &data)
	}
}

func update(q *model.Queries) httprouter.Handle {
	type req struct {
		Name string
		Bio  *string
	}
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		var payload req
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			errorResponse(w, r, http.StatusBadRequest, err)
			return
		}
		id, err := strconv.ParseInt(p.ByName("id"), 10, 64)
		if err != nil {
			errorResponse(w, r, http.StatusBadRequest, err)
			return
		}
		if err := q.UpdateAuthor(r.Context(), model.UpdateAuthorParams{
			ID:   id,
			Name: payload.Name,
			Bio:  payload.Bio,
		}); err != nil {
			errorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
	}
}

func delete(q *model.Queries) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		id, err := strconv.ParseInt(p.ByName("id"), 10, 64)
		if err != nil {
			errorResponse(w, r, http.StatusBadRequest, err)
			return
		}
		if err := q.DeleteAuthor(r.Context(), id); err != nil {
			errorResponse(w, r, http.StatusInternalServerError, err)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// response helpers

func response(w http.ResponseWriter, r *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		klog.ErrorS(err, "could not send response")
	}
}

func errorResponse(w http.ResponseWriter, r *http.Request, status int, err error) {
	var msg = map[string]string{
		"error": err.Error(),
	}
	response(w, r, status, &msg)
}

// middleware

func loggingMw(next httprouter.Handle) httprouter.Handle {
	if !klog.V(2).Enabled() {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		m := httpsnoop.CaptureMetricsFn(w, func(w http.ResponseWriter) {
			next(w, r, p)
		})
		klog.InfoS("access",
			"method", r.Method,
			"path", r.URL,
			"query", r.URL.RawQuery,
			"code", m.Code,
			"duration", m.Duration,
			"bytes_send", m.Written,
		)
	}
}
