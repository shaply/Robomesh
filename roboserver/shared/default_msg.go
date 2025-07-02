package shared

func (msg *DefaultMsg) GetMsg() string {
	return msg.Msg
}

func (msg *DefaultMsg) GetPayload() any {
	return msg.Payload
}

func (msg *DefaultMsg) GetSource() string {
	return msg.Source
}

func (msg *DefaultMsg) GetReplyChan() chan any {
	return msg.ReplyChan
}
