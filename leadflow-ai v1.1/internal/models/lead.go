package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SocialLinks map[string]string

func (s SocialLinks) Value() (driver.Value, error) {
	if s == nil {
		return "{}", nil
	}
	return json.Marshal(s)
}

func (s *SocialLinks) Scan(value interface{}) error {
	if value == nil {
		*s = SocialLinks{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		if str, ok := value.(string); ok {
			b = []byte(str)
		} else {
			*s = SocialLinks{}
			return nil
		}
	}
	return json.Unmarshal(b, s)
}

const (
	LeadStatusNew        = "new"
	LeadStatusAnalyzed   = "analyzed"
	LeadStatusContacted  = "contacted"
	LeadStatusRejected   = "rejected"
)

type Lead struct {
	ID               uuid.UUID     `db:"id" json:"id"`
	TaskID           uuid.NullUUID `db:"task_id" json:"task_id,omitempty"`
	CompanyName      string        `db:"company_name" json:"company_name"`
	Website          *string       `db:"website" json:"website,omitempty"`
	Email            *string       `db:"email" json:"email,omitempty"`
	Phone            *string       `db:"phone" json:"phone,omitempty"`
	Socials          SocialLinks   `db:"socials" json:"socials"`
	QualityScore     *int          `db:"quality_score" json:"quality_score,omitempty"`
	AIRecommendation *string       `db:"ai_recommendation" json:"ai_recommendation,omitempty"`
	AIMessage        *string       `db:"ai_message" json:"ai_message,omitempty"`
	Status           string        `db:"status" json:"status"`
	CreatedAt        time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time     `db:"updated_at" json:"updated_at"`
}
