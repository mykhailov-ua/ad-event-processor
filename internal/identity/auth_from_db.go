package identity

func loginResultFromDTO(dto LoginDTO) LoginResult {
	return LoginResult{
		AccessToken:  dto.AccessToken,
		RefreshToken: dto.RefreshToken,
		User:         authUserFromDB(dto.User),
	}
}
