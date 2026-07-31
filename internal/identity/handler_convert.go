package identity

import (
	"espx/internal/identity/db"
	"espx/internal/identity/pb"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func uuidFromPg(u pgtype.UUID) uuid.UUID {
	return uuid.UUID(u.Bytes)
}

func userToPB(user db.User) *pb.User {
	return &pb.User{
		Id:         uuidFromPg(user.ID).String(),
		Email:      user.Email,
		Role:       user.Role,
		CustomerId: uuidFromPg(user.CustomerID).String(),
		CreatedAt:  timestamppb.New(user.CreatedAt.Time),
	}
}

