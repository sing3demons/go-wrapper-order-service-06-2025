package logAction

const (
	Consuming    = "[CONSUMING]"
	Producing    = "[PRODUCING]"
	AppLogic     = "[APP_LOGIC]"
	HttpRequest  = "[HTTP_REQUEST]"
	HttpResponse = "[HTTP_RESPONSE]"
	DbRequest    = "[DB_REQUEST]"
	DbResponse   = "[DB_RESPONSE]"
	Exception    = "[EXCEPTION]"
	Inbound      = "[INBOUND]"
	Outbound     = "[OUTBOUND]"
	System       = "[SYSTEM]"
	Produced     = "[PRODUCED]"
)

type LoggerAction struct {
	Action            string `json:"action"`
	ActionDescription string `json:"actionDescription"`
	SubAction         string `json:"subAction,omitempty"`
}

func CONSUMING(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            Consuming,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func PRODUCING(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            Producing,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func INBOUND(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            Inbound,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func OUTBOUND(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            Outbound,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func APP_LOGIC(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            AppLogic,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func HTTP_REQUEST(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            HttpRequest,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func HTTP_RESPONSE(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            HttpResponse,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func DB_REQUEST(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            DbRequest,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func DB_RESPONSE(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            DbResponse,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func EXCEPTION(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            Exception,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func SYSTEM(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            System,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func PRODUCED(desc string, subAction string) LoggerAction {
	return LoggerAction{
		Action:            Produced,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}
