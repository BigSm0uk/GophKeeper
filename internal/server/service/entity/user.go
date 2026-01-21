package entity

import (
	"fmt"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     *string   `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) MapToResponse() (*pb.RegisterResponse, error) {
	if u == nil {
		return nil, fmt.Errorf("user is nil")
	}
	return &pb.RegisterResponse{
		UserId:    u.ID,
		Username:  u.Username,
		CreatedAt: timestamppb.New(u.CreatedAt),
	}, nil
}

func MapFromRequest(rq *pb.RegisterRequest) (*User, error) {
	if rq == nil {
		return nil, fmt.Errorf("request is nil")
	}
	return &User{
		Username: rq.Username,
		Password: rq.Password,
		Email:    rq.Email,
	}, nil
}
