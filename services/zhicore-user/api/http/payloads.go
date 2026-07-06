package httpapi

type userProfileResp struct {
	PublicID               string `json:"publicId"`
	Nickname               string `json:"nickname"`
	AvatarFileID           string `json:"avatarFileId,omitempty"`
	AvatarURL              string `json:"avatarUrl,omitempty"`
	Bio                    string `json:"bio,omitempty"`
	StrangerMessageAllowed bool   `json:"strangerMessageAllowed"`
	ProfileVersion         int64  `json:"profileVersion"`
}

type relationshipPageResp struct {
	Items      []userProfileResp `json:"items"`
	NextCursor string            `json:"nextCursor,omitempty"`
	HasMore    bool              `json:"hasMore"`
}
