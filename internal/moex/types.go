package moex

type Coupon struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type Cashflow struct {
	Date    string  `json:"date"`
	Amount  float64 `json:"amount"`
	Kind    string  `json:"kind"`
	Taxable bool    `json:"taxable"`
}

type MarketUniverse []Bond

type Bond struct {
	ISIN                 string     `json:"isin"`
	SECID                *string    `json:"secid"`
	Ticker               *string    `json:"ticker"`
	Name                 *string    `json:"name"`
	ShortName            *string    `json:"short_name"`
	LatinName            *string    `json:"latin_name"`
	Type                 *string    `json:"type"`
	TypeName             *string    `json:"type_name"`
	BondType             *string    `json:"bond_type"`
	BondSubtype          *string    `json:"bond_subtype"`
	InstrumentGroup      *string    `json:"instrument_group"`
	IssuerID             *int       `json:"issuer_id"`
	RegNumber            *string    `json:"reg_number"`
	IssueDate            *string    `json:"issue_date"`
	StartDateMOEX        *string    `json:"start_date_moex"`
	RegistryDate         *string    `json:"registry_date"`
	MaturityDate         *string    `json:"maturity_date"`
	DaysToRedemption     *int       `json:"days_to_redemption"`
	FaceValue            *float64   `json:"face_value"`
	InitialFaceValue     *float64   `json:"initial_face_value"`
	FaceUnit             *string    `json:"face_unit"`
	Currency             *string    `json:"currency"`
	IssueSize            *float64   `json:"issue_size"`
	CouponFrequency      *int       `json:"coupon_frequency"`
	CouponPercent        *float64   `json:"coupon_percent"`
	CouponValue          *float64   `json:"coupon_value"`
	CouponType           *string    `json:"coupon_type"`
	NextCouponDate       *string    `json:"next_coupon_date"`
	ListLevel            *int       `json:"list_level"`
	IsQualifiedInvestors *bool      `json:"is_qualified_investors"`
	HasProspectus        *bool      `json:"has_prospectus"`
	HasDefault           *bool      `json:"has_default"`
	HasTechnicalDefault  *bool      `json:"has_technical_default"`
	Price                *float64   `json:"price"`
	YieldToMaturity      *float64   `json:"yield_to_maturity"`
	Duration             *float64   `json:"duration"`
	AccruedInterest      *float64   `json:"accrued_interest"`
	ValueToday           *float64   `json:"value_today"`
	NumTrades            *int       `json:"num_trades"`
	MarketDataAsOf       *string    `json:"market_data_as_of"`
	LotSize              *int       `json:"lot_size"`
	MorningSession       *bool      `json:"morning_session"`
	EveningSession       *bool      `json:"evening_session"`
	WeekendSession       *bool      `json:"weekend_session"`
	CouponCalendar       []Coupon   `json:"coupon_calendar"`
	CashflowSchedule     []Cashflow `json:"cashflow_schedule"`
}
