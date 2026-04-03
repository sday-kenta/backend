package incident

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/incidenterr"
	"github.com/sday-kenta/backend/internal/repo"
	"github.com/sday-kenta/backend/internal/usererr"
	"github.com/sday-kenta/backend/pkg/mailsender"
	"github.com/sday-kenta/backend/pkg/objectstorage"
)

// UseCase implements usecase.Incident.
type UseCase struct {
	repo         repo.IncidentRepo
	userRepo     repo.UserRepo
	categoryRepo repo.CategoryRepo
	geoRepo      repo.GeoRepo
}

// New creates incident use case.
func New(r repo.IncidentRepo, userRepo repo.UserRepo, categoryRepo repo.CategoryRepo, geoRepo repo.GeoRepo) *UseCase {
	return &UseCase{repo: r, userRepo: userRepo, categoryRepo: categoryRepo, geoRepo: geoRepo}
}

// Create creates a new incident and snapshots reporter data for document generation.
func (uc *UseCase) Create(ctx context.Context, userID int64, input entity.CreateIncidentInput) (entity.Incident, error) {
	user, err := uc.ensureRequesterExists(ctx, userID)
	if err != nil {
		return entity.Incident{}, err
	}

	status, err := normalizeCreateStatus(input.Status)
	if err != nil {
		return entity.Incident{}, err
	}

	preparedLocation, err := prepareLocation(input.City, input.Street, input.House, input.AddressText, input.Latitude, input.Longitude)
	if err != nil {
		return entity.Incident{}, err
	}
	if err = uc.validatePointAllowed(ctx, preparedLocation.Latitude, preparedLocation.Longitude, input.Latitude != nil && input.Longitude != nil); err != nil {
		return entity.Incident{}, err
	}

	category, err := uc.categoryRepo.GetByID(ctx, input.CategoryID)
	if err != nil {
		return entity.Incident{}, incidenterr.ErrCategoryNotFound
	}

	incident := entity.Incident{
		UserID:           userID,
		CategoryID:       input.CategoryID,
		CategoryTitle:    category.Title,
		Title:            strings.TrimSpace(input.Title),
		Description:      strings.TrimSpace(input.Description),
		Status:           status,
		DepartmentName:   strings.TrimSpace(input.DepartmentName),
		City:             preparedLocation.City,
		Street:           preparedLocation.Street,
		House:            preparedLocation.House,
		AddressText:      preparedLocation.AddressText,
		Latitude:         preparedLocation.Latitude,
		Longitude:        preparedLocation.Longitude,
		ReporterFullName: buildReporterFullName(user),
		ReporterEmail:    strings.TrimSpace(user.Email),
		ReporterPhone:    strings.TrimSpace(user.Phone),
		ReporterAddress:  buildReporterAddress(user),
	}
	if incident.DepartmentName == "" {
		incident.DepartmentName = deriveDepartment(category.Title)
	}
	if incident.Status == entity.IncidentStatusPublished {
		now := time.Now().UTC()
		incident.PublishedAt = &now
	}

	if err = uc.repo.Create(ctx, &incident); err != nil {
		return entity.Incident{}, fmt.Errorf("IncidentUseCase - Create - uc.repo.Create: %w", err)
	}

	return uc.repo.GetByID(ctx, incident.ID)
}

// List returns incidents according to filter.
func (uc *UseCase) List(ctx context.Context, filter entity.IncidentFilter) ([]entity.Incident, error) {
	incidents, err := uc.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("IncidentUseCase - List - uc.repo.List: %w", err)
	}

	return incidents, nil
}

// GetByID returns incident by ID.
func (uc *UseCase) GetByID(ctx context.Context, requesterID int64, isAdmin bool, id int64) (entity.Incident, error) {
	if _, err := uc.ensureRequesterExists(ctx, requesterID); err != nil {
		return entity.Incident{}, err
	}

	incident, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return entity.Incident{}, err
	}
	if err = ensureCanView(incident, requesterID, isAdmin); err != nil {
		return entity.Incident{}, err
	}

	return incident, nil
}

// Update updates incident if requester is owner or admin.
func (uc *UseCase) Update(ctx context.Context, requesterID int64, isAdmin bool, id int64, input entity.UpdateIncidentInput) (entity.Incident, error) {
	if _, err := uc.ensureRequesterExists(ctx, requesterID); err != nil {
		return entity.Incident{}, err
	}

	incident, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return entity.Incident{}, err
	}
	if err = ensureCanManage(incident, requesterID, isAdmin); err != nil {
		return entity.Incident{}, err
	}

	if input.CategoryID != nil && *input.CategoryID != incident.CategoryID {
		category, catErr := uc.categoryRepo.GetByID(ctx, *input.CategoryID)
		if catErr != nil {
			return entity.Incident{}, incidenterr.ErrCategoryNotFound
		}
		incident.CategoryID = *input.CategoryID
		incident.CategoryTitle = category.Title
		incident.DepartmentName = deriveDepartment(category.Title)
	}
	if input.Title != nil {
		incident.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		incident.Description = strings.TrimSpace(*input.Description)
	}
	if input.Status != nil {
		status, statusErr := normalizeStatus(*input.Status)
		if statusErr != nil {
			return entity.Incident{}, statusErr
		}
		if status == entity.IncidentStatusPublished && !isAdmin {
			return entity.Incident{}, incidenterr.ErrForbidden
		}
		incident.Status = status
		if status == entity.IncidentStatusPublished {
			now := time.Now().UTC()
			incident.PublishedAt = &now
		} else {
			incident.PublishedAt = nil
		}
	}
	if input.DepartmentName != nil && strings.TrimSpace(*input.DepartmentName) != "" {
		incident.DepartmentName = strings.TrimSpace(*input.DepartmentName)
	}
	if input.City != nil {
		incident.City = strings.TrimSpace(*input.City)
	}
	if input.Street != nil {
		incident.Street = strings.TrimSpace(*input.Street)
	}
	if input.House != nil {
		incident.House = strings.TrimSpace(*input.House)
	}
	if input.AddressText != nil {
		incident.AddressText = strings.TrimSpace(*input.AddressText)
	}
	if input.Latitude != nil {
		incident.Latitude = *input.Latitude
	}
	if input.Longitude != nil {
		incident.Longitude = *input.Longitude
	}

	preparedLocation, err := prepareLocation(
		incident.City,
		incident.Street,
		incident.House,
		incident.AddressText,
		&incident.Latitude,
		&incident.Longitude,
	)
	if err != nil {
		return entity.Incident{}, err
	}
	hasCoordinates := input.Latitude != nil || input.Longitude != nil || incident.Latitude != 0 || incident.Longitude != 0
	if err = uc.validatePointAllowed(ctx, preparedLocation.Latitude, preparedLocation.Longitude, hasCoordinates); err != nil {
		return entity.Incident{}, err
	}
	incident.City = preparedLocation.City
	incident.Street = preparedLocation.Street
	incident.House = preparedLocation.House
	incident.AddressText = preparedLocation.AddressText
	incident.Latitude = preparedLocation.Latitude
	incident.Longitude = preparedLocation.Longitude

	if err = uc.repo.Update(ctx, &incident); err != nil {
		return entity.Incident{}, fmt.Errorf("IncidentUseCase - Update - uc.repo.Update: %w", err)
	}

	return uc.repo.GetByID(ctx, id)
}

// Delete removes incident and returns photo metadata for best-effort storage cleanup.
func (uc *UseCase) Delete(ctx context.Context, requesterID int64, isAdmin bool, id int64) ([]entity.IncidentPhoto, error) {
	if _, err := uc.ensureRequesterExists(ctx, requesterID); err != nil {
		return nil, err
	}

	incident, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = ensureCanManage(incident, requesterID, isAdmin); err != nil {
		return nil, err
	}

	photos := append([]entity.IncidentPhoto(nil), incident.Photos...)
	if err = uc.repo.Delete(ctx, id); err != nil {
		return nil, fmt.Errorf("IncidentUseCase - Delete - uc.repo.Delete: %w", err)
	}

	return photos, nil
}

// CreatePhoto adds a photo to incident.
func (uc *UseCase) CreatePhoto(ctx context.Context, requesterID int64, isAdmin bool, incidentID int64, photo entity.IncidentPhoto) (entity.IncidentPhoto, error) {
	if _, err := uc.ensureRequesterExists(ctx, requesterID); err != nil {
		return entity.IncidentPhoto{}, err
	}

	incident, err := uc.repo.GetByID(ctx, incidentID)
	if err != nil {
		return entity.IncidentPhoto{}, err
	}
	if err = ensureCanManage(incident, requesterID, isAdmin); err != nil {
		return entity.IncidentPhoto{}, err
	}

	photo.IncidentID = incidentID
	photo.SortOrder = len(incident.Photos)
	if err = uc.repo.CreatePhoto(ctx, &photo); err != nil {
		return entity.IncidentPhoto{}, fmt.Errorf("IncidentUseCase - CreatePhoto - uc.repo.CreatePhoto: %w", err)
	}

	return photo, nil
}

// DeletePhoto deletes a photo from incident and returns metadata for storage cleanup.
func (uc *UseCase) DeletePhoto(ctx context.Context, requesterID int64, isAdmin bool, incidentID, photoID int64) (entity.IncidentPhoto, error) {
	if _, err := uc.ensureRequesterExists(ctx, requesterID); err != nil {
		return entity.IncidentPhoto{}, err
	}

	incident, err := uc.repo.GetByID(ctx, incidentID)
	if err != nil {
		return entity.IncidentPhoto{}, err
	}
	if err = ensureCanManage(incident, requesterID, isAdmin); err != nil {
		return entity.IncidentPhoto{}, err
	}

	photo, err := uc.repo.DeletePhoto(ctx, incidentID, photoID)
	if err != nil {
		return entity.IncidentPhoto{}, err
	}

	return photo, nil
}

// RenderDocument builds an HTML document suitable for download or print.
func (uc *UseCase) RenderDocument(ctx context.Context, requesterID int64, isAdmin bool, incidentID int64) (entity.IncidentDocument, error) {
	if _, err := uc.ensureRequesterExists(ctx, requesterID); err != nil {
		return entity.IncidentDocument{}, err
	}

	incident, err := uc.repo.GetByID(ctx, incidentID)
	if err != nil {
		return entity.IncidentDocument{}, err
	}
	if err = ensureCanManage(incident, requesterID, isAdmin); err != nil {
		return entity.IncidentDocument{}, err
	}

	html, err := renderIncidentHTML(buildIncidentDocumentView(incident))
	if err != nil {
		return entity.IncidentDocument{}, err
	}

	return entity.IncidentDocument{
		FileName:    fmt.Sprintf("incident-%d.html", incident.ID),
		ContentType: "text/html; charset=utf-8",
		Subject:     fmt.Sprintf("Обращение по инциденту #%d", incident.ID),
		BodyHTML:    html,
	}, nil
}

// SendDocumentByEmail sends the generated document to the given email or to the reporter email by default.
func (uc *UseCase) SendDocumentByEmail(ctx context.Context, requesterID int64, isAdmin bool, incidentID int64, email string) error {
	incident, err := uc.repo.GetByID(ctx, incidentID)
	if err != nil {
		return err
	}
	if err = ensureCanManage(incident, requesterID, isAdmin); err != nil {
		return err
	}

	emailDoc, err := uc.buildEmailDocument(ctx, incident)
	if err != nil {
		return fmt.Errorf("IncidentUseCase - SendDocumentByEmail - buildEmailDocument: %w", err)
	}
	html, err := renderIncidentHTML(emailDoc.View)
	if err != nil {
		return err
	}
	doc := entity.IncidentDocument{
		FileName:          fmt.Sprintf("incident-%d.html", incident.ID),
		ContentType:       "text/html; charset=utf-8",
		Subject:           fmt.Sprintf("Обращение по инциденту #%d", incident.ID),
		BodyHTML:          html,
		InlineAttachments: emailDoc.InlineAttachments,
	}

	to := strings.TrimSpace(email)
	if to == "" {
		to = strings.TrimSpace(incident.ReporterEmail)
	}
	if to == "" {
		return incidenterr.ErrDocumentEmailEmpty
	}

	if err = mailsender.SendMailWithAttachment(
		doc.Subject,
		doc.BodyHTML,
		[]string{to},
		doc.FileName,
		[]byte(doc.BodyHTML),
		"text/html; charset=utf-8",
		doc.InlineAttachments,
	); err != nil {
		return fmt.Errorf("IncidentUseCase - SendDocumentByEmail - mailsender.SendMailWithAttachment: %w", err)
	}

	return nil
}

func (uc *UseCase) buildEmailDocument(ctx context.Context, incident entity.Incident) (emailDocumentBuildResult, error) {
	result := emailDocumentBuildResult{View: buildIncidentDocumentView(incident)}
	if len(incident.Photos) == 0 {
		return result, nil
	}

	storage, err := objectstorage.NewFromEnv(ctx)
	if err != nil {
		return result, nil
	}

	for idx, photo := range incident.Photos {
		if strings.TrimSpace(photo.FileKey) == "" {
			continue
		}

		content, downloadErr := storage.Download(ctx, photo.FileKey)
		if downloadErr != nil {
			continue
		}

		contentID := fmt.Sprintf("incident-photo-%d-%d", incident.ID, idx+1)
		result.View.Photos[idx].Src = template.URL("cid:" + contentID)
		result.InlineAttachments = append(result.InlineAttachments, entity.InlineAttachment{
			ContentID:   contentID,
			FileName:    fmt.Sprintf("incident-%d-photo-%d", incident.ID, idx+1),
			ContentType: normalizePhotoContentType(photo.ContentType),
			Body:        content,
		})
	}

	return result, nil
}

func (uc *UseCase) ensureRequesterExists(ctx context.Context, requesterID int64) (entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, requesterID)
	if err != nil {
		if errors.Is(err, usererr.ErrNotFound) {
			return entity.User{}, incidenterr.ErrRequesterNotFound
		}
		return entity.User{}, fmt.Errorf("IncidentUseCase - ensureRequesterExists - uc.userRepo.GetByID: %w", err)
	}

	return user, nil
}

func (uc *UseCase) validatePointAllowed(ctx context.Context, lat, lon float64, hasCoordinates bool) error {
	if !hasCoordinates || uc.geoRepo == nil {
		return nil
	}

	_, err := uc.geoRepo.FindContainingZone(ctx, lat, lon)
	if err != nil {
		if errors.Is(err, entity.ErrZoneNotFound) {
			return entity.ErrOutOfAllowedZone
		}
		return fmt.Errorf("IncidentUseCase - validatePointAllowed - uc.geoRepo.FindContainingZone: %w", err)
	}

	return nil
}

type preparedLocation struct {
	City        string
	Street      string
	House       string
	AddressText string
	Latitude    float64
	Longitude   float64
}

type incidentDocumentView struct {
	ID               int64
	DepartmentName   string
	CategoryTitle    string
	Title            string
	AddressText      string
	Latitude         float64
	Longitude        float64
	Description      string
	ReporterFullName string
	ReporterEmail    string
	ReporterPhone    string
	ReporterAddress  string
	Photos           []incidentDocumentPhotoView
}

type incidentDocumentPhotoView struct {
	Src template.URL
}

type emailDocumentBuildResult struct {
	View              incidentDocumentView
	InlineAttachments []entity.InlineAttachment
}

func prepareLocation(city, street, house, addressText string, latitude, longitude *float64) (preparedLocation, error) {
	city = strings.TrimSpace(city)
	street = strings.TrimSpace(street)
	house = strings.TrimSpace(house)
	addressText = strings.TrimSpace(addressText)

	if latitude == nil || longitude == nil {
		if latitude != nil || longitude != nil {
			return preparedLocation{}, incidenterr.ErrInvalidCoordinates
		}
		if addressText == "" && city == "" && street == "" && house == "" {
			return preparedLocation{}, incidenterr.ErrLocationRequired
		}
		return preparedLocation{City: city, Street: street, House: house, AddressText: coalesceAddress(addressText, city, street, house)}, nil
	}

	if addressText == "" {
		addressText = coalesceAddress(addressText, city, street, house)
	}
	if addressText == "" {
		return preparedLocation{}, incidenterr.ErrLocationRequired
	}

	return preparedLocation{
		City:        city,
		Street:      street,
		House:       house,
		AddressText: addressText,
		Latitude:    *latitude,
		Longitude:   *longitude,
	}, nil
}

func normalizeStatus(status string) (string, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		status = entity.IncidentStatusReview
	}
	switch status {
	case entity.IncidentStatusDraft, entity.IncidentStatusReview, entity.IncidentStatusPublished:
		return status, nil
	default:
		return "", incidenterr.ErrInvalidStatus
	}
}

func normalizeCreateStatus(status string) (string, error) {
	status, err := normalizeStatus(status)
	if err != nil {
		return "", err
	}
	if status == entity.IncidentStatusPublished {
		return entity.IncidentStatusReview, nil
	}
	return status, nil
}

func ensureCanManage(incident entity.Incident, requesterID int64, isAdmin bool) error {
	if isAdmin {
		return nil
	}
	if requesterID == 0 || incident.UserID != requesterID {
		return incidenterr.ErrForbidden
	}
	return nil
}

func ensureCanView(incident entity.Incident, requesterID int64, isAdmin bool) error {
	return ensureCanManage(incident, requesterID, isAdmin)
}

func deriveDepartment(categoryTitle string) string {
	title := strings.ToLower(strings.TrimSpace(categoryTitle))
	switch {
	case strings.Contains(title, "парков"):
		return "ГИБДД"
	case strings.Contains(title, "просроч"), strings.Contains(title, "товар"):
		return "Роспотребнадзор"
	default:
		return "Профильное ведомство"
	}
}

func buildReporterFullName(user entity.User) string {
	parts := []string{strings.TrimSpace(user.LastName), strings.TrimSpace(user.FirstName), strings.TrimSpace(user.MiddleName)}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " ")
}

func buildReporterAddress(user entity.User) string {
	parts := []string{strings.TrimSpace(user.City), strings.TrimSpace(user.Street), strings.TrimSpace(user.House)}
	if apartment := strings.TrimSpace(user.Apartment); apartment != "" {
		parts = append(parts, "кв. "+apartment)
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, ", ")
}

func coalesceAddress(addressText string, parts ...string) string {
	if strings.TrimSpace(addressText) != "" {
		return strings.TrimSpace(addressText)
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, ", ")
}

func buildIncidentDocumentView(incident entity.Incident) incidentDocumentView {
	view := incidentDocumentView{
		ID:               incident.ID,
		DepartmentName:   incident.DepartmentName,
		CategoryTitle:    incident.CategoryTitle,
		Title:            incident.Title,
		AddressText:      incident.AddressText,
		Latitude:         incident.Latitude,
		Longitude:        incident.Longitude,
		Description:      incident.Description,
		ReporterFullName: incident.ReporterFullName,
		ReporterEmail:    incident.ReporterEmail,
		ReporterPhone:    incident.ReporterPhone,
		ReporterAddress:  incident.ReporterAddress,
		Photos:           make([]incidentDocumentPhotoView, 0, len(incident.Photos)),
	}

	for _, photo := range incident.Photos {
		view.Photos = append(view.Photos, incidentDocumentPhotoView{Src: template.URL(photo.FileURL)})
	}

	return view
}

func normalizePhotoContentType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

func renderIncidentHTML(doc incidentDocumentView) (string, error) {
	const tpl = `<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <title>Обращение по инциденту #{{.ID}}</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 32px; line-height: 1.45; color: #111; }
    h1, h2 { margin-bottom: 12px; }
    .meta { margin-bottom: 24px; }
    .label { font-weight: bold; }
    .photo { margin: 16px 0; page-break-inside: avoid; }
    .photo img {
      display: block;
      width: 100%;
      max-width: 720px;
      height: 320px;
      object-fit: contain;
      object-position: center;
      border: 1px solid #d9d9d9;
      border-radius: 8px;
      background: #f7f7f7;
    }
    @media print { body { margin: 12mm; } }
  </style>
</head>
<body>
  <h1>Обращение в {{.DepartmentName}}</h1>
  <div class="meta">
    <div><span class="label">Инцидент:</span> #{{.ID}}</div>
    <div><span class="label">Рубрика:</span> {{.CategoryTitle}}</div>
    <div><span class="label">Тема:</span> {{.Title}}</div>
    <div><span class="label">Адрес:</span> {{.AddressText}}</div>
    <div><span class="label">Координаты:</span> {{printf "%.6f" .Latitude}}, {{printf "%.6f" .Longitude}}</div>
  </div>

  <h2>Описание</h2>
  <p>{{.Description}}</p>

  <h2>Заявитель</h2>
  <div><span class="label">ФИО:</span> {{.ReporterFullName}}</div>
  <div><span class="label">E-mail:</span> {{.ReporterEmail}}</div>
  <div><span class="label">Телефон:</span> {{.ReporterPhone}}</div>
  <div><span class="label">Адрес:</span> {{.ReporterAddress}}</div>

  {{if .Photos}}
  <h2>Фотографии</h2>
  {{range .Photos}}
    <div class="photo"><img src="{{.Src}}" alt="Фотография инцидента"></div>
  {{end}}
  {{end}}

  <p style="margin-top: 32px;">Документ сформирован автоматически сервисом «Сознательный гражданин».</p>
</body>
</html>`

	t, err := template.New("incident_document").Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("IncidentUseCase - renderIncidentHTML - template.Parse: %w", err)
	}

	var buf bytes.Buffer
	if err = t.Execute(&buf, doc); err != nil {
		return "", fmt.Errorf("IncidentUseCase - renderIncidentHTML - template.Execute: %w", err)
	}

	return buf.String(), nil
}
