package commonlog

type LoggerActionEnum string

const (
	CONSUMING     string = "[CONSUMING]"
	PRODUCING     string = "[PRODUCING]"
	APP_LOGIC     string = "[APP_LOGIC]"
	HTTP_REQUEST  string = "[HTTP_REQUEST]"
	HTTP_RESPONSE string = "[HTTP_RESPONSE]"
	DB_REQUEST    string = "[DB_REQUEST]"
	DB_RESPONSE   string = "[DB_RESPONSE]"
	EXCEPTION     string = "[EXCEPTION]"
	INBOUND       string = "[INBOUND]"
	OUTBOUND      string = "[OUTBOUND]"
	SYSTEM        string = "[SYSTEM]"
	PRODUCED      string = "[PRODUCED]"
)

type DBActionEnum string

const (
	DB_CREATE DBActionEnum = "CREATE"
	DB_READ   DBActionEnum = "READ"
	DB_UPDATE DBActionEnum = "UPDATE"
	DB_DELETE DBActionEnum = "DELETE"
	DB_NONE   DBActionEnum = "NONE"
)

type SubActionEnum string

const (
	SUB_CREATE SubActionEnum = "CREATE"
	SUB_READ   SubActionEnum = "READ"
	SUB_UPDATE SubActionEnum = "UPDATE"
	SUB_DELETE SubActionEnum = "DELETE"
	SUB_NONE   SubActionEnum = "NONE"
)

type ILoggerActionData struct {
	Action            string `json:"action"`
	ActionDescription string `json:"actionDescription"`
	SubAction         string `json:"subAction,omitempty"`
}

type LoggerAction struct{}

func (LoggerAction) CONSUMING(desc string, subAction string) ILoggerActionData {
	return ILoggerActionData{
		Action:            CONSUMING,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func (LoggerAction) PRODUCING(desc string, subAction string) ILoggerActionData {
	return ILoggerActionData{
		Action:            PRODUCING,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func (LoggerAction) INBOUND(desc string, subAction string) ILoggerActionData {
	return ILoggerActionData{
		Action:            INBOUND,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func (LoggerAction) OUTBOUND(desc string, subAction string) ILoggerActionData {
	return ILoggerActionData{
		Action:            OUTBOUND,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func (LoggerAction) APP_LOGIC(desc string, subAction string) ILoggerActionData {
	return ILoggerActionData{
		Action:            APP_LOGIC,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func (LoggerAction) HTTP_REQUEST(desc string, subAction string) ILoggerActionData {
	return ILoggerActionData{
		Action:            HTTP_REQUEST,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func (LoggerAction) HTTP_RESPONSE(desc string, subAction string) ILoggerActionData {
	return ILoggerActionData{
		Action:            HTTP_RESPONSE,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func (LoggerAction) DB_REQUEST(operation DBActionEnum, desc string) ILoggerActionData {
	return ILoggerActionData{
		Action:            DB_REQUEST,
		ActionDescription: desc,
		SubAction:         string(operation),
	}
}

func (LoggerAction) DB_RESPONSE(operation DBActionEnum, desc string) ILoggerActionData {
	return ILoggerActionData{
		Action:            DB_RESPONSE,
		ActionDescription: desc,
		SubAction:         string(operation),
	}
}

func (LoggerAction) EXCEPTION(desc string, subAction string) ILoggerActionData {
	return ILoggerActionData{
		Action:            EXCEPTION,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func (LoggerAction) SYSTEM(desc string, subAction string) ILoggerActionData {
	return ILoggerActionData{
		Action:            SYSTEM,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}

func (LoggerAction) PRODUCED(desc string, subAction string) ILoggerActionData {
	return ILoggerActionData{
		Action:            PRODUCED,
		ActionDescription: desc,
		SubAction:         subAction,
	}
}
