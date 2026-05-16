package domain

import (
	"time"
)

type Department struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	ParentID  *int      `gorm:"index" json:"parent_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	Employees []Employee   `gorm:"foreignKey:DepartmentID;constraint:OnDelete:CASCADE" json:"employees,omitempty"`
	Children  []Department `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"children,omitempty"`
}

type Employee struct {
	ID           int        `gorm:"primaryKey" json:"id"`
	DepartmentID int        `gorm:"not null;index" json:"department_id"`
	FullName     string     `gorm:"not null" json:"full_name"`
	Position     string     `gorm:"not null" json:"position"`
	HiredAt      *time.Time `gorm:"type:date" json:"hired_at"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
}
