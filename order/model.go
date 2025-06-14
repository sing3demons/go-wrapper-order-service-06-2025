package order

import (
	"time"
)

type Order struct {
	ID         string    `json:"id"`
	Href       string    `json:"href,omitempty"`
	CustomerID string    `json:"customer_id"`
	Products   []string  `json:"products"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeletedAt  time.Time `json:"-"`
}

func (o *Order) IsDeleted() bool {
	return !o.DeletedAt.IsZero()
}
func (o *Order) IsActive() bool {
	return o.DeletedAt.IsZero()
}
func (o *Order) IsPending() bool {
	return o.Status == "pending"
}
func (o *Order) IsCompleted() bool {
	return o.Status == "completed"
}
