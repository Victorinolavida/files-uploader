package dto

type CommonFileResponse struct {
	UploadID  string `json:"upload_id"`
	Mode      string `json:"mode"`
	ExpiresAt string `json:"expires_at"`
}

type FileSingleUploadResponse struct {
	CommonFileResponse
	Method  string         `json:"method"`
	URL     string         `json:"url"`
	Headers map[string]any `json:"headers"`
}

type FileMultipleUploadPart struct {
	PartNumber int            `json:"part_number"`
	URL        string         `json:"url"`
	Headers    map[string]any `json:"headers"`
	Method     string         `json:"method"`
}

type FileMultipleUploadResponse struct {
	CommonFileResponse
	PartSize  int                      `json:"part_size"`
	PartCount int                      `json:"part_count"`
	Parts     []FileMultipleUploadPart `json:"parts"`
}

type FilePartsRequest struct {
	PartNumbers []int `json:"part_numbers"`
}

type FilePartResponse struct {
	PartNumber int    `json:"part_number"`
	URL        string `json:"url"`
}

type FilePartsResponse struct {
	Parts     []FilePartResponse `json:"parts"`
	ExpiresAt string             `json:"expired_at"`
}

type FileCompletePartRequest struct {
	PartNumber int    `json:"part_number"`
	Etag       string `json:"etag"`
}
type FileCompleteRequest struct {
	Parts FileCompletePartRequest `json:"parts"`
}
type FileCompletedResponse struct {
	UploadID string `json:"upload_id"`
	Status   string `json:"status"`
}
