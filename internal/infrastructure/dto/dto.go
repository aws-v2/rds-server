package dto

import "time"

type CreateInstanceInput struct {
	Name     string `json:"name" binding:"required"`
	User     string `json:"user" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type InstanceOutput struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Engine    string    `json:"engine"`
	Port      int       `json:"port"`
	User      string    `json:"user"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type ListInstancesOutput struct {
	Instances []InstanceOutput `json:"instances"`
}
