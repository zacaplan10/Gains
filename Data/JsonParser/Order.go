package JsonParser

// Order represents the main structure of the JSON.
type Order struct {
	Session                  string          `json:"session"`
	Duration                 string          `json:"duration"`
	OrderType                string          `json:"orderType"`
	ComplexOrderStrategyType string          `json:"complexOrderStrategyType"`
	Quantity                 float64         `json:"quantity"`
	FilledQuantity           float64         `json:"filledQuantity"`
	RemainingQuantity        float64         `json:"remainingQuantity"`
	RequestedDestination     string          `json:"requestedDestination"`
	DestinationLinkName      string          `json:"destinationLinkName"`
	Price                    float64         `json:"price"`
	OrderLegCollection       []OrderLeg      `json:"orderLegCollection"`
	OrderStrategyType        string          `json:"orderStrategyType"`
	OrderId                  int64           `json:"orderId"`
	Cancelable               bool            `json:"cancelable"`
	Editable                 bool            `json:"editable"`
	Status                   string          `json:"status"`
	EnteredTime              string          `json:"enteredTime"`
	CloseTime                string          `json:"closeTime"`
	Tag                      string          `json:"tag"`
	AccountNumber            int64           `json:"accountNumber"`
	OrderActivityCollection  []OrderActivity `json:"orderActivityCollection"`
}

// OrderLeg represents each leg of the order.
type OrderLeg struct {
	OrderLegType   string     `json:"orderLegType"`
	LegId          int        `json:"legId"`
	Instrument     Instrument `json:"instrument"`
	Instruction    string     `json:"instruction"`
	PositionEffect string     `json:"positionEffect"`
	Quantity       float64    `json:"quantity"`
	Price          float64    `json:"price"`
}

// Instrument represents the details of the instrument in the order leg.
type Instrument struct {
	AssetType    string `json:"assetType"`
	Cusip        string `json:"cusip"`
	Symbol       string `json:"symbol"`
	InstrumentId int64  `json:"instrumentId"`
}

// OrderActivity represents the activity related to the order.
type OrderActivity struct {
	ActivityType           string         `json:"activityType"`
	ActivityId             int64          `json:"activityId"`
	ExecutionType          string         `json:"executionType"`
	Quantity               float64        `json:"quantity"`
	OrderRemainingQuantity float64        `json:"orderRemainingQuantity"`
	ExecutionLegs          []ExecutionLeg `json:"executionLegs"`
}

// ExecutionLeg represents the details of each execution leg.
type ExecutionLeg struct {
	LegId             int64   `json:"legId"`
	Quantity          float64 `json:"quantity"`
	MismarkedQuantity float64 `json:"mismarkedQuantity"`
	Price             float64 `json:"price"`
	Time              string  `json:"time"`
	InstrumentId      int64   `json:"instrumentId"`
}
