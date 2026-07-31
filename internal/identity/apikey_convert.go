package identity

import (
	"espx/internal/identity/db"
	"espx/internal/identity/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func apiKeyFromListRow(row db.ListUserAPIKeysRow) APIKey {
	key := APIKey{
		ID:        uuidFromPg(row.ID).String(),
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		key.ExpiresAt = &t
	}
	return key
}

func APIKeyToPB(key APIKey) *pb.APIKey {
	out := &pb.APIKey{
		Id:        key.ID,
		Name:      key.Name,
		CreatedAt: timestamppb.New(key.CreatedAt),
	}
	if key.ExpiresAt != nil {
		out.ExpiresAt = timestamppb.New(*key.ExpiresAt)
	}
	return out
}

func createAPIKeyResultToPB(result CreateAPIKeyResult) *pb.CreateAPIKeyResponse {
	resp := &pb.CreateAPIKeyResponse{
		Id:     result.ID,
		Name:   result.Name,
		RawKey: result.RawKey,
	}
	if result.ExpiresAt != nil {
		resp.ExpiresAt = timestamppb.New(*result.ExpiresAt)
	}
	return resp
}
