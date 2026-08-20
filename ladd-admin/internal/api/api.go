package api

type UploadRequest struct {
	Filename   string `json:"filename"`
	Ciphertext []byte `json:"ciphertext"`
	Signature  []byte `json:"signature"`
}

func (r UploadRequest) SigningBytes() []byte {
	b := make([]byte, 0, len(r.Filename)+1+len(r.Ciphertext))
	b = append(b, r.Filename...)
	b = append(b, 0)
	b = append(b, r.Ciphertext...)
	return b
}

type UploadResponse struct {
	OK      bool   `json:"ok"`
	Entries int    `json:"entries,omitempty"`
	Message string `json:"message,omitempty"`
}
