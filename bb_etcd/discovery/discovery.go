package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"bbm_lib/bb_etcd"
	"bbm_lib/utils/connect"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"go.etcd.io/etcd/clientv3"
)

type bbServiceDiscovery struct {
	cli        *clientv3.Client  //etcd client
	serverList map[string]string //map[serverName]string
	sync.RWMutex
}

func NewBBService(endpoints []string) (BBEtcdDiscoveryInterface, error) {
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
	return &bbServiceDiscovery{
		cli:        cli,
		serverList: make(map[string]string),
	}, nil
}

type BBEtcdDiscoveryInterface interface {
	NewClientService(serverName string) error
	ServerAllList() (list map[string]bb_etcd.ServerData)
	ServerList(serverName string) (list []bb_etcd.ServerData)
	GetServer(serverName string) (server bb_etcd.ServerData, err error)
	Close() error

	watchService(serverNamePrefix string) error
	setServiceList(key string, value string)
	deleteServiceList(key string)
}

func (b *bbServiceDiscovery) NewClientService(serverName string) error {
	return b.watchService(serverName)
}

func (b *bbServiceDiscovery) ServerAllList() (list map[string]bb_etcd.ServerData) {
	list = make(map[string]bb_etcd.ServerData)
	for k, v := range b.serverList {
		tmp := bb_etcd.ServerData{}
		_ = json.Unmarshal([]byte(v), &tmp)
		list[k] = tmp
	}
	return
}

func (b *bbServiceDiscovery) ServerList(serverName string) (list []bb_etcd.ServerData) {
	for k, v := range b.serverList {
		if k == serverName {
			tmp := bb_etcd.ServerData{}
			_ = json.Unmarshal([]byte(v), &tmp)
			list = append(list, tmp)
		}
	}
	return
}

func (b *bbServiceDiscovery) GetServer(serverName string) (server bb_etcd.ServerData, err error) {
	for k, v := range b.serverList {
		if k == serverName {
			tmp := bb_etcd.ServerData{}
			_ = json.Unmarshal([]byte(v), &tmp)
			if err = connect.TelnetIPPort(tmp.Host, tmp.Port); err != nil {
				log.Printf("%v", err)
				continue
			}
			return tmp, nil
		}
	}
	return server, fmt.Errorf("serverName %v not exist", serverName)
}

func (b *bbServiceDiscovery) watchService(serverName string) error {
	resp, err := b.cli.Get(context.Background(), serverName, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, ev := range resp.Kvs {
		b.setServiceList(string(ev.Key), string(ev.Value))
	}

	go b.watcher(serverName)

	return nil
}

func (b *bbServiceDiscovery) watcher(serName string) {
	rch := b.cli.Watch(context.Background(), serName, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT:
				b.setServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE:
				b.deleteServiceList(string(ev.Kv.Key))
			}
		}
	}
}

func (b *bbServiceDiscovery) setServiceList(key string, value string) {
	b.RLock()
	defer b.RUnlock()
	delete(b.serverList, key)
	b.serverList[key] = value
}

func (b *bbServiceDiscovery) deleteServiceList(key string) {
	b.RLock()
	defer b.RUnlock()
	delete(b.serverList, key)
}

func (b *bbServiceDiscovery) Close() error {
	return b.cli.Close()
}
