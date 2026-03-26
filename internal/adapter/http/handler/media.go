package handler

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/gin-gonic/gin"
)

type MediaHandler struct {
	uploadDir string
}

func NewMediaHandler(uploadDir string) *MediaHandler {
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}
	return &MediaHandler{uploadDir: uploadDir}
}

func (m *MediaHandler) UploadFile(ctx *gin.Context) {
	var req uploadFileRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(http.StatusBadRequest, "file is required"))
		return
	}

	ext := filepath.Ext(req.File.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(m.uploadDir, filename)

	if err := ctx.SaveUploadedFile(req.File, savePath); err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError(http.StatusInternalServerError, "failed to save file"))
		return
	}

	url := fmt.Sprintf("/api/media/image/%s", filename)
	ctx.JSON(http.StatusOK, fileResponse{
		Success:    true,
		StatusCode: http.StatusOK,
		Message:    "file uploaded successfully",
		URL:        &url,
	})
}

func (m *MediaHandler) GetImage(ctx *gin.Context) {
	filename := ctx.Param("filename")
	if filename == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError(http.StatusBadRequest, "filename is required"))
		return
	}

	filePath := filepath.Join(m.uploadDir, filepath.Base(filename))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		ctx.JSON(http.StatusNotFound, response.NewError(http.StatusNotFound, "image not found"))
		return
	}

	ctx.File(filePath)
}

type uploadFileRequest struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

type fileResponse struct {
	Success    bool    `json:"success"`
	StatusCode int     `json:"status_code"`
	Message    string  `json:"message"`
	URL        *string `json:"url,omitempty"`
}
