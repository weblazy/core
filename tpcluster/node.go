package tpcluster

import (
	"github.com/weblazy/teleport"
	"lazygo/core/consistenthash/unsafehash"
	"lazygo/core/database/redis"
	"lazygo/core/logx"
	"lazygo/core/mapreduce"
	"lazygo/core/syncx"
	"lazygo/core/timingwheel"

	"strconv"
	"time"
)

type (
	NodeConf struct {
		RedisConf      redis.RedisConf
		RedisMaxCount  uint32
		ClientPeerConf tp.PeerConfig
		TransPeerConf  tp.PeerConfig
		TransPort      int64  //Internal communication port
		MasterAddress  string //Master address
		Password       string //Password for auth when connect to master
		PingInterval   int    //Heartbeat interval
	}

	Test struct {
		Reloadable             bool
		PingInterval           int64
		PingNotResPonseLimit   int64
		PingData               string
		CecreteKey             string
		Router                 func()
		SendToWorkerBufferSize int64
		SendToClientBufferSize int64
		nodeSessions           map[string]tp.Session
		startTime              time.Time
		// gatewayConnections map[string]string
		// businessConnections map[string]tp.Session
	}
	NodeInfo struct {
		bizRedis       *redis.Redis
		nodeConf       NodeConf
		masterSession  tp.Session
		nodeSessions   map[string]tp.CtxSession
		clientSessions map[string]tp.Session
		uidSessions    *syncx.ConcurrentDoubleMap
		groupSessions  *syncx.ConcurrentDoubleMap
		clientPeer     tp.Peer                  //External communication peer
		clientAddress  string                   //External communication address
		transPeer      tp.Peer                  //Internal communication peer
		transAddress   string                   //Internal communication address
		timer          *timingwheel.TimingWheel //Timingwheel
		startTime      time.Time
		userHashRing   *unsafehash.Consistent //UsHash ring storage userId
		groupHashRing  *unsafehash.Consistent //UsHash ring storage groupId
	}

	Message struct {
		uid  string
		path string
		data interface{}
	}
)

var (
	nodeInfo NodeInfo
)

const (
	PERSISTENCE_CONNECTION_PING_INTERVAL = 25
	redisInterval                        = 10
	redisZsortKey                        = "tpcluster_node"
)

// NewPeer creates a new peer.
func StartNode(cfg NodeConf, controllers []interface{}, globalLeftPlugin ...tp.Plugin) {
	redis := redis.NewRedis(cfg.RedisConf.Host, cfg.RedisConf.Type, cfg.RedisConf.Pass)
	timer, err := timingwheel.NewTimingWheel(time.Second, 300, func(k, v interface{}) {
		logx.Errorf("%s auth timeout", k)
		err := v.(tp.Session).Close()
		if err != nil {
			logx.Error(err)
		}
	})
	defer timer.Stop()
	if err != nil {
		logx.Fatal(err)
	}
	if cfg.PingInterval == 0 {
		cfg.PingInterval = 10
	}
	nodeInfo = NodeInfo{
		nodeConf:       cfg,
		nodeSessions:   make(map[string]tp.CtxSession),
		clientSessions: make(map[string]tp.Session),
		bizRedis:       redis,
		uidSessions:    syncx.NewConcurrentDoubleMap(32),
		groupSessions:  syncx.NewConcurrentDoubleMap(32),
		startTime:      time.Now(),
		timer:          timer,
		userHashRing:   unsafehash.NewConsistent(cfg.RedisMaxCount),
		groupHashRing:  unsafehash.NewConsistent(cfg.RedisMaxCount),
	}
	port := strconv.FormatInt(int64(cfg.ClientPeerConf.ListenPort), 10)
	nodeInfo.clientAddress = cfg.ClientPeerConf.LocalIP + ":" + port
	globalLeftPlugin = append(globalLeftPlugin, new(postDisconnectPlugin))
	nodeInfo.transPeer = tp.NewPeer(cfg.TransPeerConf)
	nodeInfo.clientPeer = tp.NewPeer(cfg.ClientPeerConf, globalLeftPlugin...)
	for _, value := range controllers {
		nodeInfo.clientPeer.RoutePush(value)
	}
	nodeInfo.transPeer.RouteCall(new(NodeCall))
	nodeInfo.transPeer.RoutePush(new(NodePush))
	go nodeInfo.transPeer.ListenAndServe()
	SendPing()
	UpdateRedis()
	go ConnectToMaster(cfg)
	nodeInfo.clientPeer.ListenAndServe()
}

//Connect to master
func ConnectToMaster(cfg NodeConf) {
	sess, stat := nodeInfo.transPeer.Dial(cfg.MasterAddress)
	if !stat.OK() {
		tp.Fatalf("%v", stat)
	}
	nodeInfo.masterSession = sess
	port := strconv.FormatInt(int64(cfg.TransPeerConf.ListenPort), 10)
	nodeInfo.transAddress = cfg.TransPeerConf.LocalIP + ":" + port
	var result int
	auth := &Auth{
		Password:     nodeInfo.nodeConf.Password,
		TransAddress: nodeInfo.transAddress,
	}
	stat = sess.Call("/master_call/auth",
		auth,
		&result,
	).Status()
}

func OnClientClose(ping *string) {

}

//Determine if a uid is online
func IsOnline(uid string) bool {
	now := time.Now().Unix()
	node := nodeInfo.userHashRing.Get(uid)
	addrMap, err := node.Extra.(*redis.Redis).Hgetall(userPrefix + uid)
	if err == nil {
		return false
	}
	for _, value := range addrMap {
		old, _ := strconv.ParseInt(value, 10, 64)
		if now < old {
			return true
		}
	}
	return false
}

//Get online users in the group
func GroupOnline(gid string) []string {
	now := time.Now().Unix()
	node := nodeInfo.groupHashRing.Get(gid)
	uids := make([]string, 0)
	addrMap, err := node.Extra.(*redis.Redis).Hgetall(groupPrefix + gid)
	if err == nil {
		return uids
	}
	for key, value := range addrMap {
		old, _ := strconv.ParseInt(value, 10, 64)
		if now < old {
			uids = append(uids, key)
		}
	}
	return uids
}

//Get session from PreCtx
func GetSession(context tp.PreCtx) tp.Session {
	sid := context.Session().ID()
	session, _ := context.Peer().GetSession(sid)
	return session
}

//Get bind uid with session
func BindUid(uid string, context tp.PreCtx) error {
	now := time.Now().Unix()
	node := nodeInfo.userHashRing.Get(uid)
	err := node.Extra.(*redis.Redis).Hset(userPrefix+uid, nodeInfo.transAddress, strconv.FormatInt(now, 10))
	if err != nil {
		return err
	}
	sid := context.Session().ID()
	session, _ := context.Peer().GetSession(sid)
	nodeInfo.uidSessions.StoreWithPlugin(uid, sid, session, func() {
		oldUid := session.CasUid(uid)
		if oldUid != "" && oldUid != uid {
			nodeInfo.uidSessions.DeleteWithoutLock(oldUid, sid)
		}
	})
	return nil
}

//Send message to a uid
func SendToUid(uid string, path string, req interface{}) (int, *tp.Status) {
	now := time.Now().Unix()
	node := nodeInfo.userHashRing.Get(uid)
	ipMap, err := node.Extra.(*redis.Redis).Hgetall(userPrefix + uid)
	if err != nil {
		mapreduce.MapVoid(func(source chan<- interface{}) {
			for key, value := range ipMap {
				expir, _ := strconv.ParseInt(value, 10, 64)
				if now > expir {
					source <- key
				}
			}
		}, func(item interface{}) {
			sid := item.(string)
			session, ok := nodeInfo.transPeer.GetSession(sid)
			if ok {
				session.Push(
					"/node_push/send_to_uid",
					&Message{
						uid:  uid,
						path: path,
						data: req,
					},
				)
			}

		})
	}
	return 0, nil
}

//Join a group
func JoinGroup(gid string, session tp.Session) (int, *tp.Status) {
	sid := session.ID()
	nodeInfo.groupSessions.Store(gid, sid, session)
	return 0, nil
}

//Leave group
func LeaveGroup(gid string, session tp.Session) (int, *tp.Status) {
	sid := session.ID()
	nodeInfo.groupSessions.Delete(gid, sid)
	return 0, nil
}

//Send message to group
func SendToGroup(gid string, path string, req interface{}) (int, *tp.Status) {
	sessionMap, ok := nodeInfo.groupSessions.LoadMap(gid)
	if ok {
		mapreduce.MapVoid(func(source chan<- interface{}) {
			for _, session := range sessionMap {
				source <- session
			}
		}, func(item interface{}) {
			session := item.(tp.Session)
			session.Push(
				path,
				req,
			)
		})
	}
	return 0, nil
}

func JoinGroupNew(gid, uid string) (int, *tp.Status) {
	now := time.Now().Unix()
	node := nodeInfo.groupHashRing.Get(gid)
	err := node.Extra.(*redis.Redis).Hset(groupPrefix+gid, uid, strconv.FormatInt(now, 10))
	if err != nil {
		logx.Fatal(err)
	}
	return 0, nil
}

func LeaveGroupNew(gid, uid string) (int, *tp.Status) {
	node := nodeInfo.groupHashRing.Get(gid)
	_, err := node.Extra.(*redis.Redis).Hdel(groupPrefix+gid, uid)
	if err != nil {
		logx.Fatal(err)
	}
	return 0, nil
}

func SendToGroupNew(gid string, path string, req interface{}) (int, *tp.Status) {
	uids := GroupOnline(gid)
	mapreduce.MapVoid(func(source chan<- interface{}) {
		for _, uid := range uids {
			sidMap, ok := nodeInfo.uidSessions.LoadMap(uid)
			if ok {
				for _, value := range sidMap {
					source <- value
				}
			}

		}
	}, func(item interface{}) {
		session := item.(tp.Session)
		session.Push(
			path,
			req,
		)
	})
	return 0, nil
}

//Heartbeat
func SendPing() {
	go func() {
		for {
			time.Sleep(time.Duration(nodeInfo.nodeConf.PingInterval) * time.Second)
			nodeInfo.masterSession.Push(
				"/master_push/ping",
				"ping",
			)
			nodeInfo.transPeer.RangeSession(func(sess tp.Session) bool {
				if sess != nodeInfo.masterSession {
					sess.Push(
						"/node_push/ping",
						"ping",
					)
				}

				return true
			})
		}
	}()
}

//Update clients num
func UpdateRedis() {
	go func() {
		for {
			time.Sleep(redisInterval * time.Second)
			nodeInfo.bizRedis.Zadd(redisZsortKey, int64(nodeInfo.clientPeer.CountSession()), nodeInfo.clientAddress)
		}
	}()
}
