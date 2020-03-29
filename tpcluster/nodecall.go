package tpcluster

import (
	"github.com/weblazy/teleport"
	"lazygo/core/logx"
)

type NodeCall struct {
	tp.CallCtx
}

// Add handles addition request
func (n *NodeCall) UpdateNodeList(nodeList *[]string) (int64, *tp.Status) {
	for _, value := range *nodeList {
		sess, stat := nodeInfo.transPeer.Dial(value)
		if !stat.OK() {
			tp.Fatalf("%v", stat)
		}

		var result int
		auth := &Auth{
			Password:     nodeInfo.nodeConf.Password,
			TransAddress: nodeInfo.transAddress,
		}
		stat = sess.Call("/node_call/auth",
			auth,
			&result,
		).Status()
	}

	return 0, nil
}

func (n *NodeCall) Auth(args *Auth) (int, *tp.Status) {
	session := n.Session()
	sessionId := session.ID()
	peer := n.Peer()
	psession, ok := peer.GetSession(sessionId)
	if args.Password != nodeInfo.nodeConf.Password && ok {
		logx.Errorf("密码错误，非法链接:%s", sessionId)
		psession.Close()
		return 0, nil
	}
	nodeInfo.nodeSessions[sessionId] = session
	return 1, nil
}
