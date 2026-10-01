package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type FileStatus string

const (
	PendingStatus FileStatus = FileStatus("pending")
	Uploaded      FileStatus = FileStatus("uploaded")
	Validating    FileStatus = FileStatus("validating")
	Cancelled     FileStatus = FileStatus("cancelled")
)

type File struct {
	bun.BaseModel `bun:"table:files,alias:f"`
	ID            uuid.UUID `bun:"id,pk"`
	UserID        uuid.UUID `bun:"user_id"`
	ObjectKey     string    `bun:"object_key"`
	FileName      string    `bun:"file_name"`
	ContentType   string    `bun:"content_type"`
	SizeBites     int       `bun:"size_bytes"`
	Status        string    `bun:"status"`
	CreatedAt     time.Time `bun:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at"`
	DeletedAt     time.Time `bun:"deleted_at"`
}

func NewFile(
	userID uuid.UUID,
	filename,
	contentType string,
	size int,
) *File {
	id, _ := uuid.NewV7()
	f := &File{
		ID:          id,
		CreatedAt:   time.Now(),
		UserID:      userID,
		FileName:    filename,
		ContentType: contentType,
		Status:      string(PendingStatus),
		SizeBites:   size,
	}

	f.generateObjectKey()
	return f
}

func (f *File) generateObjectKey() {
	f.ObjectKey = fmt.Sprintf("quarantine/%s/%s", f.UserID, f.ID)
}

func (f *File) SetAsUploaded() {
	f.Status = string(Uploaded)
	f.UpdatedAt = time.Now()
}

func (f *File) GenerateFinalKey() {
	f.ObjectKey = fmt.Sprintf("files/%s/%s", f.UserID, f.ID)
}
