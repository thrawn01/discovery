package discovery

import (
	"github.com/mailgun/scroll/vulcand"
	"github.com/miekg/dns"
	"net"
	"fmt"
	"text/template"
	"github.com/pkg/errors"
	"bytes"
	"time"
	"runtime"
)

type ServiceData struct {
	Target string
	Port   int
}

// If you query for SRV records on OSX golang uses the golang version of
// dns for lookup (See https://golang.org/pkg/net/#hdr-Name_Resolution)
//
// The golang implementation in turn by passes the /etc/resolver system
// on OSX which kubegun/minikube uses for dns resolution.
func directLookupSRV(host string) ([]ServiceData, error) {
	// Find the dns service
	addrs, err := net.LookupHost("kube-dns.kube-system.svc.cluster.local")
	if err != nil {
		return nil, err
	}

	// Query DNS Directly for SRV records
	c := new(dns.Client)
	c.Timeout = time.Second * 3

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeSRV)
	r, _, err := c.Exchange(m, net.JoinHostPort(addrs[0], "53"))
	if err != nil {
		return nil, err
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("Invalid answer name after SRV query for %s\n", host)
	}

	var results []ServiceData
	// Stuff must be in the answer section
	for _, a := range r.Answer {
		srv := a.(*dns.SRV)
		results = append(results, ServiceData{Target: srv.Target, Port: int(srv.Port)})
		//fmt.Printf("Port %d\n", srv.Port)
		//fmt.Printf("Target %s\n", srv.Target)
	}
	return results, nil
}

func builtinLookupSRV(host string) ([]ServiceData, error) {
	_, records, err := net.LookupSRV("client", "tcp", host)
	if err != nil {
		return nil, err
	}
	var results []ServiceData
	for _, srv := range records {
		results = append(results, ServiceData{Target: srv.Target, Port: int(srv.Port)})
	}
	return results, nil
}

// Return a list of addresses for the service requested, in the format specified
func Services(host string, format string) ([]string, error) {
	var err error

	tmpl, err := template.New("service").Parse(format)
	if err != nil {
		return nil, errors.Wrap(err, "while creating template for service lookup")
	}

	// Construct a full SRV domain
	host = fmt.Sprintf("_client._tcp.%s", host)

	var records []ServiceData
	if runtime.GOOS == "darwin" {
		records, err = directLookupSRV(host)
	} else {
		records, err = builtinLookupSRV(host)
	}
	if err != nil {
		return nil, err
	}

	var result []string
	for _, record := range records {
		var buf bytes.Buffer
		if err = tmpl.Execute(&buf, record); err != nil {
			return nil, errors.Wrap(err, "while executing template")
		}
		result = append(result, buf.String())
	}
	return result, nil
}

// Return a single address for the service requested, in the format specified
func Service(host string, format string) (string, error) {
	results, err := Services(host, format)
	if err != nil {
		return "", err
	}
	return results[0], nil
}

// Return a vulcand config object suitable for use by scroll
func VulcandConfig() vulcand.Config {
	return vulcand.Config{}
}
