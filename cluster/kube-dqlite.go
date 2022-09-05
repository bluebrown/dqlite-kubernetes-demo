package cluster

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/canonical/go-dqlite/app"
	"github.com/canonical/go-dqlite/client"
	"k8s.io/klog/v2"
)

// create the listener and dial configs using the cert found in the certDir
// the cert is expected be named tls.crt and the key tls.key
func MakeTlsConfig(certDir string) (listen, dial *tls.Config, err error) {
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

// compute the pod and cluster addresses based on the pod name and the sql port
// the pod-name is expected to have a numeric suffix (e.g. "pod-1")
// if the suffix is not -0, the cluster slice will contain exactly 1 element
// that is the pod-name with its suffix substituted for -0.
// this assumes that in any setup, a pod with the suffix -0 , exists
// i.e. a kubernetes StatefulSet
//
//	$(POD_NAME).$(SERVICE_NAME).$(NAMESPACE).svc.$(CLUSTER_DOMAIN):$(SQL_PORT)
func ComputeAddrs(pod, svc, ns, domain string, sqlPort string, useFqdn bool) (podAddr string, clusterAddrs []string) {
	var suffix string
	if useFqdn {
		// use the fully qualified domain name as described above
		suffix = fmt.Sprintf("%s.%s.svc.%s", svc, ns, domain)
	} else {
		// use only pod.service relying on the search option in /etc/resolv.conf
		suffix = fmt.Sprintf("%s", svc)
	}
	isZero := strings.HasSuffix(pod, "-0")
	if !isZero {
		zero := regexp.MustCompile(`-\d+$`).ReplaceAllString(pod, "-0")
		clusterAddrs = []string{net.JoinHostPort(fmt.Sprintf("%s.%s", zero, suffix), sqlPort)}
	}
	return net.JoinHostPort(fmt.Sprintf("%s.%s", pod, suffix), sqlPort), clusterAddrs
}

// logger function to configure dqlite with klog logs
func DqliteKlog(level client.LogLevel, format string, args ...interface{}) {
	var lvl klog.Level
	switch level {
	case client.LogError:
		lvl = 0
	case client.LogWarn:
		lvl = 1
	case client.LogInfo:
		lvl = 2
	case client.LogDebug:
		lvl = 3
	default:
		lvl = 4
	}
	if klog.V(lvl).Enabled() {
		klog.InfoS(fmt.Sprintf(format, args...), "level", level.String())
	}
}

// get an environment variable or return a default value
func GetEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
