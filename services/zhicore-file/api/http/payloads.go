package httpapi

type UploadFileResp struct {
	FileID        string `json:"fileId"`
	URL           string `json:"url"`
	FileSize      int64  `json:"fileSize"`
	FileHash      string `json:"fileHash,omitempty"`
	InstantUpload bool   `json:"instantUpload"`
	UploadTime    string `json:"uploadTime,omitempty"`
	AccessLevel   string `json:"accessLevel"`
	OriginalName  string `json:"originalName"`
	ContentType   string `json:"contentType"`
}
