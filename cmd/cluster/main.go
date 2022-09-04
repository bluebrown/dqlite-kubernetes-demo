package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/canonical/go-dqlite/app"
	"k8s.io/klog/v2"

	"github.com/bluebrown/dqlite-kubernetes-demo/cluster"
	"github.com/bluebrown/dqlite-kubernetes-demo/crud"
	"github.com/bluebrown/dqlite-kubernetes-demo/model"
)

var (
	certDir  = "./certs"
	dataDir  = "./data"
	dbName   = "test"
	httpPort = "8080"
	sqlPort  = "9000"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	ctx := context.TODO()
	if err := run(ctx); err != nil {
		klog.ErrorS(err, "failed to run program")
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// get basic values from environment
	certDir = cluster.GetEnv("CERT_DIR", certDir)
	dataDir = cluster.GetEnv("DATA_DIR", dataDir)
	dbName = cluster.GetEnv("DB_NAME", dbName)
	httpPort = cluster.GetEnv("HTTP_PORT", httpPort)
	sqlPort = cluster.GetEnv("SQL_PORT", sqlPort)

	// get kubernetes dns info
	podName := cluster.GetEnv("POD_NAME", "")
	svcName := cluster.GetEnv("SERVICE_NAME", "")
	nsName := cluster.GetEnv("NAMESPACE", "default")
	domain := cluster.GetEnv("CLUSTER_DOMAIN", "cluster.local")

	// create the TLS config
	listen, dial, err := cluster.MakeTlsConfig(certDir)
	if err != nil {
		return err
	}

	// set the data path based on datadir
	dataPath := path.Join(dataDir, podName)

	// create the data dataPath if it doesn't exist
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return err
	}

	// get the addresses to use
	podAddr, clusterAddrs := cluster.ComputeAddrs(podName, svcName, nsName, domain, sqlPort)

	// create a new dqlite app
	dqlite, err := app.New(
		dataPath,
		app.WithLogFunc(cluster.DqliteKlog),
		app.WithTLS(listen, dial),
		app.WithAddress(podAddr),
		app.WithCluster(clusterAddrs),
	)
	if err != nil {
		return err
	}

	// wait for the app to become ready
	if err := dqlite.Ready(ctx); err != nil {
		return err
	}

	// defer the clean closing of the app
	defer func() {
		klog.V(2).InfoS("shutdown", "msg", "dqlite handover")
		if err := dqlite.Handover(ctx); err != nil {
			klog.ErrorS(err, "failed to handover")
		}
		if err := dqlite.Close(); err != nil {
			klog.ErrorS(err, "failed to release resources")
		}
	}()

	// open the database
	db, err := dqlite.Open(ctx, dbName)
	if err != nil {
		return err
	}

	// defer closing the database
	defer func() {
		klog.V(2).InfoS("shutdown", "msg", "closing database")
		if err := db.Close(); err != nil {
			klog.ErrorS(err, "failed to close db")
		}
	}()

	// check if the database is reachable
	if err := db.PingContext(ctx); err != nil {
		return err
	}

	// apply migrations, if its the zero pod
	if strings.HasSuffix(podName, "-0") {
		if err := model.Migrate(ctx, db); err != nil {
			return err
		}
	}

	// create the microservice and use its http handler
	httpHandler := crud.New(ctx, db)

	// create a new server
	klog.InfoS("startup", "httpPort", httpPort, "sqlPort", sqlPort)
	if err := http.ListenAndServe(":"+httpPort, httpHandler); err != http.ErrServerClosed {
		return err
	}

	return nil
}
