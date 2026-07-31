package identity

import (
	"espx/internal/identity/pb"

	"github.com/google/uuid"
)

func loginResultFromDTO(dto LoginDTO) LoginResult {
	return LoginResult{
		AccessToken:  dto.AccessToken,
		RefreshToken: dto.RefreshToken,
		User:         authUserFromDB(dto.User),
	}
}

func loginDTOToPB(dto LoginDTO) *pb.LoginResponse {
	return &pb.LoginResponse{
		AccessToken:  dto.AccessToken,
		RefreshToken: dto.RefreshToken,
		User:         userToPB(dto.User),
	}
}

func authUserToPB(user AuthUser) *pb.User {
	return &pb.User{
		Id:         user.ID.String(),
		Email:      user.Email,
		Role:       user.Role,
		CustomerId: user.CustomerID.String(),
	}
}

func refreshResultToPB(result RefreshResult) *pb.RefreshTokenResponse {
	return &pb.RefreshTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}
}

func loginResultFromPB(resp pb.LoginResponse) LoginResult {
	out := LoginResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}
	if resp.User != nil {
		uid, _ := uuid.Parse(resp.User.Id)
		cid, _ := uuid.Parse(resp.User.CustomerId)
		out.User = AuthUser{
			ID:         uid,
			Email:      resp.User.Email,
			Role:       resp.User.Role,
			CustomerID: cid,
		}
	}
	return out
}
