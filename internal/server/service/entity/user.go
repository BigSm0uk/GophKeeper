package entity

import (
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/server/domain/models"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type User struct {
	ID        string
	Username  string
	Email     *string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
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

func MapUserFromRequest(rq *pb.RegisterRequest) (*User, error) {
	if rq == nil {
		return nil, fmt.Errorf("request is nil")
	}
	return &User{
		Username: rq.Username,
		Password: rq.Password,
		Email:    rq.Email,
	}, nil
}

// MapUserToProfile converts domain User to protobuf UserProfile.
func MapUserToProfile(user *models.User, storageUsed, storageLimit int64) (*pb.UserProfile, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}
	profile := &pb.UserProfile{
		UserId:       user.ID,
		Username:     user.Username,
		Email:        user.Email,
		CreatedAt:    timestamppb.New(user.CreatedAt),
		StorageUsed:  storageUsed,
		StorageLimit: storageLimit,
	}
	return profile, nil
}

// MapSessionToSessionInfo converts domain Session to protobuf SessionInfo.
func MapSessionToSessionInfo(session *models.Session, isCurrent bool) (*pb.SessionInfo, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}
	info := &pb.SessionInfo{
		SessionId:  session.ID,
		ClientId:   session.ClientID,
		IpAddress:  session.IPAddress,
		CreatedAt:  timestamppb.New(session.CreatedAt),
		LastUsedAt: timestamppb.New(session.LastUsedAt),
		IsCurrent:  isCurrent,
	}
	return info, nil
}
