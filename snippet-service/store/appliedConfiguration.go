package store

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ValidateAppliedConfiguration(c *pb.AppliedConfiguration) error {
	if err := c.Validate(); err != nil {
		return errInvalidArgument(err)
	}
	return nil
}

type AppliedConfiguration struct {
	RecordID string `bson:"_id"`
	pb.AppliedConfiguration
}

func MakeAppliedConfiguration(c *pb.AppliedConfiguration) AppliedConfiguration {
	return AppliedConfiguration{
		AppliedConfiguration: pb.AppliedConfiguration{
			Id:              c.GetId(),
			DeviceId:        c.GetDeviceId(),
			ConfigurationId: c.GetConfigurationId().Clone(),
			ExecutedBy:      c.CloneExecutedBy(),
			Resources:       c.CloneAppliedConfiguration_Resources(),
			Owner:           c.GetOwner(),
			Timestamp:       c.GetTimestamp(),
		},
	}
}

func (c *AppliedConfiguration) GetAppliedConfiguration() *pb.AppliedConfiguration {
	if c == nil {
		return nil
	}
	return &c.AppliedConfiguration
}

func (c *AppliedConfiguration) UnmarshalBSON(data []byte) error {
	var recordID string
	update := func(json map[string]interface{}) error {
		recordIDI, ok := json[pb.RecordIDKey]
		if ok {
			recordID = recordIDI.(primitive.ObjectID).Hex()
		}
		delete(json, pb.RecordIDKey)
		return nil
	}
	err := pkgMongo.UnmarshalProtoBSON(data, &c.AppliedConfiguration, update)
	if err != nil {
		return err
	}
	if c.GetId() == "" && recordID != "" {
		c.RecordID = recordID
	}
	return nil
}

type UpdateAppliedConfigurationResourceRequest struct {
	AppliedConfigurationID string
	AppliedCondition       *pb.AppliedConfiguration_LinkedTo
	StatusFilter           []pb.AppliedConfiguration_Resource_Status
	Resource               *pb.AppliedConfiguration_Resource
}

func (u *UpdateAppliedConfigurationResourceRequest) Validate() error {
	if _, err := uuid.Parse(u.AppliedConfigurationID); err != nil {
		return errInvalidArgument(fmt.Errorf("invalid ID(%v): %w", u.AppliedConfigurationID, err))
	}
	if u.Resource == nil {
		return errInvalidArgument(errors.New("resource is required"))
	}
	if err := u.Resource.Validate(); err != nil {
		return errInvalidArgument(err)
	}
	return nil
}
