package custom

import (
	"github.com/davyxu/cellnet"
	"github.com/davyxu/cellnet/proc"
)

func init() {
	proc.RegisterProcessor("tcp.mb", func(bundle proc.ProcessorBundle, userCallback cellnet.EventCallback, args ...interface{}) {
		bundle.SetTransmitter(new(mbMessageTransmitter))
		/*
			发送消息时先调用hooker.OnOutboundEvent对要发送的消息进行拦截处理，然后调用Transmitter.OnSendMessage进行消息发送
			调用Transmitter.OnRecvMessage进行消息接收，接收调用hooker.OnInboundEvent对消息进行拦截处理，然后调用EventCallback把消息推送到逻辑层处理
		*/
		//bundle.SetHooker(new(MsgHooker)) //可以不设置
		bundle.SetCallback(proc.NewQueuedEventCallback(userCallback))
	})
	proc.RegisterProcessor("tcp.proxy", func(bundle proc.ProcessorBundle, userCallback cellnet.EventCallback, args ...interface{}) {

		bundle.SetTransmitter(new(proxyMessageTransmitter))
		//	bundle.SetHooker(new(MsgHooker))
		bundle.SetCallback(proc.NewQueuedEventCallback(userCallback))

	})
}
