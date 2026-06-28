package httpserver_test

import (
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/cbr"
	"github.com/Chutchev/myinvesthelper_moex_gateway/internal/moex"
)

var _ moex.Service = moex.NewStubService()
var _ cbr.Service = cbr.NewStubService()
