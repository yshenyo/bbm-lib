# etcd service discovery and registry

### discovery

```
svr, err := discovery.NewBBService([]string{"http://127.0.0.1:2379"})
if err != nil {
	panic("new bbService error")
}

serverName := "svtTest"
//Get aLl server 
svr.ServerAllList()
//Get a list of all server named "test"
svr.ServerList(serverName)
//Randomly get a live server
svr.GetServer(serverName)
//Close
svr.Close()

```

