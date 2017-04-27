package discovery

import (
	"bytes"
	"fmt"
	"net"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/mailgun/scroll/vulcand"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

type ServiceData struct {
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
func directLookupSRV(service, portName, network string) ([]ServiceData, error) {
	// TODO: Detect if we are on K8 or Mesos

	// Find the dns service
	addrs, err := net.LookupHost(Fqdn("kube-dns", "kube-system"))
	if err != nil {
		return nil, err
	}

	// Construct a full SRV domain request
	service = fmt.Sprintf("_%s._%s.%s", portName, network, Fqdn(service, "default"))

	// Query DNS Directly for SRV records
	c := new(dns.Client)
	c.Timeout = time.Second * 3

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(service), dns.TypeSRV)
	r, _, err := c.Exchange(m, net.JoinHostPort(addrs[0], "53"))
	if err != nil {
		return nil, err
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("Invalid answer name after SRV query for %s\n", service)
	}

	var results []ServiceData
	// Stuff must be in the answer section
	for _, a := range r.Answer {
		srv := a.(*dns.SRV)
		results = append(results, ServiceData{
			Target:   srv.Target,
			PortName: portName,
			Net:      network,
			Service:  service,
			Port:     int(srv.Port),
		})
	}
	return results, nil
}

func builtinLookupSRV(service, portName, network string) ([]ServiceData, error) {
	// TODO: FQDN the host here?
	_, records, err := net.LookupSRV(portName, network, service)
	if err != nil {
		return nil, err
	}
	var results []ServiceData
	for _, srv := range records {
		results = append(results, ServiceData{
			Target:   srv.Target,
			PortName: portName,
			Net:      network,
			Service:  service,
			Port:     int(srv.Port),
		})
	}
	return results, nil
}

// Return a list of discovered endpoints for the service requested, in the format specified
// Available format variables are `.Target`, `.Port`, `.PortName`, `.Net`, `.Service`
//
//	// "etcd" is the name of the service registered and "client" is the name of the port
// 	// when the services is registered with K8 or Mesos
//	endpoints, err := discovery.Services("etcd", "client", "tcp", "http://{{.Target}}:{{.Port}}")
//	if err != nil {
//		panic(err)
//	}
//
func Services(service, portName, net, format string) ([]string, error) {
	var err error

	tmpl, err := template.New("service").Parse(format)
	if err != nil {
		return nil, errors.Wrap(err, "while creating template for service lookup")
	}

	var records []ServiceData
	if runtime.GOOS == "darwin" {
		records, err = directLookupSRV(service, portName, net)
	} else {
		records, err = builtinLookupSRV(service, portName, net)
	}
	if err != nil {
		return nil, err
	}

	var result []string
	for _, record := range records {
		record.Target = strings.TrimRight(record.Target, ".")
		record.PortName = portName
		record.Net = net
		record.Service = service

		var buf bytes.Buffer
		if err = tmpl.Execute(&buf, record); err != nil {
			return nil, errors.Wrap(err, "while executing template")
		}
		result = append(result, buf.String())
	}
	return result, nil
}

// Return a single address for the service requested, in the format specified
func Service(host, portName, net, format string) (string, error) {
	results, err := Services(host, portName, net, format)
	if err != nil {
		return "", err
	}
	return results[0], nil
}

// Return a vulcand config object suitable for use by scroll
func VulcandConfig() vulcand.Config {
	return vulcand.Config{}
}
