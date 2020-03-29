package tpcluster

import (
	// "encoding/json"
	// "fmt"
	"github.com/weblazy/teleport"
	"lazygo/core/logx"
)

type (
	MasterPush struct {
		tp.PushCtx
	}
	Auth struct {
		TransAddress string //Node address ip:port
		Password     string //Password for auth when node connect on
	}
)

//Heartbeat
func (m *MasterPush) Ping(ping *string) *tp.Status {
	sessionId := m.Session().ID()
	logx.Errorf("%s:%s", sessionId, *ping)
	return nil
}

func (m *MasterPush) OnClientConnect(ping *string) *tp.Status {
	sessionId := m.Session().ID()
	logx.Errorf("%s:%s", sessionId, *ping)
	return nil
}

func (m *MasterPush) OnClientClose(ping *string) *tp.Status {
	sessionId := m.Session().ID()
	logx.Errorf("%s:%s", sessionId, *ping)
	return nil
}

//Auth the node
func (m *MasterPush) Auth(args *Auth) *tp.Status {
	session := m.Session()
	sessionId := session.ID()

	peer := m.Peer()
	psession, ok := peer.GetSession(sessionId)
	masterInfo.timer.RemoveTimer(sessionId) //Cancel timeingwheel task
	if args.Password != masterInfo.masterConf.Password && ok {
		logx.Errorf("Connect:%s,Wrong password:%s", sessionId, args.Password)
		psession.Close()
		return StatusUnauthorized
	}
	masterInfo.setSession(psession, args.TransAddress)
	masterInfo.broadcastAddresses() //Notify all node nodes that new nodes have joined
	return nil
}
