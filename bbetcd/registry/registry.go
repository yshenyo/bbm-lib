package registry

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zjbobingtech/bbm-lib/bbetcd"
	"github.com/zjbobingtech/bbm-lib/utils/connect"
	"go.etcd.io/etcd/clientv3"
)

type bbServiceRegistry struct {
	Cli     *clientv3.Client //etcd client
	leaseID clientv3.LeaseID
}

func NewBBService(endpoints []string) (BBEtcdRegistryInterface, error) {
	for _, address := range endpoints {
		if err := connect.TelnetAddress(address); err != nil {
			return nil, err
		}
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &bbServiceRegistry{
		Cli: cli,
	}, nil
}

type BBEtcdRegistryInterface interface {
	Register(serverName, serverHost string, serverPort int, f ...func()) error
	Revoke() error
}

func (b *bbServiceRegistry) Register(serverName, serverHost string, serverPort int, f ...func()) error {
	lease, err := b.Cli.Grant(context.Background(), 15)
	if err != nil {
		return err
	}
	b.leaseID = lease.ID
	key := bbetcd.GetEtcdKey(serverName, serverHost, serverPort)
	serverData := bbetcd.ServerData{
		ServerName: serverName,
		Host:       serverHost,
		Port:       serverPort,
	}
	serverDataByte, err := json.Marshal(serverData)
	if err != nil {
		return err
	}
	value := string(serverDataByte[:])

	_, err = b.Cli.Put(context.Background(), key, value, clientv3.WithLease(lease.ID))
	if err != nil {
		return err
	}

	keepaliveChan, err := b.Cli.KeepAlive(context.Background(), lease.ID)
	if err != nil {
		return err
	}

	go func() {
		for ka := range keepaliveChan {
			if ka == nil {
				// todo you can add some function for check if down
				for _, v := range f {
					v()
				}
			}
			//fmt.Println("renew:", ka.ID, time.Now().Format("2006-01-02 15:04:05"))
		}
	}()
	return nil
}

func (b *bbServiceRegistry) Revoke() error {
	if _, err := b.Cli.Revoke(context.Background(), b.leaseID); err != nil {
		return err
	}
	return nil
}
