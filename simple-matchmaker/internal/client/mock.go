package mpclient

import "fmt"

// MockMultiplayClient is a basic client that returns fixed data for testing.
type MockMultiplayClient struct {
}

func (m MockMultiplayClient) Allocate(fleet, region string, profile int64, uuid string) (*AllocateResponse, error) {
	fmt.Printf("Mock Allocated: %s", uuid)
	return &AllocateResponse{
		ProfileID: 0,
		UUID:      "",
		RegionID:  "",
		Created:   "",
		Error:     "",
	}, nil
}

func (m MockMultiplayClient) Allocations(fleet string, uuids ...string) ([]AllocationResponse, error) {
	fmt.Printf("Mock allocations response: %v", uuids)
	return []AllocationResponse{
		{
			ProfileID: 0,
			UUID:      "123-123-123",
			Regions:   "",
			Created:   "",
			Requested: "",
			Fulfilled: "",
			ServerID:  0,
			FleetID:   "",
			RegionID:  "",
			MachineID: 0,
			IP:        "127.0.0.1", // This port lines up with the simulated gameserver port the matchmaker will run.
			GamePort:  9200,
			Error:     "",
		},
	}, nil
}

func (m MockMultiplayClient) Deallocate(fleet, uuid string) error {
	fmt.Printf("deallocate: %s, %s\n", fleet, uuid)
	return nil
}
