package twofa_service_api

import (
	pb "github.com/vbncursed/vkr/twofa/internal/pb/twofa_api"
)

// Service defines the contract the API layer requires from the TwoFA service.
type Service interface {
	// Methods added in Phase 7
}

// TwoFAServiceAPI implements the gRPC TwoFAServiceServer interface.
type TwoFAServiceAPI struct {
	pb.UnimplementedTwoFAServiceServer
	service Service
}

// NewTwoFAServiceAPI creates a new TwoFAServiceAPI instance.
func NewTwoFAServiceAPI(service Service) *TwoFAServiceAPI {
	return &TwoFAServiceAPI{
		service: service,
	}
}
