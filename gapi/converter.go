package gapi

import (
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertUser(user *db.User) *pb.User {
	return &pb.User{
		Username:  user.Username,
		FullName:  user.FullName,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt.Time),
	}
}
