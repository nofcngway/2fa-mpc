package twofa_service_api

import (
	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
	"github.com/vbncursed/vkr/twofa/internal/services/twofaService"
)

// TwoFAServiceAPI implements the gRPC TwoFAServiceServer interface.
type TwoFAServiceAPI struct {
	pb.UnimplementedTwoFAServiceServer
	service *twofaService.TwoFAService
}

// NewTwoFAServiceAPI creates a new TwoFAServiceAPI instance.
func NewTwoFAServiceAPI(service *twofaService.TwoFAService) *TwoFAServiceAPI {
	return &TwoFAServiceAPI{
		service: service,
	}
}
