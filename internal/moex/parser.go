package moex

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/apperrors"
)

// responseBlock represents one MOEX ISS data block (columns + data rows).
type responseBlock struct {
	Columns []string `json:"columns"`
	Data    [][]any  `json:"data"`
}

// indexedRow pairs column names with their cell values.
func indexedRow(columns []string, row []any) map[string]any {
	result := make(map[string]any, len(columns))
	for i, col := range columns {
		if i < len(row) {
			result[col] = row[i]
		}
	}
	return result
}

// pickValue returns the first non-nil, non-empty value among the given column names.
func pickValue(row map[string]any, names ...string) any {
	for _, name := range names {
		if val, ok := row[name]; ok && val != nil && val != "" {
			return val
		}
	}
	return nil
}

// parseOptionalInt returns nil when the raw value is nil, null, or empty.
// Returns ErrParseError for populated but invalid input.
func parseOptionalInt(raw any) (*int, error) {
	if raw == nil || raw == "" {
		return nil, nil
	}
	switch v := raw.(type) {
	case float64:
		n := int(v)
		return &n, nil
	case string:
		if v == "" {
			return nil, nil
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("%w: int %q", apperrors.ErrParseError, v)
		}
		return &n, nil
	}
	return nil, fmt.Errorf("%w: unsupported int type %T", apperrors.ErrParseError, raw)
}

// parseOptionalFloat returns nil when the raw value is nil, null, or empty.
// Returns ErrParseError for populated but invalid input.
func parseOptionalFloat(raw any) (*float64, error) {
	if raw == nil || raw == "" {
		return nil, nil
	}
	switch v := raw.(type) {
	case float64:
		return &v, nil
	case string:
		if v == "" {
			return nil, nil
		}
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: float %q", apperrors.ErrParseError, v)
		}
		return &n, nil
	}
	return nil, fmt.Errorf("%w: unsupported float type %T", apperrors.ErrParseError, raw)
}

// parseOptionalBool returns nil when the raw value is nil, null, or empty.
// Recognizes true/false, 1/0, "yes"/"no" (case-insensitive).
// Returns ErrParseError for populated but invalid input.
func parseOptionalBool(raw any) (*bool, error) {
	if raw == nil || raw == "" {
		return nil, nil
	}
	switch v := raw.(type) {
	case bool:
		return &v, nil
	case float64:
		if v == 1 {
			b := true
			return &b, nil
		}
		if v == 0 {
			b := false
			return &b, nil
		}
		return nil, fmt.Errorf("%w: bool %v", apperrors.ErrParseError, v)
	case string:
		if v == "" {
			return nil, nil
		}
		switch strings.ToLower(v) {
		case "true", "1", "yes":
			b := true
			return &b, nil
		case "false", "0", "no":
			b := false
			return &b, nil
		}
		return nil, fmt.Errorf("%w: bool %q", apperrors.ErrParseError, v)
	}
	return nil, fmt.Errorf("%w: unsupported bool type %T", apperrors.ErrParseError, raw)
}

// normalizeCouponType maps Russian/English coupon type labels to "floating" or "fixed".
func normalizeCouponType(raw string) string {
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "перемен") ||
		strings.Contains(lower, "плава") ||
		strings.Contains(lower, "float") ||
		strings.Contains(lower, "variable") {
		return "floating"
	}
	if strings.Contains(lower, "фикс") ||
		strings.Contains(lower, "постоян") ||
		strings.Contains(lower, "fixed") ||
		strings.Contains(lower, "constant") {
		return "fixed"
	}
	return raw
}

// ParseBondDescription decodes a MOEX bond description response into a Bond.
// The description data is a property table where column 0 is the key and column 2 is the value.
func ParseBondDescription(data []byte) (Bond, error) {
	var resp struct {
		Description *responseBlock `json:"description"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return Bond{}, fmt.Errorf("%w: description JSON: %v", apperrors.ErrParseError, err)
	}
	if resp.Description == nil || len(resp.Description.Data) == 0 {
		return Bond{}, fmt.Errorf("%w: description block missing or empty", apperrors.ErrParseError)
	}

	// Build a map from key to value for the description properties.
	// Column 0 is the key, column 2 is the value.
	props := make(map[string]any)
	for _, row := range resp.Description.Data {
		if len(row) < 3 {
			continue
		}
		key, ok := row[0].(string)
		if !ok || key == "" {
			continue
		}
		props[key] = row[2]
	}

	bond := Bond{}

	// Map fields from description properties
	{
		var val any
		if v, ok := props["GROUPNAME"]; ok && v != nil && v != "" {
			val = v
		} else if v, ok := props["GROUP"]; ok && v != nil && v != "" {
			val = v
		}
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.InstrumentGroup = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.InstrumentGroup = &s
			}
		}
	}

	{
		var val any
		if v, ok := props["MATDATE"]; ok && v != nil && v != "" {
			val = v
		} else if v, ok := props["MATURITYDATE"]; ok && v != nil && v != "" {
			val = v
		}
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.MaturityDate = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.MaturityDate = &s
			}
		}
	}

	{
		var val any
		if v, ok := props["FACEVALUE"]; ok && v != nil && v != "" {
			val = v
		} else if v, ok := props["INITIALFACEVALUE"]; ok && v != nil && v != "" {
			val = v
		}
		if val != nil && val != "" {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return Bond{}, err
			}
			bond.FaceValue = f
			bond.InitialFaceValue = f
		}
	}

	if val, ok := props["FACEUNIT"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.FaceUnit = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.FaceUnit = &s
			}
		}
	}

	if val, ok := props["CURRENCY"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.Currency = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.Currency = &s
			}
		}
	}

	if val, ok := props["ISSUESIZE"]; ok {
		if val != nil && val != "" {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return Bond{}, err
			}
			bond.IssueSize = f
		}
	}

	if val, ok := props["COUPONPERCENT"]; ok {
		if val != nil && val != "" {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return Bond{}, err
			}
			bond.CouponPercent = f
		}
	}

	if val, ok := props["COUPONVALUE"]; ok {
		if val != nil && val != "" {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return Bond{}, err
			}
			bond.CouponValue = f
		}
	}

	{
		var val any
		if v, ok := props["COUPONTYPE"]; ok && v != nil && v != "" {
			val = v
		} else if v, ok := props["COUPON_TYPE"]; ok && v != nil && v != "" {
			val = v
		} else if v, ok := props["COUPON_TYPE_NAME"]; ok && v != nil && v != "" {
			val = v
		}
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				normalized := normalizeCouponType(v)
				bond.CouponType = &normalized
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				normalized := normalizeCouponType(s)
				bond.CouponType = &normalized
			}
		}
	}

	if val, ok := props["COUPONFREQUENCY"]; ok {
		if val != nil && val != "" {
			n, err := parseOptionalInt(val)
			if err != nil {
				return Bond{}, err
			}
			bond.CouponFrequency = n
		}
	}

	if val, ok := props["ISSUEDATE"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.IssueDate = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.IssueDate = &s
			}
		}
	}

	if val, ok := props["STARTDATEMOEX"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.StartDateMOEX = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.StartDateMOEX = &s
			}
		}
	}

	if val, ok := props["REGISTRYDATE"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.RegistryDate = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.RegistryDate = &s
			}
		}
	}

	if val, ok := props["REGNUMBER"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.RegNumber = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.RegNumber = &s
			}
		}
	}

	if val, ok := props["ISSUERID"]; ok {
		if val != nil && val != "" {
			n, err := parseOptionalInt(val)
			if err != nil {
				return Bond{}, err
			}
			bond.IssuerID = n
		}
	}

	if val, ok := props["BONDTYPE"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.BondType = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.BondType = &s
			}
		}
	}

	if val, ok := props["BONDSUBTYPE"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.BondSubtype = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.BondSubtype = &s
			}
		}
	}

	if val, ok := props["SHORTNAME"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.ShortName = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.ShortName = &s
			}
		}
	}

	if val, ok := props["NAME"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.Name = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.Name = &s
			}
		}
	}

	if val, ok := props["LATINNAME"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.LatinName = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.LatinName = &s
			}
		}
	}

	if val, ok := props["TYPE"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.Type = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.Type = &s
			}
		}
	}

	if val, ok := props["TYPENAME"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.TypeName = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.TypeName = &s
			}
		}
	}

	if val, ok := props["LISTLEVEL"]; ok {
		if val != nil && val != "" {
			n, err := parseOptionalInt(val)
			if err != nil {
				return Bond{}, err
			}
			bond.ListLevel = n
		}
	}

	if val, ok := props["ISQUALIFIEDINVESTORS"]; ok {
		if val != nil && val != "" {
			b, err := parseOptionalBool(val)
			if err != nil {
				return Bond{}, err
			}
			bond.IsQualifiedInvestors = b
		}
	}

	if val, ok := props["HASPROSPECTUS"]; ok {
		if val != nil && val != "" {
			b, err := parseOptionalBool(val)
			if err != nil {
				return Bond{}, err
			}
			bond.HasProspectus = b
		}
	}

	if val, ok := props["HASDEFAULT"]; ok {
		if val != nil && val != "" {
			b, err := parseOptionalBool(val)
			if err != nil {
				return Bond{}, err
			}
			bond.HasDefault = b
		}
	}

	if val, ok := props["HASTECHNICALDEFAULT"]; ok {
		if val != nil && val != "" {
			b, err := parseOptionalBool(val)
			if err != nil {
				return Bond{}, err
			}
			bond.HasTechnicalDefault = b
		}
	}

	if val, ok := props["NEXTCOUPONDATE"]; ok {
		if val != nil && val != "" {
			switch v := val.(type) {
			case string:
				bond.NextCouponDate = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.NextCouponDate = &s
			}
		}
	}

	if val, ok := props["DAYS TO REDEMPTION"]; ok {
		if val != nil && val != "" {
			n, err := parseOptionalInt(val)
			if err != nil {
				return Bond{}, err
			}
			bond.DaysToRedemption = n
		}
	}

	return bond, nil
}

// ParseUniverseResponse decodes a MOEX universe response into a MarketUniverse.
// It joins securities and marketdata blocks by SECID.
func ParseUniverseResponse(data []byte) (MarketUniverse, error) {
	var resp struct {
		Securities *responseBlock `json:"securities"`
		MarketData *responseBlock `json:"marketdata"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("%w: universe JSON: %v", apperrors.ErrParseError, err)
	}
	if resp.Securities == nil {
		return nil, fmt.Errorf("%w: securities block missing", apperrors.ErrParseError)
	}

	// Build a map from SECID to Bond from the securities block.
	secMap := make(map[string]Bond)
	for _, row := range resp.Securities.Data {
		if len(row) == 0 {
			continue
		}
		props := indexedRow(resp.Securities.Columns, row)
		secid := pickValue(props, "SECID")
		if secid == nil {
			continue
		}
		secidStr, _ := secid.(string)
		if secidStr == "" {
			continue
		}

		bond := Bond{ISIN: secidStr, SECID: &secidStr, Ticker: &secidStr}

		if val := pickValue(props, "SHORTNAME"); val != nil {
			switch v := val.(type) {
			case string:
				bond.ShortName = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.ShortName = &s
			}
		}

		if val := pickValue(props, "LOTSIZE"); val != nil {
			n, err := parseOptionalInt(val)
			if err != nil {
				return nil, err
			}
			bond.LotSize = n
		}

		if val := pickValue(props, "FACEVALUE"); val != nil {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return nil, err
			}
			bond.FaceValue = f
		}

		if val := pickValue(props, "FACEUNIT"); val != nil {
			switch v := val.(type) {
			case string:
				bond.FaceUnit = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.FaceUnit = &s
			}
		}

		if val := pickValue(props, "MATDATE"); val != nil {
			switch v := val.(type) {
			case string:
				bond.MaturityDate = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.MaturityDate = &s
			}
		}

		if val := pickValue(props, "COUPONPERCENT"); val != nil {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return nil, err
			}
			bond.CouponPercent = f
		}

		if val := pickValue(props, "COUPONVALUE"); val != nil {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return nil, err
			}
			bond.CouponValue = f
		}

		if val := pickValue(props, "ISSUESIZE"); val != nil {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return nil, err
			}
			bond.IssueSize = f
		}

		secMap[secidStr] = bond
	}

	// If there's no marketdata block, return what we have.
	if resp.MarketData == nil || len(resp.MarketData.Data) == 0 {
		result := make(MarketUniverse, 0, len(secMap))
		for _, bond := range secMap {
			result = append(result, bond)
		}
		return result, nil
	}

	// Join marketdata rows with securities by SECID.
	result := make(MarketUniverse, 0, len(secMap))
	for _, row := range resp.MarketData.Data {
		if len(row) == 0 {
			continue
		}
		props := indexedRow(resp.MarketData.Columns, row)
		secid := pickValue(props, "SECID")
		if secid == nil {
			continue
		}
		secidStr, _ := secid.(string)
		if secidStr == "" {
			continue
		}

		bond, exists := secMap[secidStr]
		if !exists {
			continue
		}

		// Map market data fields
		if val := pickValue(props, "LAST", "LCURRENTPRICE"); val != nil {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return nil, err
			}
			bond.Price = f
		}

		if val := pickValue(props, "YIELD"); val != nil {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return nil, err
			}
			bond.YieldToMaturity = f
		}

		if val := pickValue(props, "DURATION"); val != nil {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return nil, err
			}
			bond.Duration = f
		}

		if val := pickValue(props, "ACCRUEDINT", "ACCINT"); val != nil {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return nil, err
			}
			bond.AccruedInterest = f
		}

		if val := pickValue(props, "VALTODAY"); val != nil {
			f, err := parseOptionalFloat(val)
			if err != nil {
				return nil, err
			}
			bond.ValueToday = f
		}

		if val := pickValue(props, "NUMTRADES"); val != nil {
			n, err := parseOptionalInt(val)
			if err != nil {
				return nil, err
			}
			bond.NumTrades = n
		}

		if val := pickValue(props, "SYSTIME", "UPDATETIME"); val != nil {
			switch v := val.(type) {
			case string:
				bond.MarketDataAsOf = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.MarketDataAsOf = &s
			}
		}

		if val := pickValue(props, "LOTSIZE"); val != nil {
			n, err := parseOptionalInt(val)
			if err != nil {
				return nil, err
			}
			bond.LotSize = n
		}

		if val := pickValue(props, "CURRENCYID", "FACEUNIT"); val != nil {
			switch v := val.(type) {
			case string:
				bond.Currency = &v
			case float64:
				s := strconv.FormatFloat(v, 'f', -1, 64)
				bond.Currency = &s
			}
		}

		result = append(result, bond)
	}

	return result, nil
}

// ParseBondization decodes a MOEX bondization response into coupons and cashflows.
// Coupons are taxable; principal cashflows are non-taxable.
func ParseBondization(data []byte) ([]Coupon, []Cashflow, error) {
	var resp struct {
		Coupons    *responseBlock `json:"coupons"`
		Principal  *responseBlock `json:"principal"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, fmt.Errorf("%w: bondization JSON: %v", apperrors.ErrParseError, err)
	}

	var coupons []Coupon
	var cashflows []Cashflow

	// Parse coupons
	if resp.Coupons != nil && len(resp.Coupons.Data) > 0 {
		for _, row := range resp.Coupons.Data {
			if len(row) == 0 {
				continue
			}
			props := indexedRow(resp.Coupons.Columns, row)

			dateRaw := pickValue(props, "COUPONDATE", "DATE")
			if dateRaw == nil {
				continue
			}
			var dateStr string
			switch v := dateRaw.(type) {
			case string:
				dateStr = v
			case float64:
				dateStr = strconv.FormatFloat(v, 'f', -1, 64)
			}

			valRaw := pickValue(props, "VALUE", "COUPONVALUE", "LEGALCLOSEPRICE")
			if valRaw == nil {
				continue
			}
			val, err := parseOptionalFloat(valRaw)
			if err != nil {
				return nil, nil, err
			}
			if val == nil {
				continue
			}

			coupons = append(coupons, Coupon{
				Date:  dateStr,
				Value: *val,
			})

			cashflows = append(cashflows, Cashflow{
				Date:    dateStr,
				Amount:  *val,
				Kind:    "coupon",
				Taxable: true,
			})
		}
	}

	// Parse principal (amortization)
	if resp.Principal != nil && len(resp.Principal.Data) > 0 {
		for _, row := range resp.Principal.Data {
			if len(row) == 0 {
				continue
			}
			props := indexedRow(resp.Principal.Columns, row)

			dateRaw := pickValue(props, "AMORTDATE", "DATE", "MATDATE")
			if dateRaw == nil {
				continue
			}
			var dateStr string
			switch v := dateRaw.(type) {
			case string:
				dateStr = v
			case float64:
				dateStr = strconv.FormatFloat(v, 'f', -1, 64)
			}

			valRaw := pickValue(props, "VALUE", "AMORTVALUE", "FACEVALUE", "VALUEPRC")
			if valRaw == nil {
				continue
			}
			val, err := parseOptionalFloat(valRaw)
			if err != nil {
				return nil, nil, err
			}
			if val == nil {
				continue
			}

			cashflows = append(cashflows, Cashflow{
				Date:    dateStr,
				Amount:  *val,
				Kind:    "principal",
				Taxable: false,
			})
		}
	}

	// Sort cashflows by (date, kind)
	sort.Slice(cashflows, func(i, j int) bool {
		if cashflows[i].Date != cashflows[j].Date {
			return cashflows[i].Date < cashflows[j].Date
		}
		return cashflows[i].Kind < cashflows[j].Kind
	})

	return coupons, cashflows, nil
}

// ParseMarketData decodes a MOEX market-data response into BondMarketData.
// Returns (nil, nil) when the data block is empty.
func ParseMarketData(data []byte) (*BondMarketData, error) {
	var resp struct {
		MarketData *responseBlock `json:"marketdata"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("%w: market data JSON: %v", apperrors.ErrParseError, err)
	}
	if resp.MarketData == nil || len(resp.MarketData.Data) == 0 {
		return nil, nil
	}
	row := indexedRow(resp.MarketData.Columns, resp.MarketData.Data[0])

	md := &BondMarketData{}

	priceRaw := pickValue(row, "LAST", "LCURRENTPRICE")
	price, err := parseOptionalFloat(priceRaw)
	if err != nil {
		return nil, err
	}
	md.Price = price

	yieldRaw := pickValue(row, "YIELD")
	yield, err := parseOptionalFloat(yieldRaw)
	if err != nil {
		return nil, err
	}
	md.YieldToMaturity = yield

	durRaw := pickValue(row, "DURATION")
	dur, err := parseOptionalFloat(durRaw)
	if err != nil {
		return nil, err
	}
	md.Duration = dur

	accRaw := pickValue(row, "ACCRUEDINT", "ACCINT")
	acc, err := parseOptionalFloat(accRaw)
	if err != nil {
		return nil, err
	}
	md.AccruedInterest = acc

	valRaw := pickValue(row, "VALTODAY")
	val, err := parseOptionalFloat(valRaw)
	if err != nil {
		return nil, err
	}
	md.ValueToday = val

	tradesRaw := pickValue(row, "NUMTRADES")
	trades, err := parseOptionalInt(tradesRaw)
	if err != nil {
		return nil, err
	}
	md.NumTrades = trades

	timeRaw := pickValue(row, "SYSTIME", "UPDATETIME")
	if timeRaw != nil {
		switch v := timeRaw.(type) {
		case string:
			md.MarketDataAsOf = &v
		case float64:
			s := strconv.FormatFloat(v, 'f', -1, 64)
			md.MarketDataAsOf = &s
		}
	}

	lotRaw := pickValue(row, "LOTSIZE")
	lot, err := parseOptionalInt(lotRaw)
	if err != nil {
		return nil, err
	}
	md.LotSize = lot

	currencyRaw := pickValue(row, "CURRENCYID", "FACEUNIT")
	if currencyRaw != nil {
		switch v := currencyRaw.(type) {
		case string:
			md.Currency = &v
		case float64:
			s := strconv.FormatFloat(v, 'f', -1, 64)
			md.Currency = &s
		}
	}

	faceRaw := pickValue(row, "FACEUNIT")
	if faceRaw != nil {
		switch v := faceRaw.(type) {
		case string:
			md.FaceUnit = &v
		case float64:
			s := strconv.FormatFloat(v, 'f', -1, 64)
			md.FaceUnit = &s
		}
	}

	return md, nil
}
