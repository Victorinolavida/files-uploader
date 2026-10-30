package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func CeilDiv(a, b int) int {
	if b == 0 {
		panic("division by zero")
	}
	return (a + b - 1) / b
}

type FileStatus string
type UploadMode string

const (
	PendingStatus   FileStatus = FileStatus("pending")
	Uploaded        FileStatus = FileStatus("uploaded")
	UploadingStatus FileStatus = FileStatus("uploading")
	Validating      FileStatus = FileStatus("validating")
	Rejected        FileStatus = FileStatus("rejected")

	SinglePart UploadMode = UploadMode("single")
	MultiPart  UploadMode = UploadMode("multipart")
)

const (
	MaxSize    = 1 << 32 // 4GB,
	BreakPoint = 10 << 20
)

type File struct {
	bun.BaseModel     `bun:"table:files,alias:f"`
	ID                uuid.UUID `bun:"id,pk"`
	UserID            uuid.UUID `bun:"user_id"`
	MultipartUploadID *string   `bun:"multipart_upload_id"`
	ObjectKey         string    `bun:"object_key"`
	FileName          string    `bun:"file_name"`
	ContentType       string    `bun:"content_type"`
	Mode              string    `bun:"mode"`
	PartSize          int       `bun:"part_size"`
	PartCount         int       `bun:"part_count"`
	SizeBites         int       `bun:"size_bytes"`
	Status            string    `bun:"status"`
	CreatedAt         time.Time `bun:"created_at"`
	UpdatedAt         time.Time `bun:"updated_at"`
	DeletedAt         time.Time `bun:"deleted_at"`
	RejectReason      string    `bun:"reject_reason"`
}

func NewFile(
	userID uuid.UUID,
	filename,
	contentType string,
	size int,
) (*File, error) {
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
	if f.SizeBites == 0 {
		return nil, errors.New("file can not have size 0")
	}
	if f.SizeBites > MaxSize {
		return nil, errors.New("file is bigger than 4GB")
	}
	f.Mode = string(SinglePart)
	if f.SizeBites >= BreakPoint {
		f.Mode = string(MultiPart)
	}
	f.PartCount = CeilDiv(f.SizeBites, BreakPoint)
	f.PartSize = CeilDiv(f.SizeBites, f.PartCount)

	f.generateObjectKey()
	return f, nil
}

func (f *File) generateObjectKey() {
	f.ObjectKey = fmt.Sprintf("quarantine/%s/%s", f.UserID, f.ID)
}

func (f *File) SetAsUploaded() {
	f.Status = string(Uploaded)
	f.UpdatedAt = time.Now()
	f.GenerateFinalKey()
}

func (f *File) GenerateFinalKey() {
	f.ObjectKey = fmt.Sprintf("files/%s/%s", f.UserID, f.ID)
}

func (f *File) Reject(reason string) {
	now := time.Now()
	f.RejectReason = reason
	f.DeletedAt = now
	f.UpdatedAt = now
	f.Status = string(Rejected)
}

func (f *File) SetAsUploading() error {
	if f.Status != string(PendingStatus) {
		return errors.New("can not pass to uploading status")
	}
	f.Status = string(UploadingStatus)
	f.UpdatedAt = time.Now()
	return nil
}
