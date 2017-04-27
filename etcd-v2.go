package discovery

import (
	"context"
	"time"

	etcdv2 "github.com/coreos/etcd/client"
	"github.com/pkg/errors"
)

type CancelFunc func()

type EtcdConfig interface {
	Get(string) (string, error)
	String(string, *string, string) error
	Watch(string, func(string, []byte) error) (error, CancelFunc)
}

type EtcdV2Config struct {
	api     etcdv2.KeysAPI
	timeout time.Duration
}

func NewEtcdV2Config(conf *etcdv2.Config) (EtcdConfig, error) {
	// Get our etcd endpoints
	endpoints, err := FormatServices("etcd", "client", "tcp", "http://{{.Target}}:{{.Port}}")
	if err != nil {
		return nil, err
	}

	if conf == nil {
		conf = &etcdv2.Config{
			Endpoints:               endpoints,
			HeaderTimeoutPerRequest: time.Second * 5,
		}
	} else {
		conf.Endpoints = endpoints
	}

	client, err := etcdv2.New(*conf)
	if err != nil {
		return nil, errors.Wrap(err, "while instanciating etcd2 client")
	}

	return &EtcdV2Config{
		api:     etcdv2.NewKeysAPI(client),
		timeout: conf.HeaderTimeoutPerRequest,
	}, nil
}

func (s *EtcdV2Config) Get(key string) (string, error) {
	ctx, _ := context.WithTimeout(context.Background(), s.timeout)
	resp, err := s.api.Get(ctx, key, nil)
	if err != nil {
		return "", errors.Wrapf(err, "during etcd GET on '%s'", key)
	}
	return resp.Node.Value, nil
}

func (s *EtcdV2Config) String(key string, dest *string, defValue string) error {
	var value string
	var err error

	if value, err = s.Get(key); err != nil {
		*dest = defValue
		return err
	}
	*dest = value
	return nil
}

func (s *EtcdV2Config) Watch(prefix string, callBack func(string, []byte) error) (error, CancelFunc) {
	// Not Implemented
	return nil, nil
}
