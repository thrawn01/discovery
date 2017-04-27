package discovery

import (
	"bytes"
	"fmt"
	"net"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

type ServiceInfo struct {
	Service  string
	Net      string
	Target   string
	PortName string
	Port     int
}

func Fqdn(service, namespace string) string {
	// TODO: Detect if we are on K8 or Mesos
	return fmt.Sprintf("%s.%s.svc.cluster.local.", service, namespace)
}

// If you query for SRV records on OSX golang uses the golang version of
// dns for lookup (See https://golang.org/pkg/net/#hdr-Name_Resolution)
//
// The golang implementation in turn by passes the /etc/resolver system
// on OSX which kubegun/minikube uses for dns resolution.
func directLookupSRV(service, portName, network string) ([]ServiceInfo, error) {
	// TODO: Detect if we are on K8 or Mesos

	// Find the dns service
	addrs, err := net.LookupHost(Fqdn("kube-dns", "kube-system"))
	if err != nil {
		return nil, err
	}

	// Construct a full SRV domain request
	fqdn := fmt.Sprintf("_%s._%s.%s", portName, network, Fqdn(service, "default"))

	// Query DNS Directly for SRV records
	c := new(dns.Client)
	c.Timeout = time.Second * 3

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(fqdn), dns.TypeSRV)
	r, _, err := c.Exchange(m, net.JoinHostPort(addrs[0], "53"))
	if err != nil {
		return nil, err
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("Unknown service '%s'; SRV query for %s failed\n", service, fqdn)
	}

	var results []ServiceInfo
	// Stuff must be in the answer section
	for _, a := range r.Answer {
		srv := a.(*dns.SRV)
		results = append(results, ServiceInfo{
			Target:   strings.TrimRight(srv.Target, "."),
			PortName: portName,
			Net:      network,
			Service:  service,
			Port:     int(srv.Port),
		})
	}
	return results, nil
}

// Uses the builtin golang net.LookupSRV on systems that have their
// /etc/resolv.conf configured correctly
func builtinLookupSRV(service, portName, network string) ([]ServiceInfo, error) {
	// TODO: FQDN the host here?
	_, records, err := net.LookupSRV(portName, network, service)
	if err != nil {
		return nil, err
	}
	var results []ServiceInfo
	for _, record := range records {
		results = append(results, ServiceInfo{
			Target:   strings.TrimRight(record.Target, "."),
			PortName: portName,
			Net:      network,
			Service:  service,
			Port:     int(record.Port),
		})
	}
	return results, nil
}

// Return a list of discovered endpoints for the service requested,
//
//	// "etcd" is the name of the service registered and "client" is the name of the port
// 	// when the services is registered with K8 or Mesos
//	endpoints, err := discovery.Services("etcd", "client", "tcp")
//	if err != nil {
//		panic(err)
//	}
//	fmt.Printf("Target: %s\n", endpoints[0].Target)
//
func Services(service, portName, net string) ([]ServiceInfo, error) {
	// TODO: Cache the results

	if runtime.GOOS == "darwin" {
		return directLookupSRV(service, portName, net)
	} else {
		return builtinLookupSRV(service, portName, net)
	}
}

// Return a list of services in the format specified
func FormatServices(service, portName, net, format string) ([]string, error) {
	var results []string

	services, err := Services(service, portName, net)
	if err != nil {
		return results, err
	}
	return Format(services, format)
}

// Return a single service in the format specified
func FormatService(service, portName, net, format string) (string, error) {
	results, err := FormatServices(service, portName, net, format)
	if err != nil {
		return "", err
	}
	return results[0], nil
}

// Return a single address for the service requested, in the format specified
func Service(host, portName, net string) (ServiceInfo, error) {
	results, err := Services(host, portName, net)
	if err != nil {
		return ServiceInfo{}, err
	}
	return results[0], nil
}

// Format ServiceInfo in the specified format; available variables
// are `.Target`, `.Port`, `.PortName`, `.Net`, `.Service`
func Format(services []ServiceInfo, format string) ([]string, error) {
	var results []string

	tmpl, err := template.New("service").Parse(format)
	if err != nil {
		return nil, errors.Wrap(err, "while creating template for service lookup")
	}

	for _, service := range services {
		var buf bytes.Buffer
		if err = tmpl.Execute(&buf, service); err != nil {
			return nil, errors.Wrap(err, "while executing template")
		}
		results = append(results, buf.String())
	}
	return results, nil
}
