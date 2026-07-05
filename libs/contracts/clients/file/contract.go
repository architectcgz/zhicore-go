// Package file contains zhicore-file synchronous client contracts.
package file

const (
	ValidateRefsPath = "/api/v1/internal/files/validate-refs"

	OperationContentValidateBodyMediaRefs = "content.validate_body_media_refs"
	OperationContentValidateCover         = "content.validate_cover"
)

const (
	UsageContentBodyMedia = "CONTENT_BODY_MEDIA"
	UsageContentCover     = "CONTENT_COVER"
)

type ValidateRefsRequest struct {
	Refs  []FileRef `json:"refs"`
	Usage string    `json:"usage"`
}

type FileRef struct {
	FileID string `json:"fileId"`
	Kind   string `json:"kind,omitempty"`
}

type ValidateRefsResponse struct {
	InvalidFileIDs []string `json:"invalidFileIds,omitempty"`
}
