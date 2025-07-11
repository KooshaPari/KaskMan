package services

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/repositories"
	"github.com/nfnt/resize"
	"github.com/sirupsen/logrus"
)

// AssetService handles project asset management (screenshots, videos, documents)
type AssetService struct {
	logger      *logrus.Logger
	assetRepo   repositories.ProjectAssetRepository
	projectRepo repositories.ProjectRepository
	storagePath string
}

// NewAssetService creates a new asset service instance
func NewAssetService(
	logger *logrus.Logger,
	assetRepo repositories.ProjectAssetRepository,
	projectRepo repositories.ProjectRepository,
	storagePath string,
) *AssetService {
	return &AssetService{
		logger:      logger,
		assetRepo:   assetRepo,
		projectRepo: projectRepo,
		storagePath: storagePath,
	}
}

// AssetUploadRequest represents an asset upload request
type AssetUploadRequest struct {
	ProjectID   uuid.UUID
	AssetType   string // screenshot, video, document, demo
	Title       string
	Description string
	File        multipart.File
	Filename    string
	ContentType string
	IsPublic    bool
	Tags        []string
	UploadedBy  uuid.UUID
}

// UploadAsset uploads and stores a project asset
func (s *AssetService) UploadAsset(ctx context.Context, req AssetUploadRequest) (*models.ProjectAsset, error) {
	// Validate project exists
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// Generate unique filename
	assetID := uuid.New()
	ext := filepath.Ext(req.Filename)
	filename := fmt.Sprintf("%s%s", assetID.String(), ext)
	
	// Create project asset directory
	assetDir := filepath.Join(s.storagePath, "projects", req.ProjectID.String(), "assets")
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create asset directory: %w", err)
	}
	
	// Save file
	filePath := filepath.Join(assetDir, filename)
	if err := s.saveFile(req.File, filePath); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}
	
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	
	// Generate thumbnail if it's an image
	thumbnailPath := ""
	if s.isImage(req.ContentType) {
		thumbnailPath, err = s.generateThumbnail(filePath, assetDir)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to generate thumbnail")
		}
	}
	
	// Extract metadata
	metadata := s.extractMetadata(filePath, req.ContentType)
	
	// Create asset record
	asset := &models.ProjectAsset{
		BaseModel: models.BaseModel{ID: assetID},
		ProjectID: req.ProjectID,
		AssetType: req.AssetType,
		Title:     req.Title,
		Description: req.Description,
		FilePath:  filePath,
		FileSize:  fileInfo.Size(),
		MimeType:  req.ContentType,
		Thumbnail: thumbnailPath,
		Metadata:  metadata,
		Tags:      strings.Join(req.Tags, ","),
		IsPublic:  req.IsPublic,
		UploadedBy: req.UploadedBy,
	}
	
	if err := s.assetRepo.Create(ctx, asset); err != nil {
		// Clean up file if database save fails
		os.Remove(filePath)
		if thumbnailPath != "" {
			os.Remove(thumbnailPath)
		}
		return nil, fmt.Errorf("failed to save asset record: %w", err)
	}
	
	s.logger.WithFields(logrus.Fields{
		"asset_id":   asset.ID,
		"project_id": req.ProjectID,
		"type":       req.AssetType,
		"filename":   req.Filename,
	}).Info("Asset uploaded successfully")
	
	return asset, nil
}

// GenerateScreenshot captures a screenshot of a web application
func (s *AssetService) GenerateScreenshot(ctx context.Context, projectID uuid.UUID, url string, userID uuid.UUID) (*models.ProjectAsset, error) {
	// Create temporary file for screenshot
	tempDir := os.TempDir()
	screenshotFile := filepath.Join(tempDir, fmt.Sprintf("screenshot_%s.png", uuid.New().String()))
	
	// Use headless Chrome to capture screenshot
	cmd := exec.Command("chrome", 
		"--headless", 
		"--no-sandbox", 
		"--disable-gpu", 
		"--window-size=1920,1080",
		"--screenshot="+screenshotFile,
		url)
	
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}
	
	// Open screenshot file
	file, err := os.Open(screenshotFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open screenshot file: %w", err)
	}
	defer file.Close()
	defer os.Remove(screenshotFile)
	
	// Upload as asset
	uploadReq := AssetUploadRequest{
		ProjectID:   projectID,
		AssetType:   "screenshot",
		Title:       fmt.Sprintf("Screenshot of %s", url),
		Description: fmt.Sprintf("Auto-generated screenshot captured on %s", time.Now().Format("2006-01-02 15:04:05")),
		File:        file,
		Filename:    "screenshot.png",
		ContentType: "image/png",
		IsPublic:    false,
		Tags:        []string{"screenshot", "auto-generated"},
		UploadedBy:  userID,
	}
	
	return s.UploadAsset(ctx, uploadReq)
}

// GenerateVideo records a video of a web application
func (s *AssetService) GenerateVideo(ctx context.Context, projectID uuid.UUID, url string, duration int, userID uuid.UUID) (*models.ProjectAsset, error) {
	// Create temporary file for video
	tempDir := os.TempDir()
	videoFile := filepath.Join(tempDir, fmt.Sprintf("video_%s.mp4", uuid.New().String()))
	
	// Use ffmpeg to record video
	cmd := exec.Command("ffmpeg",
		"-f", "x11grab",
		"-video_size", "1920x1080",
		"-i", ":0.0",
		"-t", fmt.Sprintf("%d", duration),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-y",
		videoFile)
	
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to record video: %w", err)
	}
	
	// Open video file
	file, err := os.Open(videoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()
	defer os.Remove(videoFile)
	
	// Upload as asset
	uploadReq := AssetUploadRequest{
		ProjectID:   projectID,
		AssetType:   "video",
		Title:       fmt.Sprintf("Demo video of %s", url),
		Description: fmt.Sprintf("Auto-generated demo video (%d seconds) captured on %s", duration, time.Now().Format("2006-01-02 15:04:05")),
		File:        file,
		Filename:    "demo.mp4",
		ContentType: "video/mp4",
		IsPublic:    false,
		Tags:        []string{"video", "demo", "auto-generated"},
		UploadedBy:  userID,
	}
	
	return s.UploadAsset(ctx, uploadReq)
}

// GetProjectAssets retrieves all assets for a project
func (s *AssetService) GetProjectAssets(ctx context.Context, projectID uuid.UUID) ([]models.ProjectAsset, error) {
	return s.assetRepo.GetByProjectID(ctx, projectID)
}

// GetAssetsByType retrieves assets by type for a project
func (s *AssetService) GetAssetsByType(ctx context.Context, projectID uuid.UUID, assetType string) ([]models.ProjectAsset, error) {
	return s.assetRepo.GetByProjectIDAndType(ctx, projectID, assetType)
}

// DeleteAsset removes an asset and its files
func (s *AssetService) DeleteAsset(ctx context.Context, assetID uuid.UUID) error {
	// Get asset record
	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return fmt.Errorf("asset not found: %w", err)
	}
	
	// Delete files
	if err := os.Remove(asset.FilePath); err != nil {
		s.logger.WithError(err).Warn("Failed to delete asset file")
	}
	
	if asset.Thumbnail != "" {
		if err := os.Remove(asset.Thumbnail); err != nil {
			s.logger.WithError(err).Warn("Failed to delete thumbnail file")
		}
	}
	
	// Delete database record
	if err := s.assetRepo.Delete(ctx, assetID); err != nil {
		return fmt.Errorf("failed to delete asset record: %w", err)
	}
	
	s.logger.WithFields(logrus.Fields{
		"asset_id":   assetID,
		"project_id": asset.ProjectID,
	}).Info("Asset deleted successfully")
	
	return nil
}

// Helper methods

func (s *AssetService) saveFile(src multipart.File, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	
	_, err = io.Copy(out, src)
	return err
}

func (s *AssetService) isImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

func (s *AssetService) generateThumbnail(imagePath, outputDir string) (string, error) {
	// Open image
	file, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	// Decode image
	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}
	
	// Resize to thumbnail
	thumbnail := resize.Thumbnail(300, 300, img, resize.Lanczos3)
	
	// Save thumbnail
	thumbnailPath := filepath.Join(outputDir, "thumb_"+filepath.Base(imagePath))
	thumbnailFile, err := os.Create(thumbnailPath)
	if err != nil {
		return "", err
	}
	defer thumbnailFile.Close()
	
	// Encode thumbnail
	if strings.HasSuffix(imagePath, ".png") {
		err = png.Encode(thumbnailFile, thumbnail)
	} else {
		err = jpeg.Encode(thumbnailFile, thumbnail, &jpeg.Options{Quality: 80})
	}
	
	if err != nil {
		return "", err
	}
	
	return thumbnailPath, nil
}

func (s *AssetService) extractMetadata(filePath, contentType string) string {
	metadata := map[string]interface{}{}
	
	if s.isImage(contentType) {
		// Extract image metadata
		if img, err := s.getImageDimensions(filePath); err == nil {
			metadata["width"] = img.Bounds().Max.X
			metadata["height"] = img.Bounds().Max.Y
		}
	} else if strings.HasPrefix(contentType, "video/") {
		// Extract video metadata using ffprobe
		if duration, err := s.getVideoDuration(filePath); err == nil {
			metadata["duration"] = duration
		}
	}
	
	// Convert to JSON string
	metadataJSON := ""
	if len(metadata) > 0 {
		// Simple JSON serialization
		parts := []string{}
		for key, value := range metadata {
			parts = append(parts, fmt.Sprintf("\"%s\": %v", key, value))
		}
		metadataJSON = "{" + strings.Join(parts, ", ") + "}"
	}
	
	return metadataJSON
}

func (s *AssetService) getImageDimensions(imagePath string) (image.Image, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	img, _, err := image.Decode(file)
	return img, err
}

func (s *AssetService) getVideoDuration(videoPath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	var duration float64
	if _, err := fmt.Sscanf(string(output), "%f", &duration); err != nil {
		return 0, err
	}
	
	return duration, nil
}