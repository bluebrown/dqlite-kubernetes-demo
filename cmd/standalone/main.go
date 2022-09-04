package main

import (
	"context"
	"database/sql"
	"flag"
	"net/http"

	"github.com/bluebrown/dqlite-kubernetes-demo/crud"
	"github.com/bluebrown/dqlite-kubernetes-demo/model"
	_ "github.com/mattn/go-sqlite3"
	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)
	dataSource := flag.String("db", ":memory:", "the sqlite database")
	flag.Parse()

	if err := run(context.TODO(), *dataSource, ":8080"); err != http.ErrServerClosed {
		klog.ErrorS(err, "could not start app")
	}
}

func run(ctx context.Context, dataSource, addr string) error {
	db, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		return err
	}
	if err := model.Migrate(ctx, db); err != nil {
		return err
	}
	router := crud.New(ctx, db)

	klog.InfoS("startup", "address", addr, "db", dataSource)
	return http.ListenAndServe(addr, router)
}
