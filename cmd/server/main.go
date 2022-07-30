package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bluebrown/kubernetes-dqlite-example/entrysvc"
	"github.com/canonical/go-dqlite/app"
	"github.com/canonical/go-dqlite/client"
	"golang.org/x/sys/unix"
)

var (
	certDir  = "./certs"
	dataDir  = "./data"
	dbName   = "test"
	httpPort = "8080"
	sqlPort  = "9000"
)

var (
	k8sPodName       string
	k8sServiceName   string
	k8sNamespace     string
	K8sClusterDomain string = "cluster.local"
)

func main() {
	ctx := contextWithSignal(context.Background(), unix.SIGPWR, unix.SIGINT, unix.SIGQUIT, unix.SIGTERM)
	if err := run(ctx); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

func dqliteLogger(level client.LogLevel, format string, args ...interface{}) {
	if level <= client.LogInfo {
		return
	}
	log.Printf(fmt.Sprintf("%s: %s\n", level.String(), format), args...)
}

func readEnv() error {
	// get basic values from environment
	certDir = getEnv("CERT_DIR", certDir)
	dataDir = getEnv("DATA_DIR", dataDir)
	dbName = getEnv("DB_NAME", dbName)
	httpPort = getEnv("HTTP_PORT", httpPort)
	sqlPort = getEnv("SQL_PORT", sqlPort)
	// get kubernetes dns values from environment
	k8sPodName = getEnv("K8S_POD_NAME", k8sPodName)
	if k8sPodName == "" {
		return fmt.Errorf("K8S_POD_NAME is not set")
	}
	k8sServiceName = getEnv("K8S_SERVICE_NAME", k8sServiceName)
	if k8sServiceName == "" {
		return fmt.Errorf("K8S_SERVICE_NAME is not set")
	}
	k8sNamespace = getEnv("K8S_NAMESPACE", k8sNamespace)
	if k8sNamespace == "" {
		return fmt.Errorf("K8S_NAMESPACE is not set")
	}
	K8sClusterDomain = getEnv("K8S_CLUSTER_DOMAIN", K8sClusterDomain)
	return nil
}

func run(ctx context.Context) error {
	if err := readEnv(); err != nil {
		return err
	}

	// get the dns based pod and cluster addresses
	podAddr, clusterAddrs := computeAddrs(k8sPodName, k8sServiceName, k8sNamespace, K8sClusterDomain, sqlPort)

	// create the data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// create the TLS config
	listen, dial, err := makeTlsConfig(certDir)
	if err != nil {
		return err
	}

	// create a new dqlite app
	dqlite, err := app.New(
		dataDir,
		app.WithLogFunc(dqliteLogger),
		app.WithTLS(listen, dial),
		app.WithAddress(podAddr),
		app.WithCluster(clusterAddrs),
	)
	if err != nil {
		return err
	}

	// wait until the app is ready
	if err := func() error {
		toCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		return dqlite.Ready(toCtx)
	}(); err != nil {
		return err
	}

	// defer the clean closing of the app
	defer func() {
		log.Println("closing dqlite app")
		if err := dqlite.Handover(ctx); err != nil {
			log.Printf("error: %v", err)
		}
		if err := dqlite.Close(); err != nil {
			log.Printf("error: %v", err)
		}
	}()

	// open the database
	db, err := dqlite.Open(ctx, dbName)
	if err != nil {
		return err
	}

	// defer closing the database
	defer func() {
		log.Println("closing database")
		if err := db.Close(); err != nil {
			log.Printf("error closing db: %v", err)
		}
	}()

	// check if the database is reachable
	if err := db.PingContext(ctx); err != nil {
		return err
	}

	// create a new entry service
	repo := entrysvc.NewEntryRepository(db)
	svc := entrysvc.NewEntryService(repo)

	// apply migrations, if its the zero pod
	if strings.HasSuffix(k8sPodName, "-0") {
		if err := repo.Migrate(ctx); err != nil {
			return err
		}
	}

	// create a new http server
	return runGracefully(ctx, &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", httpPort),
		Handler: entrysvc.NewServer(k8sPodName, svc),
	})

}

// get an environment variable or return a default value
func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

// compute the pod and cluster addresses based on the pod name and the sql port
// the podname is expected to have a numeric suffix (e.g. "pod-1")
// if the suffix is not -0, the cluster slice will contain exactly 1 element
// that is the podname with its suffix substituted for -0.
// this assumes that in any setup, a pod with the suffix -0 , exists
// i.e. a kubernetes statefulset
//  $(K8S_POD_NAME).$(K8S_SERVICE_NAME).$(K8S_NAMESPACE).svc.$(K8S_CLUSTER_DOMAIN):$(SQL_PORT)
func computeAddrs(pod, svc, ns, domain string, sqlPort string) (podAddr string, clusterAddrs []string) {
	suffix := fmt.Sprintf("%s.%s.svc.%s", svc, ns, domain)
	isZero := strings.HasSuffix(pod, "-0")
	if !isZero {
		zero := regexp.MustCompile(`-\d+$`).ReplaceAllString(pod, "-0")
		clusterAddrs = []string{net.JoinHostPort(fmt.Sprintf("%s.%s", zero, suffix), sqlPort)}
	}
	return net.JoinHostPort(fmt.Sprintf("%s.%s", pod, suffix), sqlPort), clusterAddrs
}

// create the listener and dial configs using the cert found in the certDir
// the cert is expected be named tls.crt and the key tls.key
func makeTlsConfig(certDir string) (listen, dial *tls.Config, err error) {
	certPath := filepath.Join(certDir, "tls.crt")
	keyPath := filepath.Join(certDir, "tls.key")
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, nil, err
	}
	data, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(data)
	listen, dial = app.SimpleTLSConfig(cert, pool)
	return listen, dial, nil
}

// run the server and shut down gracefully when the context is done
func runGracefully(ctx context.Context, server *http.Server) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	go func() {
		<-ctx.Done()
		log.Println("stopping server")
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(timeout); err != nil {
			log.Printf("error: %v", err)
		}
	}()
	log.Println("starting server")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func contextWithSignal(ctx context.Context, sig ...os.Signal) context.Context {
	cancelableContext, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, sig...)
		select {
		case <-sigs:
		case <-ctx.Done():
		}
	}()

	return cancelableContext
}
